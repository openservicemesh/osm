package cds

import (
	"fmt"
	"time"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
)

func svcRemote(clusterName string) func() *xds.Cluster {
	return func() *xds.Cluster {
		// The name must match the domain being cURLed in the demo
		return envoy.GetServiceCluster(clusterName)
	}
}

func svcLocal(clusterName string) func() *xds.Cluster {
	return func() *xds.Cluster {
		// The name must match the domain being cURLed in the demo
		return getServiceClusterLocal(clusterName)
	}
}

func (s *Server) newClusterDiscoveryResponse(proxy envoy.Proxyer) (*xds.DiscoveryResponse, error) {
	glog.Infof("[%s] Composing Cluster Discovery Response for proxy: %s", serverName, proxy.GetCommonName())
	resp := &xds.DiscoveryResponse{
		TypeUrl: typeUrl,
	}

	clusterFactories := []func() *xds.Cluster{
		// clusters.GetSDS,
		// clusters.GetEDS,
		// clusters.GetRDS,

		svcRemote("bookstore.mesh"),
		svcLocal("bookstore-local"),
	}

	for _, factory := range clusterFactories {
		cluster := factory()
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

	glog.V(log.LvlTrace).Infof("[%s] Constructed response: %+v", serverName, resp)

	return resp, nil
}
