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
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("envoy/sds")
)

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, certManager *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	log.Info().Str("proxy", proxy.String()).Msg("Composing SDS Discovery Response")

	// OSM currently relies on kubernetes ServiceAccount for service identity
	proxyIdentity, err := envoy.GetServiceIdentityFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingServiceIdentity)).
			Str("proxy", proxy.String()).Msg("Error retrieving ServiceAccount for proxy")
		return nil, err
	}

	var sdsResources []types.Resource

	// The DiscoveryRequest contains the requested certs
	requestedCerts := request.ResourceNames

	log.Info().Str("proxy", proxy.String()).Msgf("Creating SDS response for request for resources %v", requestedCerts)

	// 1. Issue a service certificate for this proxy
	cert, err := certManager.IssueCertificate(certificate.CommonName(proxyIdentity), cfg.GetServiceCertValidityPeriod())
	if err != nil {
		log.Error().Err(err).Str("proxy", proxy.String()).Msgf("Error issuing a certificate for proxy")
		return nil, err
	}

	// 2. Create SDS secret resources based on the requested certs in the DiscoveryRequest
	// request.ResourceNames is expected to be a list of either "service-cert:namespace/service" or "root-cert:namespace/service"
	for _, envoyProto := range getSDSSecrets(meshCatalog, cert, requestedCerts, proxy) {
		sdsResources = append(sdsResources, envoyProto)
	}

	return sdsResources, nil
}

func getSDSSecrets(mc catalog.MeshCataloger, cert *certificate.Certificate, requestedCerts []string, proxy *envoy.Proxy) (certs []*xds_auth.Secret) {
	// requestedCerts is expected to be a list of either of the following:
	// - "service-cert:service-identiy", ie: "service-cert:name.namespace.cluster.local"
	// - "root-cert-for-mtls-inbound:service-identity"
	// - "root-cert-for-mtls-outbound:namespace/service"

	// The Envoy makes a request for a list of resources (aka certificates), which we will send as a response to the SDS request.
	for _, requestedCertificate := range requestedCerts {
		sdsCert, err := secrets.UnmarshalSDSCert(requestedCertificate)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUnmarshallingSDSCert)).
				Msgf("Invalid resource kind requested: %q", requestedCertificate)
			continue
		}

		log.Debug().Str("proxy", proxy.String()).Msgf("Proxy requested cert %s", requestedCertificate)

		switch v := sdsCert.(type) {
		case *secrets.SDSServiceCert:
			// A service certificate is requested
			certs = append(certs, getServiceCertSecret(cert, requestedCertificate))
		case *secrets.SDSInboundRootCert:
			// SAN validation should not be performed by the root validation certificate used by the upstream server
			// to validate a downstream client. This is because of the following:
			// 1. SAN validation is already performed by the RBAC filter on the inbound listener's filter chain (using
			//    network RBAC filter) and each HTTP route in the inbound route ocnfiguration (using HTTP RBAC per route).
			// 2. The same root validation certificate is used to validate both in-mesh and ingress downstreams.
			//
			// For these reasons, we only perform SAN validation of peer certificates on downstream clients (ie. outbound SAN
			// validation).
			certs = append(certs, getRootCert(cert, sdsCert))
		case *secrets.SDSOutboundRootCert:
			// A root certificate used to validate a service certificate is requested
			secret := getRootCert(cert, sdsCert)

			// For the outbound certificate validation context, the SANs needs to match the list of service identities
			// corresponding to the upstream service. This means, if the sdscert.Name points to service 'X',
			// the SANs for this certificate should correspond to the service identities of 'X'.
			svcIdentitiesInCertRequest := mc.ListServiceIdentitiesForService(v.GetMeshService())
			secret.GetValidationContext().MatchSubjectAltNames = getSubjectAltNamesFromSvcIdentities(svcIdentitiesInCertRequest)
			certs = append(certs, secret)
		default:
			log.Error().Str("proxy", proxy.String()).Msgf("Unexpected certificate type %s requested by proxy", requestedCertificate)
		}
	}

	return certs
}

// getServiceCertSecret creates the struct with certificates for the service, which the
// connected Envoy proxy belongs to.
func getServiceCertSecret(cert *certificate.Certificate, name string) *xds_auth.Secret {
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
	return secret
}

func getRootCert(cert *certificate.Certificate, sdscert secrets.SDSCert) *xds_auth.Secret {
	return &xds_auth.Secret{
		// The Name field must match the tls_context.common_tls_context.tls_certificate_sds_secret_configs.name
		Name: sdscert.String(),
		Type: &xds_auth.Secret_ValidationContext{
			ValidationContext: &xds_auth.CertificateValidationContext{
				TrustedCa: &xds_core.DataSource{
					Specifier: &xds_core.DataSource_InlineBytes{
						InlineBytes: cert.GetIssuingCA(),
					},
				},
			},
		},
	}
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func getSubjectAltNamesFromSvcIdentities(serviceIdentities []identity.ServiceIdentity) []*xds_matcher.StringMatcher {
	var matchSANs []*xds_matcher.StringMatcher

	for _, si := range serviceIdentities {
		match := xds_matcher.StringMatcher{
			MatchPattern: &xds_matcher.StringMatcher_Exact{
				Exact: si.String(),
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
