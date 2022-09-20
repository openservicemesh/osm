package lds

import (
	"fmt"
	"testing"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/envoy"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestBuildOutboundHTTPFilterChain(t *testing.T) {
	lb := &listenerBuilder{
		proxyIdentity: tests.BookbuyerServiceIdentity,
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

			httpFilterChain, err := lb.buildOutboundHTTPFilterChain(tc.trafficMatch)

			assert.Equal(err != nil, tc.expectError)

			if err != nil {
				assert.Nil(httpFilterChain)
			} else {
				assert.NotNil(httpFilterChain)
				assert.Len(httpFilterChain.FilterChainMatch.PrefixRanges, len(tc.trafficMatch.DestinationIPRanges))

				for _, filter := range httpFilterChain.Filters {
					assert.Equal(envoy.HTTPConnectionManagerFilterName, filter.Name)
				}
			}
		})
	}
}

func TestBuildOutboundTCPFIlterChain(t *testing.T) {
	lb := &listenerBuilder{
		proxyIdentity: tests.BookbuyerServiceIdentity,
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
			tcpFilterChain, err := lb.buildOutboundTCPFilterChain(trafficMatch)

			assert.Equal(err != nil, tc.expectError)

			if err != nil {
				assert.Nil(tcpFilterChain)
			} else {
				assert.NotNil(tcpFilterChain)
				assert.Len(tcpFilterChain.FilterChainMatch.PrefixRanges, len(tc.destinationIPRanges))

				for _, filter := range tcpFilterChain.Filters {
					assert.Equal(envoy.TCPProxyFilterName, filter.Name)
				}
			}
		})
	}
}

func TestBuildInboundHTTPFilterChain(t *testing.T) {
	testCases := []struct {
		name                    string
		permissiveMode          bool
		trafficMatch            *trafficpolicy.TrafficMatch
		wasmStatsHeaders        map[string]string
		enableActiveHealthCheck bool
		tracingEndpoint         string
		extAuthzConfig          *auth.ExtAuthConfig

		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectedFilterNames      []string
		expectedHTTPFilters      []string
		expectError              bool
	}{
		{
			name:           "inbound HTTP filter chain with permissive mode disabled",
			permissiveMode: false,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_80_http",
				DestinationPort:     80,
				DestinationProtocol: "http",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 80},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{envoy.L4RBACFilterName, envoy.HTTPConnectionManagerFilterName},
			expectError:         false,
		},
		{
			name:           "inbound HTTP filter chain with permissive mode enabled",
			permissiveMode: true,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_90_http",
				DestinationPort:     90,
				DestinationProtocol: "http",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{envoy.HTTPConnectionManagerFilterName},
			expectError:         false,
		},
		{
			name:           "inbound HTTP filter chain with local rate limiting enabled",
			permissiveMode: true,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_90_http",
				DestinationPort:     90,
				DestinationProtocol: "http",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
				RateLimit: &policyv1alpha1.RateLimitSpec{
					Local: &policyv1alpha1.LocalRateLimitSpec{
						TCP: &policyv1alpha1.TCPLocalRateLimitSpec{
							Connections: 100,
							Unit:        "minute",
						},
					},
				},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{envoy.L4LocalRateLimitFilterName, envoy.HTTPConnectionManagerFilterName},
			expectError:         false,
		},
		{
			name:           "inbound HTTP filter chain with global TCP rate limiting enabled",
			permissiveMode: true,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_90_http",
				DestinationPort:     90,
				DestinationProtocol: "http",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
				RateLimit: &policyv1alpha1.RateLimitSpec{
					Global: &policyv1alpha1.GlobalRateLimitSpec{
						TCP: &policyv1alpha1.TCPGlobalRateLimitSpec{
							RateLimitService: policyv1alpha1.RateLimitServiceSpec{
								Host: "foo.bar",
								Port: 8080,
							},
							Descriptors: []policyv1alpha1.TCPRateLimitDescriptor{
								{
									Entries: []policyv1alpha1.TCPRateLimitDescriptorEntry{
										{
											Key:   "k1",
											Value: "v1",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{envoy.L4GlobalRateLimitFilterName, envoy.HTTPConnectionManagerFilterName},
			expectError:         false,
		},
		{
			name:           "inbound HTTP filter chain with global HTTP rate limiting enabled",
			permissiveMode: true,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_90_http",
				DestinationPort:     90,
				DestinationProtocol: "http",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
				RateLimit: &policyv1alpha1.RateLimitSpec{
					Global: &policyv1alpha1.GlobalRateLimitSpec{
						HTTP: &policyv1alpha1.HTTPGlobalRateLimitSpec{
							RateLimitService: policyv1alpha1.RateLimitServiceSpec{
								Host: "foo.bar",
								Port: 8080,
							},
						},
					},
				},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{envoy.HTTPConnectionManagerFilterName},
			expectedHTTPFilters: []string{envoy.HTTPLocalRateLimitFilterName, envoy.HTTPGlobalRateLimitFilterName, envoy.HTTPRouterFilterName},
			expectError:         false,
		},
		{
			name:           "inbound HTTP filter chain with tracing, WASM stats headers, ExtAuthz, active healthcheck",
			permissiveMode: true,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_90_http",
				DestinationPort:     90,
				DestinationProtocol: "http",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			tracingEndpoint:         "foo.com/bar",
			wasmStatsHeaders:        map[string]string{"k1": "v1", "k2": "v2"},
			enableActiveHealthCheck: true,
			extAuthzConfig:          &auth.ExtAuthConfig{Enable: true},
			expectedFilterNames:     []string{envoy.HTTPConnectionManagerFilterName},
			expectError:             false,
		},
	}

	trafficTargets := []trafficpolicy.TrafficTargetWithRoutes{
		{
			Name:        "ns-1/test-1",
			Destination: identity.ServiceIdentity("sa-1.ns-1"),
			Sources: []identity.ServiceIdentity{
				identity.ServiceIdentity("sa-2.ns-2"),
				identity.ServiceIdentity("sa-3.ns-3"),
			},
			TCPRouteMatches: nil,
		},
	}

	containsHTTPFilter := func(filters []*xds_hcm.HttpFilter, filterName string) bool {
		for _, f := range filters {
			if f.Name == filterName {
				return true
			}
		}
		return false
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			lb := &listenerBuilder{
				proxyIdentity:       tests.BookbuyerServiceIdentity,
				permissiveMesh:      tc.permissiveMode,
				trafficTargets:      trafficTargets,
				wasmStatsHeaders:    tc.wasmStatsHeaders,
				activeHealthCheck:   tc.enableActiveHealthCheck,
				httpTracingEndpoint: tc.tracingEndpoint,
			}

			filterChain, err := lb.buildInboundHTTPFilterChain(tc.trafficMatch)

			assert.Equal(err != nil, tc.expectError)
			assert.Equal(filterChain.FilterChainMatch, tc.expectedFilterChainMatch)
			assert.Len(filterChain.Filters, len(tc.expectedFilterNames))
			for i, filter := range filterChain.Filters {
				assert.Equal(tc.expectedFilterNames[i], filter.Name)
			}

			var httpFilters []*xds_hcm.HttpFilter
			for _, f := range filterChain.Filters {
				if f.Name != envoy.HTTPConnectionManagerFilterName {
					continue
				}
				hcm := &xds_hcm.HttpConnectionManager{}
				err := f.GetTypedConfig().UnmarshalTo(hcm)
				assert.Nil(err)

				httpFilters = append(httpFilters, hcm.HttpFilters...)
			}

			for _, httpFilter := range tc.expectedHTTPFilters {
				assert.True(containsHTTPFilter(httpFilters, httpFilter), "expected HTTP filter not found: "+httpFilter)
			}
		})
	}
}

func TestBuildInboundTCPFilterChain(t *testing.T) {
	testCases := []struct {
		name           string
		permissiveMode bool
		trafficMatch   *trafficpolicy.TrafficMatch

		expectedFilterChainMatch *xds_listener.FilterChainMatch
		expectedFilterNames      []string
		expectError              bool
	}{
		{
			name:           "inbound TCP filter chain with permissive mode disabled",
			permissiveMode: false,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_80_http",
				Cluster:             "ns1/svc1_90_http",
				DestinationPort:     80,
				DestinationProtocol: "tcp",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 80},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{envoy.L4RBACFilterName, envoy.TCPProxyFilterName},
			expectError:         false,
		},
		{
			name:           "inbound TCP filter chain with permissive mode enabled",
			permissiveMode: true,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_90_http",
				Cluster:             "ns1/svc1_90_http",
				DestinationPort:     90,
				DestinationProtocol: "tcp",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{envoy.TCPProxyFilterName},
			expectError:         false,
		},
		{
			name:           "inbound TCP filter chain with local TCP rate limiting enabled",
			permissiveMode: true,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_90_http",
				Cluster:             "ns1/svc1_90_http",
				DestinationPort:     90,
				DestinationProtocol: "tcp",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
				RateLimit: &policyv1alpha1.RateLimitSpec{
					Local: &policyv1alpha1.LocalRateLimitSpec{
						TCP: &policyv1alpha1.TCPLocalRateLimitSpec{
							Connections: 100,
							Unit:        "minute",
						},
					},
				},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{envoy.L4LocalRateLimitFilterName, envoy.TCPProxyFilterName},
			expectError:         false,
		},
		{
			name:           "inbound TCP filter chain with global TCP rate limiting enabled",
			permissiveMode: true,
			trafficMatch: &trafficpolicy.TrafficMatch{
				Name:                "inbound_ns1/svc1_90_http",
				Cluster:             "ns1/svc1_90_http",
				DestinationPort:     90,
				DestinationProtocol: "http",
				ServerNames:         []string{"svc1.ns1.svc.cluster.local"},
				RateLimit: &policyv1alpha1.RateLimitSpec{
					Global: &policyv1alpha1.GlobalRateLimitSpec{
						TCP: &policyv1alpha1.TCPGlobalRateLimitSpec{
							RateLimitService: policyv1alpha1.RateLimitServiceSpec{
								Host: "foo.bar",
								Port: 8080,
							},
							Descriptors: []policyv1alpha1.TCPRateLimitDescriptor{
								{
									Entries: []policyv1alpha1.TCPRateLimitDescriptorEntry{
										{
											Key:   "k1",
											Value: "v1",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort:      &wrapperspb.UInt32Value{Value: 90},
				ServerNames:          []string{"svc1.ns1.svc.cluster.local"},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
			expectedFilterNames: []string{envoy.L4GlobalRateLimitFilterName, envoy.TCPProxyFilterName},
			expectError:         false,
		},
	}

	trafficTargets := []trafficpolicy.TrafficTargetWithRoutes{
		{
			Name:        "ns-1/test-1",
			Destination: identity.ServiceIdentity("sa-1.ns-1"),
			Sources: []identity.ServiceIdentity{
				identity.ServiceIdentity("sa-2.ns-2"),
				identity.ServiceIdentity("sa-3.ns-3"),
			},
			TCPRouteMatches: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			lb := &listenerBuilder{
				proxyIdentity:  tests.BookbuyerServiceIdentity,
				permissiveMesh: tc.permissiveMode,
				trafficTargets: trafficTargets,
			}

			filterChain, err := lb.buildInboundTCPFilterChain(tc.trafficMatch)

			assert.Equal(err != nil, tc.expectError, err)
			assert.Equal(filterChain.FilterChainMatch, tc.expectedFilterChainMatch)
			assert.Len(filterChain.Filters, len(tc.expectedFilterNames))
			for i, filter := range filterChain.Filters {
				assert.Equal(filter.Name, tc.expectedFilterNames[i])
			}
		})
	}
}

// Tests buildOutboundFilterChainMatch and ensures the filter chain match returned is as expected
func TestBuildOutboundFilterChainMatch(t *testing.T) {
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

			filterChainMatch, err := buildOutboundFilterChainMatch(tc.trafficMatch)
			assert.Equal(tc.expectError, err != nil)
			assert.Equal(tc.expectedFilterChainMatch, filterChainMatch)
		})
	}
}

func TestBuildOutboundFilterChains(t *testing.T) {
	testCases := []struct {
		name                 string
		policy               *trafficpolicy.OutboundMeshTrafficPolicy
		expectedFilterChains int
	}{
		{
			name: "multiple HTTP and TCP traffic matches",
			policy: &trafficpolicy.OutboundMeshTrafficPolicy{
				TrafficMatches: []*trafficpolicy.TrafficMatch{
					{
						Name:                "1",
						DestinationPort:     80,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"1.1.1.1/32"},
					},
					{
						Name:                "2",
						DestinationPort:     90,
						DestinationProtocol: "grpc",
						DestinationIPRanges: []string{"1.1.1.1/32"},
					},
					{
						Name:                "3",
						DestinationPort:     100,
						DestinationProtocol: "tcp",
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "foo",
								Weight:      100,
							},
						},
						DestinationIPRanges: []string{"1.1.1.1/32", "2.2.2.2/32"},
					},
					{
						Name:                "4",
						DestinationPort:     100,
						DestinationProtocol: "tcp-server-first",
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "foo",
								Weight:      40,
							},
							{
								ClusterName: "bar",
								Weight:      60,
							},
						},
						DestinationIPRanges: []string{"1.1.1.1/32", "2.2.2.2/32"},
					},
				},
			},
			expectedFilterChains: 4,
		},
		{
			name:                 "nil OutboundMeshTrafficPolicy should result in 0 filter chains",
			policy:               nil,
			expectedFilterChains: 0,
		},
		{
			name:                 "nil TrafficMatch should result in 0 filter chains",
			policy:               &trafficpolicy.OutboundMeshTrafficPolicy{TrafficMatches: nil},
			expectedFilterChains: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			lb := &listenerBuilder{outboundMeshTrafficPolicy: tc.policy}

			actual := lb.buildOutboundFilterChains()
			a.Len(actual, tc.expectedFilterChains)
		})
	}
}
