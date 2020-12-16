package cds

import (
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/featureflags"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	svcList, err := meshCatalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	// Github Issue #1575
	proxyServiceName := svcList[0]

	// The clusters have to be unique, so use a map to prevent duplicates. Keys correspond to the cluster name.
	var clusters []*xds_cluster.Cluster

	proxyIdentity, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with CN=%q", proxy.GetCommonName())
		return nil, err
	}

	// Build remote clusters based on allowed outbound services
	for _, dstService := range meshCatalog.ListAllowedOutboundServicesForIdentity(proxyIdentity) {
		cluster, err := getUpstreamServiceCluster(dstService, proxyServiceName, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to construct service cluster for service %s for proxy %s", dstService.Name, proxyServiceName)
			return nil, err
		}

		if featureflags.IsBackpressureEnabled() {
			enableBackpressure(meshCatalog, cluster, dstService)
		}

		clusters = append(clusters, cluster)
	}

	// Create a local cluster for the service.
	// The local cluster will be used for incoming traffic.
	localClusterName := envoy.GetLocalClusterNameForService(proxyServiceName)
	localCluster, err := getLocalServiceCluster(meshCatalog, proxyServiceName, localClusterName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get local cluster config for proxy %s", proxyServiceName)
		return nil, err
	}
	clusters = append(clusters, localCluster)

	// Add an outbound passthrough cluster for egress
	if cfg.IsEgressEnabled() {
		clusters = append(clusters, getOutboundPassthroughCluster())
	}

	// Add an inbound prometheus cluster (from Prometheus to localhost)
	if cfg.IsPrometheusScrapingEnabled() {
		clusters = append(clusters, getPrometheusCluster())
	}

	// Add an outbound tracing cluster (from localhost to tracing sink)
	if cfg.IsTracingEnabled() {
		clusters = append(clusters, getTracingCluster(cfg))
	}

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeCDS),
	}

	for _, cluster := range clusters {
		marshalledClusters, err := ptypes.MarshalAny(cluster)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal cluster %s for proxy %s", cluster.Name, proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}

	return resp, nil
}
