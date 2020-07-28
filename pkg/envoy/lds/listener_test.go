package lds

import (
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
)

var _ = Describe("Construct inbound and outbound listeners", func() {
	Context("Test creation of outbound listener", func() {
		containsListenerFilter := func(filters []string, filterName string) bool {
			for _, filter := range filters {
				if filter == filterName {
					return true
				}
			}
			return false
		}
		It("Tests the outbound listener config with egress enabled", func() {
			cidr1 := "10.0.0.0/16"
			cidr2 := "10.2.0.0/16"
			cfg := configurator.NewFakeConfiguratorWithOptions(configurator.FakeConfigurator{
				Egress:         true,
				MeshCIDRRanges: []string{cidr1, cidr2},
			})

			listener, err := newOutboundListener(cfg)
			Expect(err).ToNot(HaveOccurred())

			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort)))

			// Test FilterChains
			Expect(len(listener.FilterChains)).To(Equal(2)) // 1. in-mesh, 2. egress
			// Test mesh FilterChain
			Expect(listener.FilterChains[0].Name).To(Equal(outboundMeshFilterChainName))
			Expect(len(listener.FilterChains[0].FilterChainMatch.PrefixRanges)).To(Equal(2)) // 2 CIDRs
			// Test egress FilterChain
			Expect(listener.FilterChains[1].Name).To(Equal(outboundEgressFilterChainName))
			Expect(listener.FilterChains[1].FilterChainMatch).Should(BeNil())

			// Test ListenerFilters
			expectedListenerFilters := []string{wellknown.OriginalDestination, wellknown.TlsInspector}
			Expect(len(listener.ListenerFilters)).To(Equal(len(expectedListenerFilters)))
			for _, filter := range listener.ListenerFilters {
				Expect(containsListenerFilter(expectedListenerFilters, filter.Name)).To(BeTrue())
			}
			Expect(listener.TrafficDirection).To(Equal(envoy_api_v2_core.TrafficDirection_OUTBOUND))
		})

		It("Tests the outbound listener config with egress disabled", func() {
			cfg := configurator.NewFakeConfiguratorWithOptions(configurator.FakeConfigurator{
				Egress: false,
			})

			listener, err := newOutboundListener(cfg)
			Expect(err).ToNot(HaveOccurred())

			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort)))

			// Test FilterChains
			Expect(len(listener.FilterChains)).To(Equal(1)) // Filter chain for in-mesh
			Expect(listener.FilterChains[0].FilterChainMatch).Should(BeNil())

			// Test that the ListenerFilters for egress don't exist
			Expect(len(listener.ListenerFilters)).To(Equal(0))
			Expect(listener.TrafficDirection).To(Equal(envoy_api_v2_core.TrafficDirection_OUTBOUND))
		})
	})

	Context("Tests building outbound egress listener", func() {
		It("Tests that building the outbound egress filter chain succeeds with valid CIDRs", func() {
			cfg := configurator.NewFakeConfiguratorWithOptions(configurator.FakeConfigurator{
				Egress:         false,
				MeshCIDRRanges: []string{"10.0.0.0/16"},
			})
			outboundListener := xds.Listener{
				FilterChains: []*listener.FilterChain{
					{
						Name: "test",
					},
				},
			}
			err := updateOutboundListenerForEgress(&outboundListener, cfg)
			Expect(err).ToNot(HaveOccurred())
		})
		It("Tests that building the outbound egress filter chain fails with invalid CIDRs", func() {
			cfg := configurator.NewFakeConfiguratorWithOptions(configurator.FakeConfigurator{
				Egress:         false,
				MeshCIDRRanges: []string{"10.0.0.0/100"},
			})
			outboundListener := xds.Listener{
				FilterChains: []*listener.FilterChain{
					{
						Name: "test",
					},
				},
			}
			err := updateOutboundListenerForEgress(&outboundListener, cfg)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test creation of inbound listener", func() {
		It("Tests the inbound listener config", func() {
			listener := newInboundListener()
			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort)))
			Expect(len(listener.ListenerFilters)).To(Equal(1)) // tls-inpsector listener filter
			Expect(listener.ListenerFilters[0].Name).To(Equal(wellknown.TlsInspector))
			Expect(listener.TrafficDirection).To(Equal(envoy_api_v2_core.TrafficDirection_INBOUND))
		})
	})

	Context("Test creation of Prometheus listener", func() {
		It("Tests the Prometheus listener config", func() {
			cfg := configurator.NewFakeConfiguratorWithOptions(configurator.FakeConfigurator{})
			connManager := getHTTPConnectionManager("fake-prometheus", cfg)
			listener, _ := buildPrometheusListener(connManager)
			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyPrometheusInboundListenerPort)))
			Expect(len(listener.ListenerFilters)).To(Equal(0)) //  no listener filters
			Expect(listener.TrafficDirection).To(Equal(envoy_api_v2_core.TrafficDirection_INBOUND))
		})
	})

	Context("Test parseCIDR", func() {
		It("Tests that a valid CIDR is parsed correctly", func() {
			cidr := "10.2.0.0/24"
			addr, prefix, err := parseCIDR(cidr)
			Expect(err).ToNot(HaveOccurred())
			Expect(addr).To(Equal("10.2.0.0"))
			Expect(prefix).To(Equal(uint32(24)))
		})

		It("Tests that an invalid CIDR returns an error", func() {
			cidr := "10.2.0.0/99"
			_, _, err := parseCIDR(cidr)
			Expect(err).To(HaveOccurred())
		})
	})
})
