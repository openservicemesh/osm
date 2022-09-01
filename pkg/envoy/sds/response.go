package sds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Secrets Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, certManager *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	log.Info().Str("proxy", proxy.String()).Msg("Composing SDS Discovery Response")

	// sdsBuilder: builds the Secret Discovery Response
	builder := NewBuilder().SetProxy(proxy).SetTrustDomain(certManager.GetTrustDomain())
	if request != nil {
		builder.SetRequestedCerts(request.ResourceNames)
	}

	// Issue a service certificate for this proxy
	cert, err := certManager.IssueCertificate(proxy.Identity.String(), certificate.Service)
	if err != nil {
		log.Error().Err(err).Str("proxy", proxy.String()).Msgf("Error issuing a certificate for proxy")
		return nil, err
	}
	builder.SetProxyCert(cert)

	// Set service identities for services in requests
	log.Trace().Msgf("Getting Service Identities for services in request for resources %v", builder.requestedCerts)
	serviceIdentitiesForOutboundServices := make(map[service.MeshService][]identity.ServiceIdentity)

	for _, svc := range meshCatalog.ListOutboundServicesForIdentity(proxy.Identity) {
		serviceIdentitiesForOutboundServices[svc] = meshCatalog.ListServiceIdentitiesForService(svc)
	}

	builder.SetServiceIdentitiesForService(serviceIdentitiesForOutboundServices)

	// Get SDS Secret Resources based on requested certs in the DiscoveryRequest
	var sdsResources = make([]types.Resource, 0, len(builder.requestedCerts))
	for _, envoyProto := range builder.Build() {
		sdsResources = append(sdsResources, envoyProto)
	}
	return sdsResources, nil
}
