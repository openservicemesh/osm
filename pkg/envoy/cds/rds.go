package cds

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

const (
	rdsClusterName = "rds"
	rdsAddress     = "rds.smc.svc.cluster.local"
	rdsPort        = uint32(15126)
)

func getRDS() *v2.Cluster {
	return &v2.Cluster{
		ConnectTimeout:       getTimeout(),
		ClusterDiscoveryType: &v2.Cluster_Type{Type: v2.Cluster_LOGICAL_DNS},
		Name:                 rdsClusterName,
		Http2ProtocolOptions: getHttp2(),
		TransportSocket:      getTransportSocket(),
		LoadAssignment:       getLoadAssignment(rdsClusterName, rdsAddress, rdsPort),
	}
}
