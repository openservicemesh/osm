package route

import (
	"fmt"
	"testing"

	set "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
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
					WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
				},
				AllowedServiceAccounts: set.NewSet(tests.BookbuyerServiceAccount),
			},
			{
				Route: trafficpolicy.RouteWeightedClusters{
					HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
					WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
				},
				AllowedServiceAccounts: set.NewSet(tests.BookbuyerServiceAccount),
			},
		},
	}

	testOutbound := &trafficpolicy.OutboundTrafficPolicy{
		Name:      "bookstore-v1",
		Hostnames: tests.BookstoreV1Hostnames,
		Routes: []*trafficpolicy.RouteWeightedClusters{
			{
				HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
					PathRegex: "/some-path",
					Methods:   []string{"GET"},
				},
				WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
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
							PathRegex: "/hello",
							Methods:   []string{"GET"},
							Headers:   map[string]string{"hello": "world"},
						},
						WeightedClusters: set.NewSet(testWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSetFromSlice(
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
							PathRegex: "/hello",
							Methods:   []string{"GET"},
							Headers:   map[string]string{"hello": "world"},
						},
						WeightedClusters: set.NewSet(testWeightedCluster),
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
				PathRegex: "/hello",
				Methods:   []string{"GET"},
				Headers:   map[string]string{"hello": "world"},
			},
			WeightedClusters: set.NewSet(testWeightedCluster),
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
	weightedClusters := set.NewSetFromSlice([]interface{}{
		service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 30},
		service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2"), Weight: 70},
	})
	testCases := []struct {
		name             string
		weightedClusters set.Set
		totalWeight      int
		direction        Direction
		pathRegex        string
		method           string
		headersMap       map[string]string
	}{
		{
			name:             "default",
			pathRegex:        "/somepath",
			method:           "GET",
			headersMap:       map[string]string{"hello": "goodbye", "header1": "another-header"},
			totalWeight:      100,
			direction:        OutboundRoute,
			weightedClusters: weightedClusters,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := buildRoute(tc.pathRegex, tc.method, tc.headersMap, tc.weightedClusters, tc.totalWeight, tc.direction)
			assert.EqualValues(tc.pathRegex, actual.Match.GetSafeRegex().Regex)
			numFound := 0
			for k, v := range tc.headersMap {
				//assert that k is in actual.Match.Headers and the v is the same
				for _, actualHeader := range actual.Match.Headers {
					if actualHeader.Name == k {
						assert.Equal(v, actualHeader.GetSafeRegexMatch().Regex)
						numFound = numFound + 1
					}
				}
			}
			foundMethod := false
			for _, actualHeader := range actual.Match.Headers {
				if actualHeader.Name == ":method" {
					assert.Equal(tc.method, actualHeader.GetSafeRegexMatch().Regex)
					foundMethod = true
					break
				}
			}
			assert.Equal(true, foundMethod)
			assert.Equal(len(tc.headersMap), numFound)
			assert.Equal(2, len(actual.GetRoute().GetWeightedClusters().Clusters))
		})
	}
}
func TestBuildWeightedCluster(t *testing.T) {
	assert := tassert.New(t)

	weightedClusters := set.NewSetFromSlice([]interface{}{
		service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1"), Weight: 30},
		service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2"), Weight: 70},
	})

	testCases := []struct {
		name             string
		weightedClusters set.Set
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
		PathRegex: "/books-bought",
		Methods:   []string{"GET", "POST"},
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
		PathRegex: "/books-bought",
		Methods:   []string{"GET", "POST"},
	}
	actual = getHeadersForRoute(routePolicy.Methods[1], routePolicy.Headers)
	assert.Equal(1, len(actual))
	assert.Equal(MethodHeaderKey, actual[0].Name)
	assert.Equal(routePolicy.Methods[1], actual[0].GetSafeRegexMatch().Regex)

	// Returns only one HeaderMatcher for a route ignoring the host
	routePolicy = trafficpolicy.HTTPRouteMatch{
		PathRegex: "/books-bought",
		Methods:   []string{"GET", "POST"},
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
