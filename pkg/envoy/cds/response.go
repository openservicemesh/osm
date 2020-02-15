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

func (s *Server) NewClusterDiscoveryResponse(proxy *envoy.Proxy) (*xds.DiscoveryResponse, error) {

	allServices, err := s.catalog.ListEndpoints("TBD")
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
		glog.V(log.LvlTrace).Infof("[%s] Constructed ClusterConfiguration: %+v", serverName, cluster)
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
