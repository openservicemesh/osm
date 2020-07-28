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

		// The maximum number of parallel requests that Envoy will make to the upstream cluster. If not specified, the default is 1024.
		threshold.MaxRequests = &wrappers.UInt32Value{
			Value: *maxConnections,
		}

		threshold.MaxPendingRequests = &wrappers.UInt32Value{
			Value: *maxConnections,
		}

		threshold.MaxRetries = &wrappers.UInt32Value{
			Value: *maxConnections,
		}

		threshold.MaxConnectionPools = &wrappers.UInt32Value{
			Value: *maxConnections,
		}

	}

	return []*xds_cluster.CircuitBreakers_Thresholds{
		threshold,
	}
}
