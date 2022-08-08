package lds

import (
	"testing"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestHTTPConnManagerBuilder(t *testing.T) {
	contains := func(filters []*xds_hcm.HttpFilter, filterName string) bool {
		for _, f := range filters {
			if f.Name == filterName {
				return true
			}
		}
		return false
	}

	testCases := []struct {
		name       string
		buildFunc  func(b *httpConnManagerBuilder)
		assertFunc func(*assert.Assertions, *xds_hcm.HttpConnectionManager)
	}{
		{
			name: "all properties are correctly set",
			buildFunc: func(b *httpConnManagerBuilder) {
				b.StatsPrefix("foo").
					RouteConfigName("bar").
					AddFilter(&xds_hcm.HttpFilter{Name: "f1"}).
					AddFilter(&xds_hcm.HttpFilter{Name: "f2"}).
					LocalReplyConfig(&xds_hcm.LocalReplyConfig{}).
					Tracing(&xds_hcm.HttpConnectionManager_Tracing{})
			},
			assertFunc: func(a *assert.Assertions, hcm *xds_hcm.HttpConnectionManager) {
				a.Equal("foo", hcm.StatPrefix)
				a.Equal("bar", hcm.GetRds().RouteConfigName)
				a.True(contains(hcm.HttpFilters, envoy.HTTPRBACFilterName))
				a.True(contains(hcm.HttpFilters, envoy.HTTPLocalRateLimitFilterName))
				a.True(contains(hcm.HttpFilters, "f1"))
				a.True(contains(hcm.HttpFilters, "f2"))
				a.ElementsMatch(&xds_hcm.LocalReplyConfig{}, hcm.LocalReplyConfig)
				a.Equal(&xds_hcm.HttpConnectionManager_Tracing{}, hcm.Tracing)
				a.True(hcm.GenerateRequestId.Value)
				a.Equal(websocketUpgradeType, hcm.UpgradeConfigs[0].UpgradeType)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			b := HTTPConnManagerBuilder()
			tc.buildFunc(b)

			filter, err := b.Build()
			hcm := &xds_hcm.HttpConnectionManager{}
			if err == nil {
				err := filter.GetTypedConfig().UnmarshalTo(hcm)
				a.Nil(err)
			}

			tc.assertFunc(a, hcm)
		})
	}
}

func TestBuildOutboundHTTPFilter(t *testing.T) {
	a := assert.New(t)
	lb := &listenerBuilder{
		httpTracingEndpoint: "foo.com/bar",
		wasmStatsHeaders:    map[string]string{"k1": "v1", "k2": "v2"},
	}

	filter, err := lb.buildOutboundHTTPFilter(route.OutboundRouteConfigName)
	a.NoError(err)
	a.Equal(filter.Name, envoy.HTTPConnectionManagerFilterName)
}

func TestBuildInboundFilterChains(t *testing.T) {
	testCases := []struct {
		name                     string
		inboundMeshTrafficPolicy *trafficpolicy.InboundMeshTrafficPolicy
		ingressTrafficPolicies   []*trafficpolicy.IngressTrafficPolicy
		expectedFilterChains     int
	}{
		{
			name: "multiple HTTP and TCP traffic matches for inbound and ingress",
			inboundMeshTrafficPolicy: &trafficpolicy.InboundMeshTrafficPolicy{
				TrafficMatches: []*trafficpolicy.TrafficMatch{
					{
						Name:                "1",
						DestinationPort:     80,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"1.1.1.1/32"},
						Cluster:             "foo",
					},
					{
						Name:                "2",
						DestinationPort:     90,
						DestinationProtocol: "grpc",
						DestinationIPRanges: []string{"1.1.1.1/32"},
						Cluster:             "foo",
					},
					{
						Name:                "3",
						DestinationPort:     100,
						DestinationProtocol: "tcp",
						Cluster:             "foo",
						DestinationIPRanges: []string{"1.1.1.1/32", "2.2.2.2/32"},
					},
					{
						Name:                "4",
						DestinationPort:     100,
						DestinationProtocol: "tcp-server-first",
						Cluster:             "foo",
						DestinationIPRanges: []string{"1.1.1.1/32", "2.2.2.2/32"},
					},
				},
			},
			ingressTrafficPolicies: []*trafficpolicy.IngressTrafficPolicy{
				{
					TrafficMatches: []*trafficpolicy.IngressTrafficMatch{
						{
							Name:           "1",
							Port:           80,
							Protocol:       "http",
							SourceIPRanges: []string{"1.1.1.1/32"},
							ServerNames:    []string{"foo.com"},
						},
						{
							Name:                     "1",
							Port:                     90,
							Protocol:                 "http",
							SourceIPRanges:           []string{"1.1.1.1/32"},
							SkipClientCertValidation: true,
						},
					},
				},
			},
			expectedFilterChains: 6, // 4 in-mesh + 2 ingress
		},
		{
			name:                     "nil InboundMeshTrafficPolicy/IngressTrafficPolicy should result in 0 filter chains",
			inboundMeshTrafficPolicy: nil,
			ingressTrafficPolicies:   nil,
			expectedFilterChains:     0,
		},
		{
			name:                     "nil TrafficMatch should result in 0 filter chains",
			inboundMeshTrafficPolicy: &trafficpolicy.InboundMeshTrafficPolicy{TrafficMatches: nil},
			ingressTrafficPolicies:   []*trafficpolicy.IngressTrafficPolicy{},
			expectedFilterChains:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			lb := &listenerBuilder{
				inboundMeshTrafficPolicy: tc.inboundMeshTrafficPolicy,
				ingressTrafficPolicies:   tc.ingressTrafficPolicies,
				proxyIdentity:            tests.BookbuyerServiceIdentity,
			}

			actual := lb.buildInboundFilterChains()
			a.Len(actual, tc.expectedFilterChains)
		})
	}
}

func TestFilterBuilder(t *testing.T) {
	testCases := []struct {
		name                   string
		fb                     *filterBuilder
		prep                   func(fb *filterBuilder)
		expectedTCPProxy       *xds_tcp_proxy.TcpProxy
		expectedNetworkFilters []string
		expectedHTTPFilters    []string
		expectErr              bool
	}{
		{
			name: "TCP proxy to single upstream cluster",
			prep: func(fb *filterBuilder) {
				fb.TCPProxy().StatsPrefix("test").Cluster("foo")
			},
			expectedTCPProxy: &xds_tcp_proxy.TcpProxy{
				StatPrefix:       "test",
				ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: "foo"},
			},
			expectedNetworkFilters: []string{envoy.TCPProxyFilterName},
		},
		{
			name: "TCP proxy to multiple upstream clusters",
			prep: func(fb *filterBuilder) {
				fb.TCPProxy().StatsPrefix("test").WeightedClusters([]service.WeightedCluster{
					{ClusterName: "foo", Weight: 40},
					{ClusterName: "bar", Weight: 60},
				})
			},
			expectedTCPProxy: &xds_tcp_proxy.TcpProxy{
				StatPrefix: "test",
				ClusterSpecifier: &xds_tcp_proxy.TcpProxy_WeightedClusters{

					WeightedClusters: &xds_tcp_proxy.TcpProxy_WeightedCluster{
						Clusters: []*xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight{
							{
								Name:   "foo",
								Weight: 40,
							},
							{
								Name:   "bar",
								Weight: 60,
							},
						},
					},
				},
			},
			expectedNetworkFilters: []string{envoy.TCPProxyFilterName},
		},
		{
			name: "TCP proxy without a valid cluster should error",
			prep: func(fb *filterBuilder) {
				fb.TCPProxy().StatsPrefix("test")
			},
			expectedTCPProxy: nil,
			expectErr:        true,
		},
		{
			name: "TCP proxy both cluster and weightedCusters should error",
			prep: func(fb *filterBuilder) {
				fb.TCPProxy().StatsPrefix("test").Cluster("foo").WeightedClusters([]service.WeightedCluster{
					{ClusterName: "foo", Weight: 40},
				})
			},
			expectedTCPProxy: nil,
			expectErr:        true,
		},
		{
			name: "HTTP filters",
			prep: func(fb *filterBuilder) {
				fb.WithRBAC([]trafficpolicy.TrafficTargetWithRoutes{
					{
						Name:        "ns-1/test-1",
						Destination: identity.ServiceIdentity("sa-1.ns-1"),
						Sources: []identity.ServiceIdentity{
							identity.ServiceIdentity("sa-2.ns-2"),
						},
						TCPRouteMatches: nil,
					},
				}, "cluster.local").
					httpConnManager()
			},
			expectedNetworkFilters: []string{envoy.L4RBACFilterName},
			expectedHTTPFilters:    []string{envoy.HTTPRBACFilterName, envoy.HTTPLocalRateLimitFilterName, envoy.HTTPRouterFilterName},
		},
	}

	getTCPProxyFilter := func(filters []*xds_listener.Filter) *xds_tcp_proxy.TcpProxy {
		for _, f := range filters {
			if f.Name == envoy.TCPProxyFilterName {
				unmarshalled := &xds_tcp_proxy.TcpProxy{}
				_ = f.GetTypedConfig().UnmarshalTo(unmarshalled)
				return unmarshalled
			}
		}
		return nil
	}

	getHCMFilter := func(filters []*xds_listener.Filter) *xds_hcm.HttpConnectionManager {
		for _, f := range filters {
			if f.Name == envoy.HTTPConnectionManagerFilterName {
				unmarshalled := &xds_hcm.HttpConnectionManager{}
				_ = f.GetTypedConfig().UnmarshalTo(unmarshalled)
				return unmarshalled
			}
		}
		return nil
	}

	containsNetworkFilter := func(filters []*xds_listener.Filter, name string) bool {
		for _, f := range filters {
			if f.Name == name {
				return true
			}
		}
		return false
	}

	containsHTTPFilter := func(filters []*xds_hcm.HttpFilter, name string) bool {
		for _, f := range filters {
			if f.Name == name {
				return true
			}
		}
		return false
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			fb := getFilterBuilder()
			tc.prep(fb)

			filters, err := fb.Build()
			a.Equal(tc.expectErr, err != nil)

			if tc.expectedTCPProxy != nil {
				actual := getTCPProxyFilter(filters)
				a.NotNil(actual)
				a.Equal(tc.expectedTCPProxy.StatPrefix, actual.StatPrefix)
				a.Equal(tc.expectedTCPProxy.ClusterSpecifier, actual.ClusterSpecifier)
			}

			for _, expectedFilter := range tc.expectedNetworkFilters {
				a.True(containsNetworkFilter(filters, expectedFilter))
			}

			hcmFilter := getHCMFilter(filters)
			for _, expectedFilter := range tc.expectedHTTPFilters {
				a.True(containsHTTPFilter(hcmFilter.HttpFilters, expectedFilter))
			}
		})
	}
}

func TestAddFilter(t *testing.T) {
	a := assert.New(t)
	hb := HTTPConnManagerBuilder()

	hb.AddFilter(&xds_hcm.HttpFilter{Name: envoy.HTTPRouterFilterName})
	hb.AddFilter(&xds_hcm.HttpFilter{Name: envoy.HTTPExtAuthzFilterName})

	// Verify the HTTP router filter is always the last filter regardless
	// of the order in which the filters are added
	a.Equal(envoy.HTTPExtAuthzFilterName, hb.filters[0].Name)
	a.Equal(envoy.HTTPRouterFilterName, hb.routerFilter.Name)

	// Verify adding router filter multiple times doesn
	hb.AddFilter(&xds_hcm.HttpFilter{Name: envoy.HTTPRouterFilterName})
	a.Equal(envoy.HTTPRouterFilterName, hb.routerFilter.Name)
}
