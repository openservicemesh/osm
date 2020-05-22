package cds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/featureflags"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(_ context.Context, catalog catalog.MeshCataloger, _ smi.MeshSpec, proxy *envoy.Proxy, _ *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	svc, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	proxyServiceName := *svc

	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed listing traffic routes for proxy for service name %q", proxyServiceName)
		return nil, err
	}
	log.Debug().Msgf("TrafficPolicies: %+v for proxy %q; service %q", allTrafficPolicies, proxy.CommonName, proxyServiceName)
	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeCDS),
	}

	clusterFactories := make(map[string]xds.Cluster)
	for _, trafficPolicies := range allTrafficPolicies {
		isSourceService := trafficPolicies.Source.Service.Equals(proxyServiceName)
		isDestinationService := trafficPolicies.Destination.Service.Equals(proxyServiceName)
		//iterate through only destination services here since envoy is programmed by destination
		service := trafficPolicies.Destination.Service
		if isSourceService {
			cluster, err := catalog.GetWeightedClusterForService(service)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to find cluster")
				return nil, err
			}
			remoteCluster, err := envoy.GetServiceCluster(string(cluster.ClusterName), proxyServiceName)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to construct service cluster for proxy %s", proxyServiceName)
			}
			clusterFactories[remoteCluster.Name] = *remoteCluster
		} else if isDestinationService {
			cluster, err := catalog.GetWeightedClusterForService(service)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to find cluster")
				return nil, err
			}
			localClusterName := string(cluster.ClusterName + envoy.LocalClusterSuffix)
			localCluster, err := getServiceClusterLocal(catalog, proxyServiceName, localClusterName)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to get local cluster for proxy %s", proxyServiceName)
				return nil, err
			}
			clusterFactories[localClusterName] = *localCluster
		}
	}

	if featureflags.IsIngressEnabled() {
		// Process ingress policy if applicable
		clusterFactories, err = getIngressServiceCluster(proxyServiceName, catalog, clusterFactories)
		if err != nil {
			return nil, err
		}
	}
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

func getIngressServiceCluster(proxyServiceName service.NamespacedService, catalog catalog.MeshCataloger, clusters map[string]xds.Cluster) (map[string]xds.Cluster, error) {
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
	localClusterName := string(ingressWeightedCluster.ClusterName + envoy.LocalClusterSuffix)
	localCluster, err := getServiceClusterLocal(catalog, proxyServiceName, localClusterName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get local cluster for proxy %s", proxyServiceName)
		return nil, err
	}
	clusters[localClusterName] = *localCluster
	return clusters, nil
}
