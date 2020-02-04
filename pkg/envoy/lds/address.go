package lds

import (
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

const (
	port    = 15001
	address = "0.0.0.0"
)

func getAddress() *core.Address {
	// TODO(draychev): figure this out from the service
	listenerPort := uint32(port)
	return &core.Address{
		Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Protocol: core.SocketAddress_TCP,
				Address:  address,
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: listenerPort,
				},
			},
		},
	}
}
