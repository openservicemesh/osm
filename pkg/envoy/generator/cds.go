package generator

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/generator/cds"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

// generateRDS creates a new Cluster Discovery Response.
func (g *EnvoyConfigGenerator) generateCDS(ctx context.Context, proxy *models.Proxy) ([]types.Resource, error) {
	meshConfig := g.catalog.GetMeshConfig()
	cb := cds.NewClusterBuilder().SetProxyIdentity(proxy.Identity).SetSidecarSpec(meshConfig.Spec.Sidecar).SetEgressEnabled(meshConfig.Spec.Traffic.EnableEgress)

	outboundMeshClusterConfigs := g.catalog.GetOutboundMeshClusterConfigs(proxy.Identity)
	cb.SetOutboundMeshTrafficClusterConfigs(outboundMeshClusterConfigs)

	proxyServices, err := g.catalog.ListServicesForProxy(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Str("proxy", proxy.String()).Msg("Error looking up MeshServices associated with proxy")
		return nil, err
	}

	inboundTPBuilder := trafficpolicy.InboundTrafficPolicyBuilder()
	inboundTPBuilder.UpstreamServices(proxyServices)
	allUpstreamSvcIncludeApex := g.catalog.GetUpstreamServicesIncludeApex(proxyServices)
	inboundTPBuilder.UpstreamServicesIncludeApex(allUpstreamSvcIncludeApex)
	upstreamTrafficSettingsPerService := make(map[service.MeshService]*policyv1alpha1.UpstreamTrafficSetting)

	for _, upstreamSvc := range allUpstreamSvcIncludeApex {
		upstreamSvc := upstreamSvc // To prevent loop variable memory aliasing in for loop
		upstreamTrafficSettingsPerService[upstreamSvc] = g.catalog.GetUpstreamTrafficSettingByService(&upstreamSvc)
	}

	inboundTPBuilder.UpstreamTrafficSettingsPerService(upstreamTrafficSettingsPerService)

	inboundMeshClusterConfigs := inboundTPBuilder.GetInboundMeshClusterConfigs()
	cb.SetInboundMeshTrafficClusterConfigs(inboundMeshClusterConfigs)

	if egressClusterConfigs, err := g.catalog.GetEgressClusterConfigs(proxy.Identity); err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress cluster configs for proxy with identity %s, skipping egress clusters", proxy.Identity)
	} else {
		cb.SetEgressTrafficClusterConfigs(egressClusterConfigs)
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

	cb.SetOpenTelemetryExtSvc(g.catalog.GetTelemetryConfig(proxy).OpenTelemetryService)

	// Build upstream, local, egress, outbound passthrough, inbound prometheus and outbound tracing clusters per mesh policies
	clusters, err := cb.Build()
	if err != nil {
		return nil, err
	}
	return clusters, nil
}
