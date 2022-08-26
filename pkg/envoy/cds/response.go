package cds

import (
	mapset "github.com/deckarep/golang-set"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	var clusters []*xds_cluster.Cluster
	meshConfig := meshCatalog.GetMeshConfig()

	// Build upstream clusters based on allowed outbound traffic policies
	outboundMeshTrafficPolicy := meshCatalog.GetOutboundMeshTrafficPolicy(proxy.Identity)
	if outboundMeshTrafficPolicy != nil {
		clusters = append(clusters, upstreamClustersFromClusterConfigs(proxy.Identity, outboundMeshTrafficPolicy.ClustersConfigs, meshConfig.Spec.Sidecar)...)
	}

	// Build local clusters based on allowed inbound traffic policies
	proxyServices, err := meshCatalog.ListServicesForProxy(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Str("proxy", proxy.String()).Msg("Error looking up MeshServices associated with proxy")
		return nil, err
	}
	inboundMeshTrafficPolicy := meshCatalog.GetInboundMeshTrafficPolicy(proxy.Identity, proxyServices)
	if inboundMeshTrafficPolicy != nil {
		clusters = append(clusters, localClustersFromClusterConfigs(inboundMeshTrafficPolicy.ClustersConfigs)...)
	}

	// Add egress clusters based on applied policies
	if egressTrafficPolicy, err := meshCatalog.GetEgressTrafficPolicy(proxy.Identity); err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress policies for proxy with identity %s, skipping egress clusters", proxy.Identity)
	} else {
		if egressTrafficPolicy != nil {
			clusters = append(clusters, getEgressClusters(egressTrafficPolicy.ClustersConfigs)...)
		}
	}

	outboundPassthroughCluser, err := getOriginalDestinationEgressCluster(envoy.OutboundPassthroughCluster, nil)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrGettingOrgDstEgressCluster.String()).
			Msgf("Failed to passthrough cluster for egress for proxy %s", envoy.OutboundPassthroughCluster)
		return nil, err
	}

	// Add an outbound passthrough cluster for egress if global mesh-wide Egress is enabled
	if meshConfig.Spec.Traffic.EnableEgress {
		clusters = append(clusters, outboundPassthroughCluser)
	}

	// Add an inbound prometheus cluster (from Prometheus to localhost)
	if enabled, err := meshCatalog.IsMetricsEnabled(proxy); err != nil {
		log.Warn().Str("proxy", proxy.String()).Msg("Could not find pod for connecting proxy, no metadata was recorded")
	} else if enabled {
		clusters = append(clusters, getPrometheusCluster())
	}

	// Add an outbound tracing cluster (from localhost to tracing sink)
	if meshConfig.Spec.Observability.Tracing.Enable {
		clusters = append(clusters, getTracingCluster(meshConfig))
	}

	return removeDups(clusters), nil
}

func removeDups(clusters []*xds_cluster.Cluster) []types.Resource {
	alreadyAdded := mapset.NewSet()
	var cdsResources []types.Resource
	for _, cluster := range clusters {
		if alreadyAdded.Contains(cluster.Name) {
			log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDuplicateClusters)).
				Msgf("Found duplicate clusters with name %s; duplicate will not be sent to proxy.", cluster.Name)
			continue
		}
		alreadyAdded.Add(cluster.Name)
		cdsResources = append(cdsResources, cluster)
	}

	return cdsResources
}
