package cds

import (
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	backpressure "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
)

func makeThresholds(spec backpressure.BackpressureSpec) []*envoy_api_v2_cluster.CircuitBreakers_Thresholds {

	threshold := &xds_cluster.CircuitBreakers_Thresholds{}

	if spec.MaxConnections != 0 {
		threshold.MaxConnections = &wrappers.UInt32Value{
			Value: spec.MaxConnections,
		}
	}
	if spec.MaxRequests != 0 {
		threshold.MaxRequests = &wrappers.UInt32Value{
			Value: spec.MaxRequests,
		}
	}
	if spec.MaxPendingRequests != 0 {
		threshold.MaxPendingRequests = &wrappers.UInt32Value{
			Value: spec.MaxPendingRequests,
		}
	}
	if spec.MaxRetries != 0 {
		threshold.MaxRetries = &wrappers.UInt32Value{
			Value: spec.MaxRetries,
		}
	}
	if spec.MaxConnectionPools != 0 {
		threshold.MaxConnectionPools = &wrappers.UInt32Value{
			Value: spec.MaxConnectionPools,
		}
	}

	return []*xds_cluster.CircuitBreakers_Thresholds{
		threshold,
	}
}
