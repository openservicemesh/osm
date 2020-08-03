package cds

import (
	"context"
	"fmt"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/featureflags"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(_ context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator) (*xds_discovery.DiscoveryResponse, error) {
	svc, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	proxyServiceName := *svc

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeCDS),
	}
	// The clusters have to be unique, so use a map to prevent duplicates. Keys correspond to the cluster name.
	clusterFactories := make(map[string]*xds_cluster.Cluster)

	outboundServices, err := catalog.ListAllowedOutboundServices(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Error listing outbound services for proxy %q", proxyServiceName)
		return nil, err
	}

	// Build remote clusters based on allowed outbound services
	for _, dstService := range outboundServices {
		if _, found := clusterFactories[dstService.String()]; found {
			// Guard against duplicates
			continue
		}

		remoteCluster, err := getRemoteServiceCluster(dstService, proxyServiceName)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to construct service cluster for proxy %s", proxyServiceName)
			return nil, err
		}

		if featureflags.IsBackpressureEnabled() {
			enableBackpressure(meshSpec, remoteCluster)
		}

		clusterFactories[remoteCluster.Name] = remoteCluster
	}

	// Create a local cluster for the service.
	// The local cluster will be used for incoming traffic.
	localClusterName := getLocalClusterName(proxyServiceName)
	localCluster, err := getLocalServiceCluster(catalog, proxyServiceName, localClusterName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get local cluster config for proxy %s", proxyServiceName)
		return nil, err
	}
	clusterFactories[localCluster.Name] = localCluster

	if cfg.IsEgressEnabled() {
		// Add a pass-through cluster for egress
		passthroughCluster := getOutboundPassthroughCluster()
		clusterFactories[passthroughCluster.Name] = passthroughCluster
	}

	for _, cluster := range clusterFactories {
		log.Debug().Msgf("Proxy service %s constructed ClusterConfiguration: %+v ", proxyServiceName, cluster)
		marshalledClusters, err := ptypes.MarshalAny(cluster)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal cluster for proxy %s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}

	if cfg.IsPrometheusScrapingEnabled() {
		prometheusCluster := getPrometheusCluster()
		marshalledCluster, err := ptypes.MarshalAny(&prometheusCluster)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshaling Prometheus cluster for proxy with CN=%s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledCluster)
	}

	if cfg.IsZipkinTracingEnabled() {
		zipkinCluster := getZipkinCluster(cfg)
		marshalledCluster, err := ptypes.MarshalAny(&zipkinCluster)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshaling Zipkin cluster for proxy with CN=%s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledCluster)
	}

	return resp, nil
}

func getLocalClusterName(proxyServiceName service.MeshService) string {
	return fmt.Sprintf("%s%s", proxyServiceName, envoy.LocalClusterSuffix)
}
