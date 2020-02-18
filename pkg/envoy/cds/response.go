package cds

import (
	"fmt"
	"time"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
)

func svcRemote(clusterName string, certificateName string) *xds.Cluster {
	// The name must match the domain being cURLed in the demo
	return envoy.GetServiceCluster(clusterName, certificateName)
}

func svcLocal(clusterName string, _ string) *xds.Cluster {
	// The name must match the domain being cURLed in the demo
	return getServiceClusterLocal(clusterName)
}

func (s *Server) NewClusterDiscoveryResponse(proxy envoy.Proxyer) (*xds.DiscoveryResponse, error) {
	allServices, err := s.catalog.ListEndpoints("TBD")
	if err != nil {
		glog.Errorf("[%s][stream] Failed listing endpoints: %+v", serverName, err)
		return nil, err
	}
	glog.Infof("[%s][stream] WeightedServices: %+v", serverName, allServices)
	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeCDS),
	}

	clusterFactories := []*xds.Cluster{}
	for targetedServiceName, weightedServices := range allServices {
		clusterFactories = append(clusterFactories, svcRemote(string(targetedServiceName), "bookstore.mesh"))
		for _, localservice := range weightedServices {
			clusterFactories = append(clusterFactories, svcLocal(string(localservice.ServiceName), "bookstore.mesh"))
		}
	}

	for _, cluster := range clusterFactories {
		glog.V(level.Trace).Infof("[%s] Constructed ClusterConfiguration: %+v", serverName, cluster)
		marshalledClusters, err := ptypes.MarshalAny(cluster)
		if err != nil {
			glog.Errorf("[%s] Failed to marshal cluster for proxy %s: %v", serverName, proxy.GetCommonName(), err)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}
	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = s.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)

	glog.V(level.Trace).Infof("[%s] Constructed response: %+v", serverName, resp)

	return resp, nil
}
