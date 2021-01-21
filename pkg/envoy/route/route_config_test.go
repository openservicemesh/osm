package route

import (
	"testing"

	set "github.com/deckarep/golang-set"
	tassert "github.com/stretchr/testify/assert"

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
			actual := BuildRouteConfiguration(tc.inbound, tc.outbound)
			assert.Equal(tc.expectedRouteConfigLen, len(actual))
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
			expectedName: "inbound_virtualHost|host",
		},
		{
			name:         "outbound virtual host",
			namePrefix:   outboundVirtualHost,
			host:         "host",
			domains:      []string{"domain1", "domain2"},
			expectedName: "outbound_virtualHost|host",
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
	input := []*trafficpolicy.Rule{
		{
			Route: trafficpolicy.RouteWeightedClusters{
				HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
					PathRegex: "/hello",
					Methods:   []string{"GET"},
					Headers:   map[string]string{"hello": "world"},
				},
				WeightedClusters: set.NewSet(testWeightedCluster),
			},
		},
	}
	actual := buildInboundRoutes(input)
	assert.Equal(1, len(actual))
	assert.Equal("/hello", actual[0].GetMatch().GetSafeRegex().Regex)
	assert.Equal("GET", actual[0].GetMatch().GetHeaders()[0].GetSafeRegexMatch().Regex)
	assert.Equal(1, len(actual[0].GetRoute().GetWeightedClusters().Clusters))
	assert.Equal(uint32(100), actual[0].GetRoute().GetWeightedClusters().TotalWeight.GetValue())
	assert.Equal("testCluster-local", actual[0].GetRoute().GetWeightedClusters().Clusters[0].Name)
	assert.Equal(uint32(100), actual[0].GetRoute().GetWeightedClusters().Clusters[0].Weight.GetValue())
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
