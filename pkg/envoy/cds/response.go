package cds

import (
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/utils"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	meshConfig := meshCatalog.GetMeshConfig()
	cb := NewClusterBuilder().SetProxyIdentity(proxy.Identity).SetSidecarSpec(meshConfig.Spec.Sidecar).SetEgressEnabled(meshConfig.Spec.Traffic.EnableEgress)

	outboundMeshTrafficPolicy := meshCatalog.GetOutboundMeshTrafficPolicy(proxy.Identity)
	if outboundMeshTrafficPolicy != nil {
		cb.SetOutboundMeshTrafficClusterConfigs(outboundMeshTrafficPolicy.ClustersConfigs)
	}

	proxyServices, err := meshCatalog.ListServicesForProxy(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Str("proxy", proxy.String()).Msg("Error looking up MeshServices associated with proxy")
		return nil, err
	}

	inboundMeshTrafficPolicy := meshCatalog.GetInboundMeshTrafficPolicy(proxy.Identity, proxyServices)
	if inboundMeshTrafficPolicy != nil {
		cb.SetInboundMeshTrafficClusterConfigs(inboundMeshTrafficPolicy.ClustersConfigs)
	}

	if egressTrafficPolicy, err := meshCatalog.GetEgressTrafficPolicy(proxy.Identity); err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress policies for proxy with identity %s, skipping egress clusters", proxy.Identity)
	} else {
		if egressTrafficPolicy != nil {
			cb.SetEgressTrafficClusterConfigs(egressTrafficPolicy.ClustersConfigs)
		}
	}

	if enabled, err := meshCatalog.IsMetricsEnabled(proxy); err != nil {
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
