package sds

import (
	xds_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/envoy/secrets"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, certManager *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	log.Info().Str("proxy", proxy.String()).Msg("Composing SDS Discovery Response")

	// sdsBuilder: builds the Secret Discovery Response
	builder := NewBuilder().SetRequestedCerts(request.ResourceNames).SetProxy(proxy).SetTrustDomain(certManager.GetTrustDomain())

	// Issue a service certificate for this proxy
	cert, err := certManager.IssueCertificate(proxy.Identity.String(), certificate.Service)
	if err != nil {
		log.Error().Err(err).Str("proxy", proxy.String()).Msgf("Error issuing a certificate for proxy")
		return nil, err
	}
	builder.SetProxyCert(cert)

	// Set service identities for services in requests
	log.Trace().Msgf("Getting Service Identities for services in request for resources %v", builder.requestedCerts)
	serviceIdentitiesForServices := getServiceIdentitiesForOutboundServices(builder.requestedCerts, meshCatalog)
	builder.SetServiceIdentitiesForService(serviceIdentitiesForServices)

	// Get SDS Secret Resources based on requested certs in the DiscoveryRequest
	var sdsResources = make([]types.Resource, 0, len(builder.requestedCerts))
	for _, envoyProto := range builder.Build() {
		sdsResources = append(sdsResources, envoyProto)
	}
	return sdsResources, nil
}

func getServiceIdentitiesForOutboundServices(requestedCerts []string, meshCatalog catalog.MeshCataloger) map[service.MeshService][]identity.ServiceIdentity {
	serviceIdentitiesForOutboundServices := make(map[service.MeshService][]identity.ServiceIdentity)
	for _, requestedCertificate := range requestedCerts {
		sdsCert, err := secrets.UnmarshalSDSCert(requestedCertificate)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUnmarshallingSDSCert)).
				Msgf("Invalid resource kind requested: %q", requestedCertificate)
			continue
		}

		if sdsCert.CertType == secrets.RootCertTypeForMTLSOutbound {
			// A root certificate requiring matching SANS
			meshSvc, err := sdsCert.GetMeshService()
			if err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingMeshService)).
					Msgf("Error unmarshalling mesh service for cert %s", sdsCert)
				continue
			}

			svcIdentities := meshCatalog.ListServiceIdentitiesForService(*meshSvc)
			if svcIdentities == nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingServiceIdentitiesForService)).
					Msgf("Error listing service identities for mesh service %s", *meshSvc)
				continue
			}
			serviceIdentitiesForOutboundServices[*meshSvc] = svcIdentities
		}
	}
	return serviceIdentitiesForOutboundServices
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func getSubjectAltNamesFromSvcIdentities(serviceIdentities []identity.ServiceIdentity, trustDomain string) []*xds_auth.SubjectAltNameMatcher {
	var matchSANs []*xds_auth.SubjectAltNameMatcher

	for _, si := range serviceIdentities {
		match := xds_auth.SubjectAltNameMatcher{
			SanType: xds_auth.SubjectAltNameMatcher_DNS,
			Matcher: &xds_matcher.StringMatcher{
				MatchPattern: &xds_matcher.StringMatcher_Exact{
					Exact: si.AsPrincipal(trustDomain),
				},
			},
		}
		matchSANs = append(matchSANs, &match)
	}

	return matchSANs
}
