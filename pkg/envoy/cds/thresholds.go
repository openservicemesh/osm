package cds

import (
	xds "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
)

func makeThresholds(maxConnections *uint32) []*xds.CircuitBreakers_Thresholds {
	// Use Envoy defaults if no limits have been defined
	if maxConnections == nil {
		return nil
	}

	threshold := &xds.CircuitBreakers_Thresholds{}

	if maxConnections != nil {
		threshold.MaxConnections = &wrappers.UInt32Value{
			Value: *maxConnections,
		}
	}

	return []*xds.CircuitBreakers_Thresholds{
		threshold,
	}
}
