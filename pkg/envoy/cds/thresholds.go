package cds

import (
	envoy_api_v2_cluster "github.com/envoyproxy/go-control-plane/envoy/api/v2/cluster"
	"github.com/golang/protobuf/ptypes/wrappers"
)

func makeThresholds(maxConnections *uint32) []*envoy_api_v2_cluster.CircuitBreakers_Thresholds {
	// Use Envoy defaults if no limits have been defined
	if maxConnections == nil {
		return nil
	}

	threshold := &envoy_api_v2_cluster.CircuitBreakers_Thresholds{}

	if maxConnections != nil {
		threshold.MaxConnections = &wrappers.UInt32Value{
			Value: *maxConnections,
		}
	}

	return []*envoy_api_v2_cluster.CircuitBreakers_Thresholds{
		threshold,
	}
}
