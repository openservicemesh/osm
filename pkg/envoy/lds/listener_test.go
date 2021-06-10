package lds

import (
	"testing"
	"time"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

var testWASM = []byte("some bytes")

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
	mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
		Enable: false,
	}).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableWASMStats: false,
	}).AnyTimes()

	// Check we get HTTP connection manager filter without Permissive mode
	filter, err := lb.getOutboundHTTPFilter(route.OutboundRouteConfigName)

	assert.NoError(err)
	assert.Equal(filter.Name, wellknown.HTTPConnectionManager)

	// Check we get HTTP connection manager filter with Permissive mode
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(true)
	mockConfigurator.EXPECT().IsTracingEnabled().Return(true)
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-endpoint")

	filter, err = lb.getOutboundHTTPFilter(route.OutboundRouteConfigName)
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
			connManager := getPrometheusConnectionManager()
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

	Context("Test creation of HTTP connection manager", func() {

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
		})

		It("Should have the correct StatPrefix", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).Times(1)
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats: false,
			}).Times(1)
			connManager := getHTTPConnectionManager("foo", mockConfigurator, nil, outbound)
			Expect(connManager.StatPrefix).To(Equal("mesh-http-conn-manager.foo"))

			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).Times(1)
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats: false,
			}).Times(1)
			connManager = getHTTPConnectionManager("bar", mockConfigurator, nil, outbound)
			Expect(connManager.StatPrefix).To(Equal("mesh-http-conn-manager.bar"))
		})

		It("Returns proper Zipkin config given when tracing is enabled", func() {
			mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).Times(1)
			mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).Times(1)
			mockConfigurator.EXPECT().GetTracingEndpoint().Return(constants.DefaultTracingEndpoint).Times(1)
			mockConfigurator.EXPECT().IsTracingEnabled().Return(true).Times(1)
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats: false,
			}).Times(1)

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, nil, outbound)

			Expect(connManager.Tracing.Verbose).To(Equal(true))
			Expect(connManager.Tracing.Provider.Name).To(Equal("envoy.tracers.zipkin"))
		})

		It("Returns proper Zipkin config given when tracing is disabled", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).Times(1)
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats: false,
			}).Times(1)

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, nil, outbound)
			var nilHcmTrace *xds_hcm.HttpConnectionManager_Tracing = nil

			Expect(connManager.Tracing).To(Equal(nilHcmTrace))
		})

		It("Returns no stats config when WASM is disabled", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().AnyTimes()
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats: false,
			}).Times(1)

			oldStatsWASMBytes := statsWASMBytes
			statsWASMBytes = testWASM

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, map[string]string{"k1": "v1"}, outbound)

			Expect(connManager.HttpFilters).To(HaveLen(2))
			Expect(connManager.HttpFilters[0].GetName()).To(Equal(wellknown.HTTPRoleBasedAccessControl))
			Expect(connManager.HttpFilters[1].GetName()).To(Equal(wellknown.Router))
			Expect(connManager.LocalReplyConfig).To(BeNil())

			// reset global state
			statsWASMBytes = oldStatsWASMBytes
		})

		It("Returns no stats config when WASM is disabled and no WASM is defined", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().AnyTimes()
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats: true,
			}).Times(1)

			oldStatsWASMBytes := statsWASMBytes
			statsWASMBytes = []byte("")

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, map[string]string{"k1": "v1"}, outbound)

			Expect(connManager.HttpFilters).To(HaveLen(2))
			Expect(connManager.HttpFilters[0].GetName()).To(Equal(wellknown.HTTPRoleBasedAccessControl))
			Expect(connManager.HttpFilters[1].GetName()).To(Equal(wellknown.Router))
			Expect(connManager.LocalReplyConfig).To(BeNil())

			// reset global state
			statsWASMBytes = oldStatsWASMBytes
		})

		It("Returns no Lua headers filter config when there are no headers to add", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().AnyTimes()
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats: true,
			}).Times(1)

			oldStatsWASMBytes := statsWASMBytes
			statsWASMBytes = testWASM

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, nil, outbound)

			Expect(connManager.HttpFilters).To(HaveLen(3))
			Expect(connManager.HttpFilters[0].GetName()).To(Equal("envoy.filters.http.wasm"))
			Expect(connManager.HttpFilters[1].GetName()).To(Equal(wellknown.HTTPRoleBasedAccessControl))
			Expect(connManager.HttpFilters[2].GetName()).To(Equal(wellknown.Router))
			Expect(connManager.LocalReplyConfig).To(BeNil())

			// reset global state
			statsWASMBytes = oldStatsWASMBytes
		})

		It("Returns proper stats config when WASM is enabled", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().AnyTimes()
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats: true,
			}).Times(1)

			oldStatsWASMBytes := statsWASMBytes
			statsWASMBytes = testWASM

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, map[string]string{"k1": "v1"}, outbound)

			Expect(connManager.GetHttpFilters()).To(HaveLen(4))
			Expect(connManager.GetHttpFilters()[0].GetName()).To(Equal(wellknown.Lua))
			Expect(connManager.GetHttpFilters()[1].GetName()).To(Equal("envoy.filters.http.wasm"))
			Expect(connManager.GetHttpFilters()[2].GetName()).To(Equal(wellknown.HTTPRoleBasedAccessControl))
			Expect(connManager.GetHttpFilters()[3].GetName()).To(Equal(wellknown.Router))

			Expect(connManager.GetLocalReplyConfig().GetMappers()[0].HeadersToAdd[0].Header.Value).To(Equal("unknown"))

			// reset global state
			statsWASMBytes = oldStatsWASMBytes
		})

		It("Returns inbound external authorization enabled connection manager when enabled by config", func() {
			mockConfigurator.EXPECT().IsTracingEnabled().AnyTimes()
			mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
				Enable:           true,
				Address:          "test.xyz",
				Port:             123,
				StatPrefix:       "pref",
				AuthzTimeout:     3 * time.Second,
				FailureModeAllow: false,
			}).Times(1)
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableWASMStats: false,
			}).Times(1)

			connManager := getHTTPConnectionManager(route.InboundRouteConfigName, mockConfigurator, nil, inbound)

			Expect(connManager.GetHttpFilters()).To(HaveLen(3))
			Expect(connManager.GetHttpFilters()[0].GetName()).To(Equal(wellknown.HTTPRoleBasedAccessControl))
			Expect(connManager.GetHttpFilters()[1].GetName()).To(Equal(wellknown.HTTPExternalAuthorization))
			Expect(connManager.GetHttpFilters()[2].GetName()).To(Equal(wellknown.Router))
		})
	})
})

func TestGetFilterMatchPredicateForPorts(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name          string
		ports         []int
		expectedMatch *xds_listener.ListenerFilterChainMatchPredicate
	}{
		{
			name:  "single port to exclude",
			ports: []int{80},
			expectedMatch: &xds_listener.ListenerFilterChainMatchPredicate{
				Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
					DestinationPortRange: &xds_type.Int32Range{
						Start: 80, // Start is inclusive
						End:   81, // End is exclusive
					},
				},
			},
		},
		{
			name:  "multiple ports to exclude",
			ports: []int{80, 90},
			expectedMatch: &xds_listener.ListenerFilterChainMatchPredicate{
				Rule: &xds_listener.ListenerFilterChainMatchPredicate_OrMatch{
					OrMatch: &xds_listener.ListenerFilterChainMatchPredicate_MatchSet{
						Rules: []*xds_listener.ListenerFilterChainMatchPredicate{
							{
								Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
									DestinationPortRange: &xds_type.Int32Range{
										Start: 80, // Start is inclusive
										End:   81, // End is exclusive
									},
								},
							},
							{
								Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
									DestinationPortRange: &xds_type.Int32Range{
										Start: 90, // Start is inclusive
										End:   91, // End is exclusive
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:          "no ports specified",
			ports:         nil,
			expectedMatch: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getFilterMatchPredicateForPorts(tc.ports)
			assert.Equal(tc.expectedMatch, actual)
		})
	}
}

func TestGetFilterMatchPredicateForTrafficMatches(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name          string
		matches       []*trafficpolicy.TrafficMatch
		expectedMatch *xds_listener.ListenerFilterChainMatchPredicate
	}{
		{
			name: "no server-first ports",
			matches: []*trafficpolicy.TrafficMatch{
				{
					DestinationProtocol: "tcp",
					DestinationPort:     80,
				},
			},
			expectedMatch: nil,
		},
		{
			name: "server-first port present",
			matches: []*trafficpolicy.TrafficMatch{
				{
					DestinationProtocol: "tcp",
					DestinationPort:     80,
				},
				{
					DestinationProtocol: "tcp-server-first",
					DestinationPort:     100,
				},
			},
			expectedMatch: &xds_listener.ListenerFilterChainMatchPredicate{
				Rule: &xds_listener.ListenerFilterChainMatchPredicate_DestinationPortRange{
					DestinationPortRange: &xds_type.Int32Range{
						Start: 100, // Start is inclusive
						End:   101, // End is exclusive
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getFilterMatchPredicateForTrafficMatches(tc.matches)
			assert.Equal(tc.expectedMatch, actual)
		})
	}
}
