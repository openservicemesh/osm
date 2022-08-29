package sds

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

type sdsBuilder struct {
	proxy *envoy.Proxy

	// Service certificate for this proxy
	serviceCert *certificate.Certificate

	// The DiscoveryRequest contains the certs to generate secrets for
	// requestedCerts is expected to be a list of either "service-cert:namespace/service" or "root-cert:namespace/service"
	requestedCerts []string

	trustDomain string

	serviceIdentitiesForOutboundServices map[service.MeshService][]identity.ServiceIdentity
}

// NewBuilder returns a new sdsBuilder
func NewBuilder() *sdsBuilder { //nolint: revive // unexported-return
	return &sdsBuilder{}
}

func (b *sdsBuilder) SetProxy(proxy *envoy.Proxy) *sdsBuilder {
	b.proxy = proxy
	return b
}

func (b *sdsBuilder) SetProxyCert(cert *certificate.Certificate) *sdsBuilder {
	b.serviceCert = cert
	return b
}

func (b *sdsBuilder) SetRequestedCerts(requestedCerts []string) *sdsBuilder {
	b.requestedCerts = requestedCerts
	return b
}

func (b *sdsBuilder) SetTrustDomain(trustDomain string) *sdsBuilder {
	b.trustDomain = trustDomain
	return b
}

func (b *sdsBuilder) SetServiceIdentitiesForService(serviceIdentitiesForServices map[service.MeshService][]identity.ServiceIdentity) *sdsBuilder {
	b.serviceIdentitiesForOutboundServices = serviceIdentitiesForServices
	return b
}

// Build generates SDS Secret Resources based on requested certs in the DiscoveryRequest
func (b *sdsBuilder) Build() []*xds_auth.Secret {
	log.Info().Str("proxy", b.proxy.String()).Msgf("Creating SDS response for request for resources %v", b.requestedCerts)
	var sdsResources = make([]*xds_auth.Secret, 0, len(b.requestedCerts))
	// requestedCerts is expected to be a list of either of the following:
	// - "service-cert:namespace/service-account"
	// - "root-cert-for-mtls-outbound:namespace/service"
	// - "root-cert-for-mtls-inbound:namespace/service-account"
	// The Envoy makes a request for a list of resources (aka certificates), which we will send as a response to the SDS request.
	for _, requestedCertificate := range b.requestedCerts {
		sdsCert, err := secrets.UnmarshalSDSCert(requestedCertificate)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUnmarshallingSDSCert)).
				Msgf("Invalid resource kind requested: %q", requestedCertificate)
			continue
		}

		log.Debug().Str("proxy", b.proxy.String()).Msgf("Proxy requested cert %s", requestedCertificate)

		switch sdsCert.CertType {
		// A service certificate is requested
		case secrets.ServiceCertType:
			envoySecret, err := b.buildServiceCertSecret(requestedCertificate)
			if err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingServiceCertSecret)).
					Str("proxy", b.proxy.String()).Msgf("Error getting service cert %s for proxy", requestedCertificate)
				continue
			}
			sdsResources = append(sdsResources, envoySecret)

		// A root certificate used to validate a service certificate is requested
		case secrets.RootCertTypeForMTLSInbound, secrets.RootCertTypeForMTLSOutbound:
			envoySecret, err := b.buildRootCertSecret(*sdsCert)
			if err != nil {
				log.Error().Err(err).Str("proxy", b.proxy.String()).Msgf("Error getting root cert %s for proxy", requestedCertificate)
				continue
			}
			sdsResources = append(sdsResources, envoySecret)

		default:
			log.Error().Str("proxy", b.proxy.String()).Msgf("Unexpected certificate type %s requested by proxy", requestedCertificate)
		}
	}
	return sdsResources
}

// buildServiceCertSecret creates the struct with certificates for the service, which the
// connected Envoy proxy belongs to.
func (b *sdsBuilder) buildServiceCertSecret(name string) (*xds_auth.Secret, error) {
	secret := &xds_auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name in the Envoy yaml config
		Name: name,
		Type: &xds_auth.Secret_TlsCertificate{
			TlsCertificate: &xds_auth.TlsCertificate{
				CertificateChain: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: b.serviceCert.GetCertificateChain(),
					},
				},
				PrivateKey: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: b.serviceCert.GetPrivateKey(),
					},
				},
			},
		},
	}
	return secret, nil
}

func (b *sdsBuilder) buildRootCertSecret(sdscert secrets.SDSCert) (*xds_auth.Secret, error) {
	secret := &xds_auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
		Name: sdscert.String(),
		Type: &xds_auth.Secret_ValidationContext{
			ValidationContext: &xds_auth.CertificateValidationContext{
				TrustedCa: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: b.serviceCert.GetTrustedCAs(),
					},
				},
			},
		},
	}

	// SAN validation should not be performed by the root validation certificate used by the upstream server
	// to validate a downstream client. This is because of the following:
	// 1. SAN validation is already performed by the RBAC filter on the inbound listener's filter chain (using
	//    network RBAC filter) and each HTTP route in the inbound route ocnfiguration (using HTTP RBAC per route).
	// 2. The same root validation certificate is used to validate both in-mesh and ingress downstreams.
	// For these reasons, we only perform SAN validation of peer certificates on downstream clients (ie. outbound SAN
	// validation).
	if sdscert.CertType == secrets.RootCertTypeForMTLSInbound {
		return secret, nil
	}

	// For the outbound certificate validation context, the SANs needs to match the list of service identities
	// corresponding to the upstream service. This means, if the sdscert.Name points to service 'X',
	// the SANs for this certificate should correspond to the service identities of 'X'.
	meshSvc, err := sdscert.GetMeshService()
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingMeshService)).
			Msgf("Error unmarshalling upstream service for outbound cert %s", sdscert)
		return nil, err
	}
	svcIdentitiesInCertRequest := b.serviceIdentitiesForOutboundServices[*meshSvc]

	secret.GetValidationContext().MatchTypedSubjectAltNames = getSubjectAltNamesFromSvcIdentities(svcIdentitiesInCertRequest, b.trustDomain)
	return secret, nil
}
