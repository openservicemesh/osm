package cds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/featureflags"
	"github.com/open-service-mesh/osm/pkg/smi"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	proxyServiceName := proxy.GetService()
	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed listing traffic routes")
		return nil, err
	}
	log.Debug().Msgf("TrafficPolicies: %+v for proxy %s", allTrafficPolicies, proxy.CommonName)
	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeCDS),
	}

	var clusterFactories []xds.Cluster
	for _, trafficPolicies := range allTrafficPolicies {
		isSourceService := envoy.Contains(proxyServiceName, trafficPolicies.Source.Services)
		isDestinationService := envoy.Contains(proxyServiceName, trafficPolicies.Destination.Services)
		//iterate through only destination services here since envoy is programmed by destination
		for _, service := range trafficPolicies.Destination.Services {
			if isSourceService {
				cluster, err := catalog.GetWeightedClusterForService(service)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to find cluster")
					return nil, err
				}
				remoteCluster := envoy.GetServiceCluster(string(cluster.ClusterName), proxyServiceName)
				clusterFactories = append(clusterFactories, remoteCluster)
			} else if isDestinationService {
				cluster, err := catalog.GetWeightedClusterForService(service)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to find cluster")
					return nil, err
				}
				localCluster, err := getServiceClusterLocal(catalog, proxyServiceName, string(cluster.ClusterName+envoy.LocalClusterSuffix))
				if err != nil {
					log.Error().Err(err).Msgf("Failed to get local cluster for proxy %s", proxyServiceName)
					return nil, err
				}
				clusterFactories = append(clusterFactories, *localCluster)
			}
		}
	}

	if featureflags.IsIngressEnabled() {
		// Process ingress policy if applicable
		clusterFactories, err = getIngressServiceCluster(proxyServiceName, catalog, clusterFactories)
		if err != nil {
			return nil, err
		}
	}

	clusterFactories = uniques(clusterFactories)
	for _, cluster := range clusterFactories {
		log.Debug().Msgf("Proxy service %s constructed ClusterConfiguration: %+v ", proxyServiceName, cluster)
		marshalledClusters, err := ptypes.MarshalAny(&cluster)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal cluster for proxy %s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}

	prometheusCluster := getPrometheusCluster(constants.EnvoyAdminCluster)
	marshalledCluster, err := ptypes.MarshalAny(&prometheusCluster)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to marshal prometheus cluster for proxy %s", proxy.GetCommonName())
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledCluster)
	return resp, nil
}

func uniques(slice []xds.Cluster) []xds.Cluster {
	var isPresent bool
	var clusters []xds.Cluster
	for _, entry := range slice {
		isPresent = false
		for _, cluster := range clusters {
			if cluster.Name == entry.Name {
				isPresent = true
			}
		}
		if !isPresent {
			clusters = append(clusters, entry)
		}
	}
	return clusters
}

func getIngressServiceCluster(proxyServiceName endpoint.NamespacedService, catalog catalog.MeshCataloger, clusters []xds.Cluster) ([]xds.Cluster, error) {
	isIngress, err := catalog.IsIngressService(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking service %s for ingress", proxyServiceName)
		return nil, err
	}
	if !isIngress {
		return clusters, nil
	}
	ingressWeightedCluster, err := catalog.GetIngressWeightedCluster(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weighted ingress clusters for proxy %s", proxyServiceName)
		return clusters, err
	}
	localCluster, err := getServiceClusterLocal(catalog, proxyServiceName, string(ingressWeightedCluster.ClusterName+envoy.LocalClusterSuffix))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get local cluster for proxy %s", proxyServiceName)
		return nil, err
	}
	clusters = append(clusters, *localCluster)
	return clusters, nil
}
