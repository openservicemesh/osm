package lds

import (
	"testing"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/featureflags"
)

// Tests TestGetFilterForService checks that a proper filter type is properly returned
// for given config parameters and service
func TestGetFilterForService(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	lb := &listenerBuilder{
		cfg: mockConfigurator,
	}

	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false)
	mockConfigurator.EXPECT().IsTracingEnabled().Return(true)
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-endpoint")

	// Check we get HTTP connection manager filter without Permissive mode
	filter, err := lb.getOutboundHTTPFilter()

	assert.NoError(err)
	assert.Equal(filter.Name, wellknown.HTTPConnectionManager)

	// Check we get HTTP connection manager filter with Permissive mode
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(true)
	mockConfigurator.EXPECT().IsTracingEnabled().Return(true)
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-endpoint")

	filter, err = lb.getOutboundHTTPFilter()
	assert.NoError(err)
	assert.Equal(filter.Name, wellknown.HTTPConnectionManager)
}

var _ = Describe("Construct inbound listeners", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).AnyTimes()
	mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).AnyTimes()

	Context("Test creation of inbound listener", func() {
		It("Tests the inbound listener config", func() {
			listener := newInboundListener()
			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort)))
			Expect(len(listener.ListenerFilters)).To(Equal(2)) // TlsInspector, OriginalDestination listener filter
			Expect(listener.ListenerFilters[0].Name).To(Equal(wellknown.TlsInspector))
			Expect(listener.TrafficDirection).To(Equal(xds_core.TrafficDirection_INBOUND))
		})
	})

	Context("Test creation of Prometheus listener", func() {
		It("Tests the Prometheus listener config", func() {
			connManager := getPrometheusConnectionManager("fake-prometheus", constants.PrometheusScrapePath, constants.EnvoyMetricsCluster)
			listener, _ := buildPrometheusListener(connManager)
			Expect(listener.Address).To(Equal(envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyPrometheusInboundListenerPort)))
			Expect(len(listener.ListenerFilters)).To(Equal(0)) //  no listener filters
			Expect(listener.TrafficDirection).To(Equal(xds_core.TrafficDirection_INBOUND))
		})
	})
})

var _ = Describe("Test getHTTPConnectionManager", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	Context("Test creation of HTTP connection manager", func() {
		It("Returns proper Zipkin config given when tracing is enabled", func() {
			mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).Times(1)
			mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).Times(1)
			mockConfigurator.EXPECT().GetTracingEndpoint().Return(constants.DefaultTracingEndpoint).Times(1)
			mockConfigurator.EXPECT().IsTracingEnabled().Return(true).Times(1)

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, nil)

			Expect(connManager.Tracing.Verbose).To(Equal(true))
			Expect(connManager.Tracing.Provider.Name).To(Equal("envoy.tracers.zipkin"))
		})

		It("Returns proper Zipkin config given when tracing is disabled", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).Times(1)

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, nil)
			var nilHcmTrace *xds_hcm.HttpConnectionManager_Tracing = nil

			Expect(connManager.Tracing).To(Equal(nilHcmTrace))
		})

		It("Returns no stats config when WASM is disabled", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().AnyTimes()
			oldWASMflag := featureflags.Features.WASMStats
			featureflags.Features.WASMStats = false

			oldStatsWASMBytes := statsWASMBytes
			statsWASMBytes = "some bytes"

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, map[string]string{"k1": "v1"})

			Expect(connManager.HttpFilters).To(HaveLen(2))
			Expect(connManager.HttpFilters[0].GetName()).To(Equal(wellknown.HTTPRoleBasedAccessControl))
			Expect(connManager.HttpFilters[1].GetName()).To(Equal(wellknown.Router))
			Expect(connManager.LocalReplyConfig).To(BeNil())

			// reset global state
			statsWASMBytes = oldStatsWASMBytes
			featureflags.Features.WASMStats = oldWASMflag
		})

		It("Returns proper stats config when WASM is enabled", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().AnyTimes()
			oldWASMflag := featureflags.Features.WASMStats
			featureflags.Features.WASMStats = true

			oldStatsWASMBytes := statsWASMBytes
			statsWASMBytes = "some bytes"

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, map[string]string{"k1": "v1"})

			Expect(connManager.GetHttpFilters()).To(HaveLen(4))
			Expect(connManager.GetHttpFilters()[0].GetName()).To(Equal(wellknown.Lua))
			Expect(connManager.GetHttpFilters()[1].GetName()).To(Equal("envoy.filters.http.wasm"))
			Expect(connManager.GetHttpFilters()[2].GetName()).To(Equal(wellknown.HTTPRoleBasedAccessControl))
			Expect(connManager.GetHttpFilters()[3].GetName()).To(Equal(wellknown.Router))

			Expect(connManager.GetLocalReplyConfig().GetMappers()[0].HeadersToAdd[0].Header.Value).To(Equal("unknown"))

			// reset global state
			statsWASMBytes = oldStatsWASMBytes
			featureflags.Features.WASMStats = oldWASMflag
		})
	})
})
