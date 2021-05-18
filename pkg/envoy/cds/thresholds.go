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

	threshold := &xds_cluster.CircuitBreakers_Thresholds{}

	if maxConnections != nil {
		threshold.MaxConnections = &wrappers.UInt32Value{
			Value: *maxConnections,
		}
	}

	return []*xds_cluster.CircuitBreakers_Thresholds{
		threshold,
	}
}

func makeWSThresholds() []*xds_cluster.CircuitBreakers_Thresholds {
	threshold := &xds_cluster.CircuitBreakers_Thresholds{}

	threshold.MaxConnections = &wrappers.UInt32Value{
		Value: MaxConnectionThreshold,
	}
	threshold.MaxRequests = &wrappers.UInt32Value{
		Value: MaxConnectionThreshold,
	}
	threshold.MaxPendingRequests = &wrappers.UInt32Value{
		Value: MaxConnectionThreshold,
	}

	return []*xds_cluster.CircuitBreakers_Thresholds{
		threshold,
	}
}
