package lds

import (
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

var _ = Describe("Construct inbound listener object", func() {
	Context("Testing the creating of outbound listener", func() {
		It("Returns an outbound listener config", func() {
			connManager := getHTTPConnectionManager("fake-outbound")
			listener, _ := buildOutboundListener(connManager)
			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort)))
			Expect(len(listener.ListenerFilters)).To(Equal(0)) // no listener filters
			Expect(listener.TrafficDirection).To(Equal(envoy_api_v2_core.TrafficDirection_OUTBOUND))
		})
	})

	Context("Testing the creating of inbound listener", func() {
		It("Returns an inbound listener config", func() {
			listener, _ := buildInboundListener()
			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort)))
			Expect(len(listener.ListenerFilters)).To(Equal(1)) // tls-inpsector listener filter
			Expect(listener.ListenerFilters[0].Name).To(Equal(wellknown.TlsInspector))
			Expect(listener.TrafficDirection).To(Equal(envoy_api_v2_core.TrafficDirection_INBOUND))
		})
	})

	Context("Testing the creating of Prometheus listener", func() {
		It("Returns Prometheus listener config", func() {
			connManager := getHTTPConnectionManager("fake-prometheus")
			listener, _ := buildPrometheusListener(connManager)
			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyPrometheusInboundListenerPort)))
			Expect(len(listener.ListenerFilters)).To(Equal(0)) //  no listener filters
			Expect(listener.TrafficDirection).To(Equal(envoy_api_v2_core.TrafficDirection_INBOUND))
		})
	})
})
