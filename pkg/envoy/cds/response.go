package cds

import (
	"context"
	"fmt"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(_ context.Context, catalog catalog.MeshCataloger, _ smi.MeshSpec, proxy *envoy.Proxy, _ *xds.DiscoveryRequest, config *configurator.Config) (*xds.DiscoveryResponse, error) {
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

	var clusterFactories []*xds.Cluster
	for _, trafficPolicies := range allTrafficPolicies {
		isSourceService := trafficPolicies.Source.Service.Equals(proxyServiceName)
		//iterate through only destination services here since envoy is programmed by destination
		dstService := trafficPolicies.Destination.Service
		if isSourceService {
			remoteCluster, err := envoy.GetServiceCluster(dstService, proxyServiceName)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to construct service cluster for proxy %s", proxyServiceName)
				return nil, err
			}
			clusterFactories = append(clusterFactories, remoteCluster)
		}
	}

	// Create a local cluster for the service.
	// The local cluster will be used for incoming traffic.
	localClusterName := getLocalClusterName(proxyServiceName)
	localCluster, err := getServiceClusterLocal(catalog, proxyServiceName, localClusterName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get local cluster config for proxy %s", proxyServiceName)
		return nil, err
	}
	clusterFactories = append(clusterFactories, localCluster)

	for _, cluster := range clusterFactories {
		log.Debug().Msgf("Proxy service %s constructed ClusterConfiguration: %+v ", proxyServiceName, cluster)
		marshalledClusters, err := ptypes.MarshalAny(cluster)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal cluster for proxy %s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}

	if config.EnablePrometheus {
		prometheusCluster := getPrometheusCluster()
		marshalledCluster, err := ptypes.MarshalAny(&prometheusCluster)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal prometheus cluster for proxy %s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledCluster)
	}

	if config.EnableTracing {
		zipkinCluster := getZipkinCluster(fmt.Sprintf("%s.%s.svc.cluster.local", "zipkin", config.OSMNamespace))
		marshalledCluster, err := ptypes.MarshalAny(&zipkinCluster)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal zipkin cluster for proxy %s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledCluster)
	}

	return resp, nil
}

func getLocalClusterName(proxyServiceName service.NamespacedService) string {
	return fmt.Sprintf("%s%s", proxyServiceName, envoy.LocalClusterSuffix)
}
