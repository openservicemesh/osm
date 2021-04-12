package route

import (
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/featureflags"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestBuildRouteConfiguration(t *testing.T) {
	assert := tassert.New(t)
	testInbound := &trafficpolicy.InboundTrafficPolicy{
		Name:      "bookstore-v1-default",
		Hostnames: tests.BookstoreV1Hostnames,
		Rules: []*trafficpolicy.Rule{
			{
				Route: trafficpolicy.RouteWeightedClusters{
					HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
					WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
				},
				AllowedServiceAccounts: mapset.NewSet(tests.BookbuyerServiceAccount),
			},
			{
				Route: trafficpolicy.RouteWeightedClusters{
					HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
					WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
				},
				AllowedServiceAccounts: mapset.NewSet(tests.BookbuyerServiceAccount),
			},
		},
	}

	testOutbound := &trafficpolicy.OutboundTrafficPolicy{
		Name:      "bookstore-v1",
		Hostnames: tests.BookstoreV1Hostnames,
		Routes: []*trafficpolicy.RouteWeightedClusters{
			{
				HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
					Path:          "/some-path",
					PathMatchType: trafficpolicy.PathMatchRegex,
					Methods:       []string{"GET"},
				},
				WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
			},
		},
	}
	testCases := []struct {
		name                   string
		inbound                []*trafficpolicy.InboundTrafficPolicy
		outbound               []*trafficpolicy.OutboundTrafficPolicy
		expectedRouteConfigLen int
	}{
		{
			name:                   "no policies provided",
			inbound:                []*trafficpolicy.InboundTrafficPolicy{},
			outbound:               []*trafficpolicy.OutboundTrafficPolicy{},
			expectedRouteConfigLen: 0,
		},
		{
			name:                   "inbound policy provided",
			inbound:                []*trafficpolicy.InboundTrafficPolicy{testInbound},
			outbound:               []*trafficpolicy.OutboundTrafficPolicy{},
			expectedRouteConfigLen: 1,
		},
		{
			name:                   "outbound policy provided",
			inbound:                []*trafficpolicy.InboundTrafficPolicy{},
			outbound:               []*trafficpolicy.OutboundTrafficPolicy{testOutbound},
			expectedRouteConfigLen: 1,
		},
		{
			name:                   "both inbound and outbound policies provided",
			inbound:                []*trafficpolicy.InboundTrafficPolicy{testInbound},
			outbound:               []*trafficpolicy.OutboundTrafficPolicy{testOutbound},
			expectedRouteConfigLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := BuildRouteConfiguration(tc.inbound, tc.outbound, nil)
			assert.Equal(tc.expectedRouteConfigLen, len(actual))
		})
	}

	statsWASMTestCases := []struct {
		name                      string
		wasmEnabled               bool
		expectedResponseHeaderLen int
	}{
		{
			name:                      "response headers added when WASM enabled",
			wasmEnabled:               true,
			expectedResponseHeaderLen: len((&envoy.Proxy{}).StatsHeaders()),
		},
		{
			name:                      "response headers not added when WASM disabled",
			wasmEnabled:               false,
			expectedResponseHeaderLen: 0,
		},
	}

	for _, tc := range statsWASMTestCases {
		t.Run(tc.name, func(t *testing.T) {
			oldWASMflag := featureflags.IsWASMStatsEnabled()
			featureflags.Features.WASMStats = tc.wasmEnabled

			actual := BuildRouteConfiguration([]*trafficpolicy.InboundTrafficPolicy{testInbound}, nil, &envoy.Proxy{})
			tassert.Len(t, actual, 1)
			tassert.Len(t, actual[0].ResponseHeadersToAdd, tc.expectedResponseHeaderLen)

			featureflags.Features.WASMStats = oldWASMflag
		})
	}
}

func TestBuildIngressRouteConfiguration(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                      string
		ingressPolicies           []*trafficpolicy.InboundTrafficPolicy
		expectedRouteConfigFields *xds_route.RouteConfiguration
	}{
		{
			name:                      "no ingress policies",
			ingressPolicies:           nil,
			expectedRouteConfigFields: nil,
		},
		{
			name: "multiple ingress policies",
			ingressPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name:      "bookstore-v1-default",
					Hostnames: []string{"bookstore-v1.default.svc.cluster.local"},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
							},
							AllowedServiceAccounts: mapset.NewSet(service.K8sServiceAccount{}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
							},
							AllowedServiceAccounts: mapset.NewSet(service.K8sServiceAccount{}),
						},
					},
				},
				{
					Name:      "foo.com",
					Hostnames: []string{"foo.com"},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
							},
							AllowedServiceAccounts: mapset.NewSet(service.K8sServiceAccount{}),
						},
					},
				},
			},
			expectedRouteConfigFields: &xds_route.RouteConfiguration{
				Name: "rds-ingress",
				VirtualHosts: []*xds_route.VirtualHost{
					{
						Name: "ingress_virtual-host|bookstore-v1.default.svc.cluster.local",
						Routes: []*xds_route.Route{
							{
								// corresponds to ingressPolicies[0].Rules[0]
							},
							{
								// corresponds to ingressPolicies[0].Rules[1]
							},
						},
					},
					{
						Name: "ingress_virtual-host|foo.com",
						Routes: []*xds_route.Route{
							{
								// corresponds to ingressPolicies[1].Rules[0]
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := BuildIngressConfiguration(tc.ingressPolicies, nil)

			if tc.expectedRouteConfigFields == nil {
				assert.Nil(actual)
				return
			}

			assert.NotNil(actual)
			assert.Equal(tc.expectedRouteConfigFields.Name, actual.Name)
			assert.Len(actual.VirtualHosts, len(tc.expectedRouteConfigFields.VirtualHosts))

			for i, vh := range actual.VirtualHosts {
				assert.Len(vh.Routes, len(tc.expectedRouteConfigFields.VirtualHosts[i].Routes))
			}
		})
	}
}

func TestBuildVirtualHostStub(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name         string
		namePrefix   string
		host         string
		domains      []string
		expectedName string
	}{
		{
			name:         "inbound virtual host",
			namePrefix:   inboundVirtualHost,
			host:         "host",
			domains:      []string{"domain1", "domain2"},
			expectedName: "inbound_virtual-host|host",
		},
		{
			name:         "outbound virtual host",
			namePrefix:   outboundVirtualHost,
			host:         "host",
			domains:      []string{"domain1", "domain2"},
			expectedName: "outbound_virtual-host|host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := buildVirtualHostStub(tc.namePrefix, tc.host, tc.domains)
			assert.Equal(tc.expectedName, actual.Name)
			assert.Equal(tc.domains, actual.Domains)
		})
	}
}
func TestBuildInboundRoutes(t *testing.T) {
	assert := tassert.New(t)

	testWeightedCluster := service.WeightedCluster{
		ClusterName: "testCluster",
		Weight:      100,
	}

	testCases := []struct {
		name       string
		inputRules []*trafficpolicy.Rule
		expectFunc func(actual []*xds_route.Route)
	}{
		{
			name: "valid route rule",
			inputRules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
							Path:          "/hello",
							PathMatchType: trafficpolicy.PathMatchRegex,
							Methods:       []string{"GET"},
							Headers:       map[string]string{"hello": "world"},
						},
						WeightedClusters: mapset.NewSet(testWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSetFromSlice(
						[]interface{}{service.K8sServiceAccount{Name: "foo", Namespace: "bar"}},
					),
				},
			},
			expectFunc: func(actual []*xds_route.Route) {
				assert.Equal(1, len(actual))
				assert.Equal("/hello", actual[0].GetMatch().GetSafeRegex().Regex)
				assert.Equal("GET", actual[0].GetMatch().GetHeaders()[0].GetSafeRegexMatch().Regex)
				assert.Equal(1, len(actual[0].GetRoute().GetWeightedClusters().Clusters))
				assert.Equal(uint32(100), actual[0].GetRoute().GetWeightedClusters().TotalWeight.GetValue())
				assert.Equal("testCluster-local", actual[0].GetRoute().GetWeightedClusters().Clusters[0].Name)
				assert.Equal(uint32(100), actual[0].GetRoute().GetWeightedClusters().Clusters[0].Weight.GetValue())
				assert.NotNil(actual[0].TypedPerFilterConfig)
			},
		},
		{
			name: "invalid route rule without Rule.AllowedServiceAccounts",
			inputRules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
							Path:          "/hello",
							PathMatchType: trafficpolicy.PathMatchRegex,
							Methods:       []string{"GET"},
							Headers:       map[string]string{"hello": "world"},
						},
						WeightedClusters: mapset.NewSet(testWeightedCluster),
					},
					AllowedServiceAccounts: nil,
				},
			},
			expectFunc: func(actual []*xds_route.Route) {
				assert.Equal(0, len(actual))
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			actual := buildInboundRoutes(tc.inputRules)
			tc.expectFunc(actual)
		})
	}
}

func TestBuildOutboundRoutes(t *testing.T) {
	assert := tassert.New(t)

	testWeightedCluster := service.WeightedCluster{
		ClusterName: "testCluster",
		Weight:      100,
	}
	input := []*trafficpolicy.RouteWeightedClusters{
		{
			HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
				Path:          "/hello",
				PathMatchType: trafficpolicy.PathMatchRegex,
				Methods:       []string{"GET"},
				Headers:       map[string]string{"hello": "world"},
			},
			WeightedClusters: mapset.NewSet(testWeightedCluster),
		},
	}
	actual := buildOutboundRoutes(input)
	assert.Equal(1, len(actual))
	assert.Equal(".*", actual[0].GetMatch().GetSafeRegex().Regex)
	assert.Equal(".*", actual[0].GetMatch().GetHeaders()[0].GetSafeRegexMatch().Regex)
	assert.Equal(1, len(actual[0].GetRoute().GetWeightedClusters().Clusters))
	assert.Equal(uint32(100), actual[0].GetRoute().GetWeightedClusters().TotalWeight.GetValue())
	assert.Equal("testCluster", actual[0].GetRoute().GetWeightedClusters().Clusters[0].Name)
	assert.Equal(uint32(100), actual[0].GetRoute().GetWeightedClusters().Clusters[0].Weight.GetValue())
}

func TestBuildRoute(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name             string
		weightedClusters mapset.Set
		totalWeight      int
		direction        Direction
		path             string
		pathMatchType    trafficpolicy.PathMatchType
		method           string
		headersMap       map[string]string
		expectedRoute    *xds_route.Route
	}{
		{
			name:          "outbound route for regex path match",
			path:          "/somepath",
			pathMatchType: trafficpolicy.PathMatchRegex,
			method:        "GET",
			headersMap:    map[string]string{"header1": "header1-val", "header2": "header2-val"},
			totalWeight:   100,
			direction:     OutboundRoute,
			weightedClusters: mapset.NewSetFromSlice([]interface{}{
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 30},
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2"), Weight: 70},
			}),

			expectedRoute: &xds_route.Route{
				Match: &xds_route.RouteMatch{
					PathSpecifier: &xds_route.RouteMatch_SafeRegex{
						SafeRegex: &xds_matcher.RegexMatcher{
							EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
							Regex:      "/somepath",
						},
					},
					Headers: []*xds_route.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
								SafeRegexMatch: &xds_matcher.RegexMatcher{
									EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
									Regex:      "GET",
								},
							},
						},
						{
							Name: "header1",
							HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
								SafeRegexMatch: &xds_matcher.RegexMatcher{
									EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
									Regex:      "header1-val",
								},
							},
						},
						{
							Name: "header2",
							HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
								SafeRegexMatch: &xds_matcher.RegexMatcher{
									EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
									Regex:      "header2-val",
								},
							},
						},
					},
				},
				Action: &xds_route.Route_Route{
					Route: &xds_route.RouteAction{
						ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
							WeightedClusters: &xds_route.WeightedCluster{
								Clusters: []*xds_route.WeightedCluster_ClusterWeight{
									{
										Name:   "osm/bookstore-1",
										Weight: &wrappers.UInt32Value{Value: 30},
									},
									{
										Name:   "osm/bookstore-2",
										Weight: &wrappers.UInt32Value{Value: 70},
									},
								},
								TotalWeight: &wrappers.UInt32Value{Value: 100},
							},
						},
					},
				},
			},
		},
		{
			name:          "inbound route for regex path match",
			path:          "/somepath",
			pathMatchType: trafficpolicy.PathMatchRegex,
			method:        "GET",
			headersMap:    map[string]string{"header1": "header1-val", "header2": "header2-val"},
			totalWeight:   100,
			direction:     InboundRoute,
			weightedClusters: mapset.NewSetFromSlice([]interface{}{
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 100},
			}),

			expectedRoute: &xds_route.Route{
				Match: &xds_route.RouteMatch{
					PathSpecifier: &xds_route.RouteMatch_SafeRegex{
						SafeRegex: &xds_matcher.RegexMatcher{
							EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
							Regex:      "/somepath",
						},
					},
					Headers: []*xds_route.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
								SafeRegexMatch: &xds_matcher.RegexMatcher{
									EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
									Regex:      "GET",
								},
							},
						},
						{
							Name: "header1",
							HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
								SafeRegexMatch: &xds_matcher.RegexMatcher{
									EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
									Regex:      "header1-val",
								},
							},
						},
						{
							Name: "header2",
							HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
								SafeRegexMatch: &xds_matcher.RegexMatcher{
									EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
									Regex:      "header2-val",
								},
							},
						},
					},
				},
				Action: &xds_route.Route_Route{
					Route: &xds_route.RouteAction{
						ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
							WeightedClusters: &xds_route.WeightedCluster{
								Clusters: []*xds_route.WeightedCluster_ClusterWeight{
									{
										Name:   "osm/bookstore-1-local",
										Weight: &wrappers.UInt32Value{Value: 100},
									},
								},
								TotalWeight: &wrappers.UInt32Value{Value: 100},
							},
						},
					},
				},
			},
		},
		{
			name:          "inbound route for exact path match",
			path:          "/somepath",
			pathMatchType: trafficpolicy.PathMatchExact,
			method:        "GET",
			headersMap:    nil,
			totalWeight:   100,
			direction:     InboundRoute,
			weightedClusters: mapset.NewSetFromSlice([]interface{}{
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 100},
			}),

			expectedRoute: &xds_route.Route{
				Match: &xds_route.RouteMatch{
					PathSpecifier: &xds_route.RouteMatch_Path{
						Path: "/somepath",
					},
					Headers: []*xds_route.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
								SafeRegexMatch: &xds_matcher.RegexMatcher{
									EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
									Regex:      "GET",
								},
							},
						},
					},
				},
				Action: &xds_route.Route_Route{
					Route: &xds_route.RouteAction{
						ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
							WeightedClusters: &xds_route.WeightedCluster{
								Clusters: []*xds_route.WeightedCluster_ClusterWeight{
									{
										Name:   "osm/bookstore-1-local",
										Weight: &wrappers.UInt32Value{Value: 100},
									},
								},
								TotalWeight: &wrappers.UInt32Value{Value: 100},
							},
						},
					},
				},
			},
		},
		{
			name:          "inbound route for prefix path match",
			path:          "/somepath",
			pathMatchType: trafficpolicy.PathMatchPrefix,
			method:        "GET",
			headersMap:    nil,
			totalWeight:   100,
			direction:     InboundRoute,
			weightedClusters: mapset.NewSetFromSlice([]interface{}{
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 100},
			}),

			expectedRoute: &xds_route.Route{
				Match: &xds_route.RouteMatch{
					PathSpecifier: &xds_route.RouteMatch_Prefix{
						Prefix: "/somepath",
					},
					Headers: []*xds_route.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
								SafeRegexMatch: &xds_matcher.RegexMatcher{
									EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
									Regex:      "GET",
								},
							},
						},
					},
				},
				Action: &xds_route.Route_Route{
					Route: &xds_route.RouteAction{
						ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
							WeightedClusters: &xds_route.WeightedCluster{
								Clusters: []*xds_route.WeightedCluster_ClusterWeight{
									{
										Name:   "osm/bookstore-1-local",
										Weight: &wrappers.UInt32Value{Value: 100},
									},
								},
								TotalWeight: &wrappers.UInt32Value{Value: 100},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := buildRoute(tc.pathMatchType, tc.path, tc.method, tc.headersMap, tc.weightedClusters, tc.totalWeight, tc.direction)

			// Assert route.Match
			assert.Equal(tc.expectedRoute.Match.PathSpecifier, actual.Match.PathSpecifier)
			assert.ElementsMatch(tc.expectedRoute.Match.Headers, actual.Match.Headers)

			// Assert route.Action
			assert.Equal(tc.expectedRoute.Action, actual.Action)
		})
	}
}

func TestBuildWeightedCluster(t *testing.T) {
	assert := tassert.New(t)

	weightedClusters := mapset.NewSetFromSlice([]interface{}{
		service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 30},
		service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2"), Weight: 70},
	})

	testCases := []struct {
		name             string
		weightedClusters mapset.Set
		totalWeight      int
		direction        Direction
	}{
		{
			name:             "outbound",
			weightedClusters: weightedClusters,
			totalWeight:      100,
			direction:        OutboundRoute,
		},
		{
			name:             "inbound",
			weightedClusters: weightedClusters,
			totalWeight:      100,
			direction:        InboundRoute,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := buildWeightedCluster(tc.weightedClusters, tc.totalWeight, tc.direction)
			assert.Len(actual.Clusters, 2)
			assert.EqualValues(uint32(tc.totalWeight), actual.TotalWeight.GetValue())
		})
	}
}

func TestSanitizeHTTPMethods(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                   string
		allowedMethods         []string
		expectedAllowedMethods []string
		direction              Direction
	}{
		{
			name:                   "returns unique list of allowed methods",
			allowedMethods:         []string{"GET", "POST", "PUT", "POST", "GET", "GET"},
			expectedAllowedMethods: []string{"GET", "POST", "PUT"},
		},
		{
			name:                   "returns wildcard allowed method (*)",
			allowedMethods:         []string{"GET", "POST", "PUT", "POST", "GET", "GET", "*"},
			expectedAllowedMethods: []string{"*"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := sanitizeHTTPMethods(tc.allowedMethods)
			assert.Equal(tc.expectedAllowedMethods, actual)
		})
	}
}

func TestNewRouteConfigurationStub(t *testing.T) {
	assert := tassert.New(t)

	testName := "testing"
	actual := NewRouteConfigurationStub(testName)

	assert.Equal(testName, actual.Name)
	assert.Nil(actual.VirtualHosts)
	assert.False(actual.ValidateClusters.Value)
}

func TestGetRegexForMethod(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wildcard HTTP method correctly translates to a match all regex",
			input:    "*",
			expected: constants.RegexMatchAll,
		},
		{
			name:     "non wildcard HTTP method correctly translates to its corresponding regex",
			input:    "GET",
			expected: "GET",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getRegexForMethod(tc.input)
			assert.Equal(tc.expected, actual)
		})
	}
}

func TestGetHeadersForRoute(t *testing.T) {
	assert := tassert.New(t)

	userAgentHeader := "user-agent"

	// Returns a list of HeaderMatcher for a route
	routePolicy := trafficpolicy.HTTPRouteMatch{
		Path:          "/books-bought",
		PathMatchType: trafficpolicy.PathMatchRegex,
		Methods:       []string{"GET", "POST"},
		Headers: map[string]string{
			userAgentHeader: "This is a test header",
		},
	}
	actual := getHeadersForRoute(routePolicy.Methods[0], routePolicy.Headers)
	assert.Equal(2, len(actual))
	assert.Equal(MethodHeaderKey, actual[0].Name)
	assert.Equal(routePolicy.Methods[0], actual[0].GetSafeRegexMatch().Regex)
	assert.Equal(userAgentHeader, actual[1].Name)
	assert.Equal(routePolicy.Headers[userAgentHeader], actual[1].GetSafeRegexMatch().Regex)

	// Returns only one HeaderMatcher for a route
	routePolicy = trafficpolicy.HTTPRouteMatch{
		Path:          "/books-bought",
		PathMatchType: trafficpolicy.PathMatchRegex,
		Methods:       []string{"GET", "POST"},
	}
	actual = getHeadersForRoute(routePolicy.Methods[1], routePolicy.Headers)
	assert.Equal(1, len(actual))
	assert.Equal(MethodHeaderKey, actual[0].Name)
	assert.Equal(routePolicy.Methods[1], actual[0].GetSafeRegexMatch().Regex)

	// Returns only one HeaderMatcher for a route ignoring the host
	routePolicy = trafficpolicy.HTTPRouteMatch{
		Path:          "/books-bought",
		PathMatchType: trafficpolicy.PathMatchRegex,
		Methods:       []string{"GET", "POST"},
		Headers: map[string]string{
			"user-agent": tests.HTTPUserAgent,
		},
	}
	actual = getHeadersForRoute(routePolicy.Methods[0], routePolicy.Headers)
	assert.Equal(2, len(actual))
	assert.Equal(MethodHeaderKey, actual[0].Name)
	assert.Equal(routePolicy.Methods[0], actual[0].GetSafeRegexMatch().Regex)
}

func TestLen(t *testing.T) {
	assert := tassert.New(t)

	clusters := clusterWeightByName([]*xds_route.WeightedCluster_ClusterWeight{
		{
			Name:   "hello1",
			Weight: &wrappers.UInt32Value{Value: uint32(50)},
		},
		{
			Name:   "hello2",
			Weight: &wrappers.UInt32Value{Value: uint32(50)},
		},
	})

	actual := clusters.Len()
	assert.Equal(2, actual)
}

func TestSwap(t *testing.T) {
	assert := tassert.New(t)

	clusters := clusterWeightByName([]*xds_route.WeightedCluster_ClusterWeight{
		{
			Name:   "hello1",
			Weight: &wrappers.UInt32Value{Value: uint32(20)},
		},
		{
			Name:   "hello2",
			Weight: &wrappers.UInt32Value{Value: uint32(50)},
		},
		{
			Name:   "hello3",
			Weight: &wrappers.UInt32Value{Value: uint32(30)},
		},
	})

	expected := clusterWeightByName([]*xds_route.WeightedCluster_ClusterWeight{
		{
			Name:   "hello1",
			Weight: &wrappers.UInt32Value{Value: uint32(20)},
		},
		{
			Name:   "hello3",
			Weight: &wrappers.UInt32Value{Value: uint32(30)},
		},
		{
			Name:   "hello2",
			Weight: &wrappers.UInt32Value{Value: uint32(50)},
		},
	})

	clusters.Swap(1, 2)
	assert.Equal(expected, clusters)
}

func TestLess(t *testing.T) {
	assert := tassert.New(t)

	clusters := clusterWeightByName([]*xds_route.WeightedCluster_ClusterWeight{
		{
			Name:   "cluster1",
			Weight: &wrappers.UInt32Value{Value: uint32(20)},
		},
		{
			Name:   "cluster1",
			Weight: &wrappers.UInt32Value{Value: uint32(50)},
		},
		{
			Name:   "cluster2",
			Weight: &wrappers.UInt32Value{Value: uint32(30)},
		},
	})

	actual := clusters.Less(1, 2)
	assert.True(actual)
	actual = clusters.Less(2, 1)
	assert.False(actual)
	actual = clusters.Less(0, 1)
	assert.True(actual)
	actual = clusters.Less(1, 0)
	assert.False(actual)
}
