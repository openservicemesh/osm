package cds

import (
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	backpressure "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
)

func makeThresholds(spec backpressure.BackpressureSpec) []*xds_cluster.CircuitBreakers_Thresholds {

	threshold := &xds_cluster.CircuitBreakers_Thresholds{}

	if spec.MaxConnections != -1 {
		threshold.MaxConnections = &wrappers.UInt32Value{
			Value: uint32(spec.MaxConnections),
		}
	} else {
		log.Trace().Msgf("Backpressure: got default value for maxcon")
	}
	if spec.MaxRequests != -1 {
		threshold.MaxRequests = &wrappers.UInt32Value{
			Value: uint32(spec.MaxRequests),
		}
	} else {
		log.Trace().Msgf("Backpressure: got default value for maxreq")
	}
	if spec.MaxPendingRequests != -1 {
		threshold.MaxPendingRequests = &wrappers.UInt32Value{
			Value: uint32(spec.MaxPendingRequests),
		}
	} else {
		log.Trace().Msgf("Backpressure: got default value for maxpending")
	}
	if spec.MaxRetries != -1 {
		threshold.MaxRetries = &wrappers.UInt32Value{
			Value: uint32(spec.MaxRetries),
		}
	} else {
		log.Trace().Msgf("Backpressure: got default value for maxretries")
	}
	if spec.MaxConnectionPools != -1 {
		threshold.MaxConnectionPools = &wrappers.UInt32Value{
			Value: uint32(spec.MaxConnectionPools),
		}
	} else {
		log.Trace().Msgf("Backpressure: got default value for maxconpools")
	}

	log.Trace().Msgf("Backpressure: CircuitBreaker threshold contains %+v", threshold)

	return []*xds_cluster.CircuitBreakers_Thresholds{
		threshold,
	}
}
