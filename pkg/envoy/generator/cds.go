package generator

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/generator/cds"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/utils"
)

// generateRDS creates a new Cluster Discovery Response.
func (g *EnvoyConfigGenerator) generateCDS(ctx context.Context, proxy *models.Proxy) ([]types.Resource, error) {
	meshConfig := g.catalog.GetMeshConfig()
	cb := cds.NewClusterBuilder().SetProxyIdentity(proxy.Identity).SetSidecarSpec(meshConfig.Spec.Sidecar).SetEgressEnabled(meshConfig.Spec.Traffic.EnableEgress)

	outboundMeshTrafficPolicy := g.catalog.GetOutboundMeshTrafficPolicy(proxy.Identity)
	if outboundMeshTrafficPolicy != nil {
		cb.SetOutboundMeshTrafficClusterConfigs(outboundMeshTrafficPolicy.ClustersConfigs)
	}

	proxyServices, err := g.catalog.ListServicesForProxy(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Str("proxy", proxy.String()).Msg("Error looking up MeshServices associated with proxy")
		return nil, err
	}

	inboundMeshTrafficPolicy := g.catalog.GetInboundMeshTrafficPolicy(proxy.Identity, proxyServices)
	if inboundMeshTrafficPolicy != nil {
		cb.SetInboundMeshTrafficClusterConfigs(inboundMeshTrafficPolicy.ClustersConfigs)
	}

	if egressTrafficPolicy, err := g.catalog.GetEgressTrafficPolicy(proxy.Identity); err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress policies for proxy with identity %s, skipping egress clusters", proxy.Identity)
	} else {
		if egressTrafficPolicy != nil {
			cb.SetEgressTrafficClusterConfigs(egressTrafficPolicy.ClustersConfigs)
		}
	}

	if enabled, err := g.catalog.IsMetricsEnabled(proxy); err != nil {
		log.Warn().Str("proxy", proxy.String()).Msg("Could not find pod for connecting proxy, no metadata was recorded")
	} else if enabled {
		cb.SetMetricsEnabled(enabled)
	}

	if meshConfig.Spec.Observability.Tracing.Enable {
		tracingAddress := envoy.GetAddress(utils.GetTracingHost(meshConfig), utils.GetTracingPort(meshConfig))
		cb.SetEnvoyTracingAddress(tracingAddress)
	}

	// Build upstream, local, egress, outbound passthrough, inbound prometheus and outbound tracing clusters per mesh policies
	clusters, err := cb.Build()
	if err != nil {
		return nil, err
	}
	return clusters, nil
}
