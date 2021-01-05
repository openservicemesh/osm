package cds

import (
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
)

func makeThresholds(maxConnections *uint32) []*xds_cluster.CircuitBreakers_Thresholds {
	// Use Envoy defaults if no limits have been defined
	if maxConnections == nil {
		return nil
	}

	return []*xds_cluster.CircuitBreakers_Thresholds{
		{
			MaxConnections: &wrappers.UInt32Value{
				Value: *maxConnections,
			},
		},
	}
}
