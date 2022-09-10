package generator

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/generator/eds"
	"github.com/openservicemesh/osm/pkg/service"
)

// generateCDS creates a new Endpoint Discovery Response.
func (g *EnvoyConfigGenerator) generateEDS(ctx context.Context, proxy *envoy.Proxy) ([]types.Resource, error) {
	meshSvcEndpoints := make(map[service.MeshService][]endpoint.Endpoint)
	builder := eds.NewEndpointsBuilder()

	for _, dstSvc := range g.catalog.ListOutboundServicesForIdentity(proxy.Identity) {
		builder.AddEndpoints(
			dstSvc,
			g.catalog.ListAllowedUpstreamEndpointsForService(proxy.Identity, dstSvc),
		)

		log.Trace().Msgf("Allowed outbound service endpoints for proxy with identity %s: %v", proxy.Identity, meshSvcEndpoints)
	}

	return builder.Build(), nil
}
