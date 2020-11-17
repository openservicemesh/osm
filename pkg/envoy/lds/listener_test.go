package lds

import (
	"net"
	"testing"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/tests"
)

// Tests TestGetFilterForService checks that a proper filter type is properly returned
// for given config parametres and service
func TestGetFilterForService(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false)
	mockConfigurator.EXPECT().IsTracingEnabled().Return(true)
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-endpoint")

	// Check we get HTTP connection manager filter without Permissive mode
	filter, err := getOutboundFilterForService(tests.BookbuyerService, mockConfigurator)

	assert.NoError(err)
	assert.Equal(filter.Name, wellknown.HTTPConnectionManager)

	// Check we get HTTP connection manager filter with Permissive mode
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(true)
	mockConfigurator.EXPECT().IsTracingEnabled().Return(true)
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-endpoint")

	filter, err = getOutboundFilterForService(tests.BookbuyerService, mockConfigurator)
	assert.NoError(err)
	assert.Equal(filter.Name, wellknown.HTTPConnectionManager)
}

// Tests TestGetFilterChainMatchForService checks that a proper filter chain match is returned
// for a given service
func TestGetFilterChainMatchForService(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

	mockCatalog.EXPECT().GetResolvableServiceEndpoints(tests.BookbuyerService).Return(
		[]endpoint.Endpoint{
			tests.Endpoint,
			{
				// Adding another IP to test multiple-endpoint filter chain
				IP: net.IPv4(192, 168, 0, 1),
			},
		},
		nil,
	)

	filterChainMatch, err := getOutboundFilterChainMatchForService(tests.BookbuyerService, mockCatalog, mockConfigurator)

	assert.NoError(err)
	assert.Equal(filterChainMatch.PrefixRanges[0].GetAddressPrefix(), tests.Endpoint.IP.String())
	assert.Equal(filterChainMatch.PrefixRanges[0].GetPrefixLen().GetValue(), uint32(32))
	assert.Equal(filterChainMatch.PrefixRanges[1].GetAddressPrefix(), net.IPv4(192, 168, 0, 1).String())
	assert.Equal(filterChainMatch.PrefixRanges[1].GetPrefixLen().GetValue(), uint32(32))

	// Test negative getOutboundFilterChainMatchForService when no endpoints are present
	mockCatalog.EXPECT().GetResolvableServiceEndpoints(tests.BookbuyerService).Return(
		[]endpoint.Endpoint{},
		nil,
	)

	filterChainMatch, err = getOutboundFilterChainMatchForService(tests.BookbuyerService, mockCatalog, mockConfigurator)
	assert.NoError(err)
	assert.Nil(filterChainMatch)
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
			Expect(len(listener.ListenerFilters)).To(Equal(1)) // tls-inspector listener filter
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

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator)

			Expect(connManager.Tracing.Verbose).To(Equal(true))
			Expect(connManager.Tracing.Provider.Name).To(Equal("envoy.tracers.zipkin"))
		})

		It("Returns proper Zipkin config given when tracing is disabled", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).Times(1)

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator)
			var nilHcmTrace *xds_hcm.HttpConnectionManager_Tracing = nil

			Expect(connManager.Tracing).To(Equal(nilHcmTrace))
		})
	})
})
