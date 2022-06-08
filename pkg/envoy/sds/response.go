package sds

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
)

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, _ configurator.Configurator, certManager *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	log.Info().Str("proxy", proxy.String()).Msg("Composing SDS Discovery Response")

	// OSM currently relies on kubernetes ServiceAccount for service identity
	s := &sdsImpl{
		meshCatalog:     meshCatalog,
		certManager:     certManager,
		serviceIdentity: proxy.Identity,
		TrustDomain:     certManager.GetTrustDomain(),
	}

	var sdsResources []types.Resource

	// The DiscoveryRequest contains the requested certs
	requestedCerts := request.ResourceNames

	log.Info().Str("proxy", proxy.String()).Msgf("Creating SDS response for request for resources %v", requestedCerts)

	// 1. Issue a service certificate for this proxy
	cert, err := certManager.IssueCertificate(s.serviceIdentity.String(), certificate.Service)
	if err != nil {
		log.Error().Err(err).Str("proxy", proxy.String()).Msgf("Error issuing a certificate for proxy")
		return nil, err
	}

	// 2. Create SDS secret resources based on the requested certs in the DiscoveryRequest
	// request.ResourceNames is expected to be a list of either "service-cert:namespace/service" or "root-cert:namespace/service"
	for _, envoyProto := range s.getSDSSecrets(cert, requestedCerts, proxy) {
		sdsResources = append(sdsResources, envoyProto)
	}

	return sdsResources, nil
}

func (s *sdsImpl) getSDSSecrets(cert *certificate.Certificate, requestedCerts []string, proxy *envoy.Proxy) (certs []*xds_auth.Secret) {
	// requestedCerts is expected to be a list of either of the following:
	// - "service-cert:namespace/service-account"
	// - "root-cert-for-mtls-outbound:namespace/service"
	// - "root-cert-for-mtls-inbound:namespace/service-account"

	// The Envoy makes a request for a list of resources (aka certificates), which we will send as a response to the SDS request.
	for _, requestedCertificate := range requestedCerts {
		sdsCert, err := secrets.UnmarshalSDSCert(requestedCertificate)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUnmarshallingSDSCert)).
				Msgf("Invalid resource kind requested: %q", requestedCertificate)
			continue
		}

		log.Debug().Str("proxy", proxy.String()).Msgf("Proxy requested cert %s", requestedCertificate)

		switch sdsCert.CertType {
		// A service certificate is requested
		case secrets.ServiceCertType:
			envoySecret, err := getServiceCertSecret(cert, requestedCertificate)
			if err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingServiceCertSecret)).
					Str("proxy", proxy.String()).Msgf("Error getting service cert %s for proxy", requestedCertificate)
				continue
			}
			certs = append(certs, envoySecret)

		// A root certificate used to validate a service certificate is requested
		case secrets.RootCertTypeForMTLSInbound, secrets.RootCertTypeForMTLSOutbound:
			envoySecret, err := s.getRootCert(cert, *sdsCert)
			if err != nil {
				log.Error().Err(err).Str("proxy", proxy.String()).Msgf("Error getting root cert %s for proxy", requestedCertificate)
				continue
			}
			certs = append(certs, envoySecret)

		default:
			log.Error().Str("proxy", proxy.String()).Msgf("Unexpected certificate type %s requested by proxy", requestedCertificate)
		}
	}

	return certs
}

// getServiceCertSecret creates the struct with certificates for the service, which the
// connected Envoy proxy belongs to.
func getServiceCertSecret(cert *certificate.Certificate, name string) (*xds_auth.Secret, error) {
	secret := &xds_auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name in the Envoy yaml config
		Name: name,
		Type: &xds_auth.Secret_TlsCertificate{
			TlsCertificate: &xds_auth.TlsCertificate{
				CertificateChain: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: cert.GetCertificateChain(),
					},
				},
				PrivateKey: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: cert.GetPrivateKey(),
					},
				},
			},
		},
	}
	return secret, nil
}

func (s *sdsImpl) getRootCert(cert *certificate.Certificate, sdscert secrets.SDSCert) (*xds_auth.Secret, error) {
	secret := &xds_auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
		Name: sdscert.String(),
		Type: &xds_auth.Secret_ValidationContext{
			ValidationContext: &xds_auth.CertificateValidationContext{
				TrustedCa: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: cert.GetTrustedCAs(),
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
	//
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
	svcIdentitiesInCertRequest := s.meshCatalog.ListServiceIdentitiesForService(*meshSvc)

	secret.GetValidationContext().MatchSubjectAltNames = getSubjectAltNamesFromSvcIdentities(svcIdentitiesInCertRequest, s.TrustDomain)
	return secret, nil
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func getSubjectAltNamesFromSvcIdentities(serviceIdentities []identity.ServiceIdentity, trustDomain string) []*xds_matcher.StringMatcher {
	var matchSANs []*xds_matcher.StringMatcher

	for _, si := range serviceIdentities {
		match := xds_matcher.StringMatcher{
			MatchPattern: &xds_matcher.StringMatcher_Exact{
				Exact: si.AsPrincipal(trustDomain),
			},
		}
		matchSANs = append(matchSANs, &match)
	}

	return matchSANs
}

func subjectAltNamesToStr(sanMatchList []*xds_matcher.StringMatcher) []string {
	var sanStr []string

	for _, sanMatcher := range sanMatchList {
		sanStr = append(sanStr, sanMatcher.GetExact())
	}
	return sanStr
}
