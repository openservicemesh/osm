package cds

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

const (
	edsClusterName = "eds"
	edsAddress     = "eds.smc.svc.cluster.local"
	edsPort        = uint32(15124)
)

func getEDS() *v2.Cluster {
	return &v2.Cluster{
		ConnectTimeout:       getTimeout(),
		ClusterDiscoveryType: &v2.Cluster_Type{Type: v2.Cluster_LOGICAL_DNS},
		Name:                 edsClusterName,
		Http2ProtocolOptions: getHttp2(),
		TransportSocket:      getTransportSocket(),
		LoadAssignment:       getLoadAssignment(edsClusterName, edsAddress, edsPort),
	}
}
