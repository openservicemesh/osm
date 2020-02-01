package cds

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

const (
	sdsAddress = "sds.smc.svc.cluster.local"
	sdsPort    = uint32(15123)
)

func getSDS() *v2.Cluster {
	return &v2.Cluster{
		ConnectTimeout:       getTimeout(),
		ClusterDiscoveryType: &v2.Cluster_Type{Type: v2.Cluster_LOGICAL_DNS},
		Name:                 sdsClusterName,
		Http2ProtocolOptions: getHttp2(),
		TransportSocket:      getTransportSocket(),
		LoadAssignment:       getLoadAssignment(sdsAddress, sdsPort),
	}
}
