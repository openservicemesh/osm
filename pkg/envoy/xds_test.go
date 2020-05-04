package envoy

import (
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Envoy tools", func() {
	Context("Test GetAddress()", func() {
		It("should return address", func() {
			addr := "blah"
			port := uint32(95346)
			actual := GetAddress(addr, port)
			expected := &envoy_api_v2_core.Address{
				Address: &envoy_api_v2_core.Address_SocketAddress{
					SocketAddress: &envoy_api_v2_core.SocketAddress{
						Protocol: envoy_api_v2_core.SocketAddress_TCP,
						Address:  addr,
						PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
							PortValue: port,
						},
					},
				},
			}

			Expect(actual).To(Equal(expected))
		})
	})
})
