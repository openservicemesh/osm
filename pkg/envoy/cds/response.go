package cds

import (
	"encoding/json"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
)

// NewClusterDiscoveryResponse creates a new Cluster Discovery Response.
func (s *Server) NewClusterDiscoveryResponse(proxy *envoy.Proxy) (*xds.DiscoveryResponse, error) {
	allServices, err := s.catalog.ListEndpoints("TBD")
	if err != nil {
		glog.Errorf("[%s] Failed listing endpoints: %+v", serverName, err)
		return nil, err
	}
	glog.Infof("[%s] WeightedServices: %+v", serverName, allServices)
	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeCDS),
	}

	var clusterFactories []*xds.Cluster
	for targetedServiceName, weightedServices := range allServices {
		clusterFactories = append(clusterFactories, envoy.GetServiceCluster(string(targetedServiceName), "bookstore.mesh"))
		for _, localservice := range weightedServices {
			clusterFactories = append(clusterFactories, getServiceClusterLocal(string(localservice.ServiceName)))
		}
	}

	for _, cluster := range clusterFactories {
		if clusterJSON, err := json.Marshal(cluster); err == nil {
			glog.V(level.Trace).Infof("[%s] Constructed ClusterConfiguration: %+v", serverName, string(clusterJSON))
		} else {
			glog.Error("[%s] Error marshaling cluster: %s", serverName, cluster)
		}
		marshalledClusters, err := ptypes.MarshalAny(cluster)
		if err != nil {
			glog.Errorf("[%s] Failed to marshal cluster for proxy %s: %v", serverName, proxy.GetCommonName(), err)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}
	return resp, nil
}
