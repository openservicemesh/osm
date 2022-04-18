package lds

import (
	"fmt"
	"testing"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetOutboundHTTPFilterChainForService(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	// Mock calls used to build the HTTP connection manager
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-api").AnyTimes()
	mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
		Enable: false,
	}).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
		EnableWASMStats: false,
	}).AnyTimes()

	lb := &listenerBuilder{
		meshCatalog:     mockCatalog,
		cfg:             mockConfigurator,
		serviceIdentity: tests.BookbuyerServiceIdentity,
	}

	testCases := []struct {
		name                     string
		trafficMatch             trafficpolicy.TrafficMatch
		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectError              bool
	}{
		{
			name: "traffic match with multiple destination IP ranges",
			trafficMatch: trafficpolicy.TrafficMatch{
				Name:            "test",
				DestinationPort: 80,
				DestinationIPRanges: []string{
					"1.1.1.1/32",
					"2.2.2.2/32",
				},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: 80}, // same as 'servicePort'
				PrefixRanges: []*xds_core.CidrRange{
					// The order is guaranteed to be sorted
					{
						AddressPrefix: "1.1.1.1",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
					{
						AddressPrefix: "2.2.2.2",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "traffic match without destination IP ranges",
			trafficMatch: trafficpolicy.TrafficMatch{
				Name:                "test",
				DestinationPort:     80,
				DestinationIPRanges: nil,
			},
			expectedFilterChainMatch: nil,
			expectError:              true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			httpFilterChain, err := lb.getOutboundHTTPFilterChainForService(tc.trafficMatch)

			assert.Equal(err != nil, tc.expectError)

			if err != nil {
				assert.Nil(httpFilterChain)
			} else {
				assert.NotNil(httpFilterChain)
				assert.Len(httpFilterChain.FilterChainMatch.PrefixRanges, len(tc.trafficMatch.DestinationIPRanges))

				for _, filter := range httpFilterChain.Filters {
					assert.Equal(wellknown.HTTPConnectionManager, filter.Name)
				}
			}
		})
	}
}

func TestGetOutboundTCPFilterChainForService(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	lb := &listenerBuilder{
		meshCatalog:     mockCatalog,
		cfg:             mockConfigurator,
		serviceIdentity: tests.BookbuyerServiceIdentity,
	}

	testCases := []struct {
		name                     string
		destinationIPRanges      []string
		servicePort              uint32
		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectError              bool
	}{
		{
			name: "service with multiple endpoints",
			destinationIPRanges: []string{
				"1.1.1.1/32",
				"2.2.2.2/32",
			},
			servicePort: 80, // this can be different from the target port in the endpoints
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: 80}, // same as 'servicePort'
				PrefixRanges: []*xds_core.CidrRange{
					// The order is guaranteed to be sorted
					{
						AddressPrefix: "1.1.1.1",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
					{
						AddressPrefix: "2.2.2.2",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:                     "service with no endpoints",
			destinationIPRanges:      nil,
			servicePort:              80,
			expectedFilterChainMatch: nil,
			expectError:              true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			trafficMatch := trafficpolicy.TrafficMatch{
				Name:                "test",
				DestinationPort:     int(tc.servicePort),
				DestinationIPRanges: tc.destinationIPRanges,
				WeightedClusters:    []service.WeightedCluster{{ClusterName: "bookstore_14001", Weight: 100}},
			}
			tcpFilterChain, err := lb.getOutboundTCPFilterChainForService(trafficMatch)

			assert.Equal(err != nil, tc.expectError)

			if err != nil {
				assert.Nil(tcpFilterChain)
			} else {
				assert.NotNil(tcpFilterChain)
				assert.Len(tcpFilterChain.FilterChainMatch.PrefixRanges, len(tc.destinationIPRanges))

				for _, filter := range tcpFilterChain.Filters {
					assert.Equal(wellknown.TCPProxy, filter.Name)
				}
			}
		})
	}
}

func TestGetInboundMeshHTTPFilterChain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	// Mock calls used to build the HTTP connection manager
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-api").AnyTimes()
	mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
		Enable: false,
	}).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
		EnableWASMStats:        false,
		EnableMulticlusterMode: true,
	}).AnyTimes()
	mockConfigurator.EXPECT().GetMeshConfig().AnyTimes()

	lb := &listenerBuilder{
		meshCatalog:     mockCatalog,
		cfg:             mockConfigurator,
		serviceIdentity: tests.BookbuyerServiceIdentity,
	}

	proxyService := tests.BookbuyerService

	testCases := []struct {
		name           string
		permissiveMode bool
		port           uint16

		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectedFilterNames      []string
		expectError              bool
	}{
		{
			name:           "inbound HTTP filter chain with permissive mode disabled",
			permissiveMode: false,
			port:           80,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 80},
				ServerNames:          []string{proxyService.ServerName()},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{wellknown.RoleBasedAccessControl, wellknown.HTTPConnectionManager},
			expectError:         false,
		},
		{
			name:           "inbound HTTP filter chain with permissive mode enabled",
			permissiveMode: true,
			port:           90,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{proxyService.ServerName()},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{wellknown.HTTPConnectionManager},
			expectError:         false,
		},
	}

	trafficTargets := []trafficpolicy.TrafficTargetWithRoutes{
		{
			Name:        "ns-1/test-1",
			Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
			Sources: []identity.ServiceIdentity{
				identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
				identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
			},
			TCPRouteMatches: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).Times(1)
			if !tc.permissiveMode {
				// mock catalog calls used to build the RBAC filter
				mockCatalog.EXPECT().ListInboundTrafficTargetsWithRoutes(lb.serviceIdentity).Return(trafficTargets, nil).Times(1)
			}

			proxyService.TargetPort = tc.port
			filterChain, err := lb.getInboundMeshHTTPFilterChain(proxyService)

			assert.Equal(err != nil, tc.expectError)
			assert.Equal(filterChain.FilterChainMatch, tc.expectedFilterChainMatch)
			assert.Len(filterChain.Filters, len(tc.expectedFilterNames))
			for i, filter := range filterChain.Filters {
				assert.Equal(filter.Name, tc.expectedFilterNames[i])
			}
		})
	}
}

func TestGetInboundMeshTCPFilterChain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	// Mock calls used to build the HTTP connection manager
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-api").AnyTimes()
	mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
		Enable: false,
	}).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
		EnableMulticlusterMode: true,
	}).AnyTimes()
	mockConfigurator.EXPECT().GetMeshConfig().AnyTimes()

	lb := &listenerBuilder{
		meshCatalog:     mockCatalog,
		cfg:             mockConfigurator,
		serviceIdentity: tests.BookbuyerServiceIdentity,
	}

	proxyService := tests.BookbuyerService

	testCases := []struct {
		name           string
		permissiveMode bool
		port           uint16

		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectedFilterNames      []string
		expectError              bool
	}{
		{
			name:           "inbound TCP filter chain with permissive mode disabled",
			permissiveMode: false,
			port:           80,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 80},
				ServerNames:          []string{proxyService.ServerName()},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{wellknown.RoleBasedAccessControl, wellknown.TCPProxy},
			expectError:         false,
		},

		{
			name:           "inbound TCP filter chain with permissive mode enabled",
			permissiveMode: true,
			port:           90,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{proxyService.ServerName()},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{wellknown.TCPProxy},
			expectError:         false,
		},
	}

	trafficTargets := []trafficpolicy.TrafficTargetWithRoutes{
		{
			Name:        "ns-1/test-1",
			Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
			Sources: []identity.ServiceIdentity{
				identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
				identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
			},
			TCPRouteMatches: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).Times(1)
			if !tc.permissiveMode {
				// mock catalog calls used to build the RBAC filter
				mockCatalog.EXPECT().ListInboundTrafficTargetsWithRoutes(lb.serviceIdentity).Return(trafficTargets, nil).Times(1)
			}

			proxyService.TargetPort = tc.port
			filterChain, err := lb.getInboundMeshTCPFilterChain(proxyService)

			assert.Equal(err != nil, tc.expectError)
			assert.Equal(filterChain.FilterChainMatch, tc.expectedFilterChainMatch)
			assert.Len(filterChain.Filters, len(tc.expectedFilterNames))
			for i, filter := range filterChain.Filters {
				assert.Equal(filter.Name, tc.expectedFilterNames[i])
			}
		})
	}
}

// Tests getOutboundFilterChainMatchForService and ensures the filter chain match returned is as expected
func TestGetOutboundFilterChainMatchForService(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

	lb := newListenerBuilder(mockCatalog, tests.BookbuyerServiceIdentity, mockConfigurator, nil)

	testCases := []struct {
		name                     string
		trafficMatch             trafficpolicy.TrafficMatch
		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectError              bool
	}{
		{
			// test case 1
			name: "outbound HTTP filter chain for traffic match with destination IP ranges",
			trafficMatch: trafficpolicy.TrafficMatch{
				Name:            "test",
				DestinationPort: 80,
				DestinationIPRanges: []string{
					"192.168.10.1/32",
					"192.168.20.2/32",
				},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: 80}, // same as 'servicePort'
				PrefixRanges: []*xds_core.CidrRange{
					{
						AddressPrefix: "192.168.10.1",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
					{
						AddressPrefix: "192.168.20.2",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
				},
			},
			expectError: false,
		},

		{
			// test case 2
			name: "outbound mesh HTTP filter chain for traffic match without destination IP ranges",
			trafficMatch: trafficpolicy.TrafficMatch{
				Name:                "test",
				DestinationPort:     80,
				DestinationIPRanges: nil,
			},
			expectedFilterChainMatch: nil,
			expectError:              true,
		},

		{
			// test case 3
			name: "outbound TCP filter chain for traffic match with destination IP ranges",
			trafficMatch: trafficpolicy.TrafficMatch{
				Name:            "test",
				DestinationPort: 90,
				DestinationIPRanges: []string{
					"192.168.10.1/32",
					"192.168.20.2/32",
				},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: 90}, // same as 'servicePort'
				PrefixRanges: []*xds_core.CidrRange{
					{
						AddressPrefix: "192.168.10.1",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
					{
						AddressPrefix: "192.168.20.2",
						PrefixLen: &wrapperspb.UInt32Value{
							Value: 32,
						},
					},
				},
			},
			expectError: false,
		},

		{
			// test case 4
			name: "outbound TCP filter chain for traffic match without destination IP ranges",
			trafficMatch: trafficpolicy.TrafficMatch{
				Name:                "test",
				DestinationPort:     80,
				DestinationIPRanges: nil,
			},
			expectedFilterChainMatch: nil,
			expectError:              true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			filterChainMatch, err := lb.getOutboundFilterChainMatchForService(tc.trafficMatch)
			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedFilterChainMatch, filterChainMatch)
		})
	}
}

func TestGetOutboundTCPFilter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testCase struct {
		name                   string
		trafficMatch           trafficpolicy.TrafficMatch
		expectedTCPProxyConfig *xds_tcp_proxy.TcpProxy
		expectError            bool
	}

	testCases := []testCase{
		{
			name: "TCP filter for upstream without any traffic split policies",
			trafficMatch: trafficpolicy.TrafficMatch{
				Name: "test",
				WeightedClusters: []service.WeightedCluster{
					{
						ClusterName: "bar/foo_14001",
						Weight:      100,
					},
				},
			},
			expectedTCPProxyConfig: &xds_tcp_proxy.TcpProxy{
				StatPrefix:       "outbound-mesh-tcp-proxy_test",
				ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: "bar/foo_14001"},
			},
			expectError: false,
		},
		{
			name: "TCP filter for upstream with matching traffic split policy",
			trafficMatch: trafficpolicy.TrafficMatch{
				Name: "test",
				WeightedClusters: []service.WeightedCluster{
					{
						ClusterName: "bar/foo-v1_14001",
						Weight:      10,
					},
					{
						ClusterName: "bar/foo-v2_14001",
						Weight:      90,
					},
				},
			},
			expectedTCPProxyConfig: &xds_tcp_proxy.TcpProxy{
				StatPrefix: "outbound-mesh-tcp-proxy_test",
				ClusterSpecifier: &xds_tcp_proxy.TcpProxy_WeightedClusters{
					WeightedClusters: &xds_tcp_proxy.TcpProxy_WeightedCluster{
						Clusters: []*xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight{
							{
								Name:   "bar/foo-v1_14001",
								Weight: 10,
							},
							{
								Name:   "bar/foo-v2_14001",
								Weight: 90,
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

			lb := newListenerBuilder(mockCatalog, tests.BookbuyerServiceIdentity, mockConfigurator, nil)
			filter, err := lb.getOutboundTCPFilter(tc.trafficMatch)

			assert := tassert.New(t)
			assert.Equal(tc.expectError, err != nil)

			actualConfig := &xds_tcp_proxy.TcpProxy{}
			err = filter.GetTypedConfig().UnmarshalTo(actualConfig)
			assert.Nil(err)
			assert.Equal(wellknown.TCPProxy, filter.Name)

			assert.Equal(tc.expectedTCPProxyConfig.ClusterSpecifier, actualConfig.ClusterSpecifier)

			assert.Equal(tc.expectedTCPProxyConfig.StatPrefix, actualConfig.StatPrefix)
		})
	}
}

func TestGetOutboundHTTPFilter(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	lb := &listenerBuilder{
		cfg: mockConfigurator,
	}

	mockConfigurator.EXPECT().IsTracingEnabled()
	mockConfigurator.EXPECT().GetTracingEndpoint()
	mockConfigurator.EXPECT().GetInboundExternalAuthConfig().Return(auth.ExtAuthConfig{
		Enable: false,
	}).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
		EnableWASMStats: false,
	}).AnyTimes()

	filter, err := lb.getOutboundHTTPFilter(route.OutboundRouteConfigName)
	assert.NoError(err)
	assert.Equal(filter.Name, wellknown.HTTPConnectionManager)
}
