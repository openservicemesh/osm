package cds

import (
	"context"
	"reflect"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	proxyServiceName := proxy.GetService()
	allTrafficPolicies, err := catalog.ListTrafficRoutes(proxyServiceName)
	if err != nil {
		glog.Errorf("[%s] Failed listing endpoints: %+v", packageName, err)
		return nil, err
	}
	glog.V(level.Debug).Infof("[%s] TrafficPolicies: %+v for proxy %s", packageName, allTrafficPolicies, proxy.CommonName)
	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeCDS),
	}

	clusterFactories := []xds.Cluster{}
	for _, trafficPolicies := range allTrafficPolicies {
		isSourceService := envoy.Contains(proxyServiceName, trafficPolicies.Source.Services)
		isDestinationService := envoy.Contains(proxyServiceName, trafficPolicies.Destination.Services)
		if isSourceService {
			for _, cluster := range trafficPolicies.Source.Clusters {
				remoteCluster := envoy.GetServiceCluster(string(cluster.ClusterName), proxyServiceName)
				clusterFactories = append(clusterFactories, remoteCluster)
			}
		} else if isDestinationService {
			for _, cluster := range trafficPolicies.Destination.Clusters {
				clusterFactories = append(clusterFactories, getServiceClusterLocal(string(cluster.ClusterName+envoy.LocalCluster)))
			}
		}
	}

	clusterFactories = uniques(clusterFactories)
	for _, cluster := range clusterFactories {
		glog.V(level.Debug).Infof("[%s] Proxy service %s constructed ClusterConfiguration: %+v ", packageName, proxyServiceName, cluster)
		marshalledClusters, err := ptypes.MarshalAny(&cluster)
		if err != nil {
			glog.Errorf("[%s] Failed to marshal cluster for proxy %s: %v", packageName, proxy.GetCommonName(), err)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}
	return resp, nil
}

func uniques(slice []xds.Cluster) []xds.Cluster {
	var isPresent bool
	clusters := []xds.Cluster{}
	for _, entry := range slice {
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
