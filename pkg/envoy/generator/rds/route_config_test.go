package rds

import (
	"fmt"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

var (
	thresholdUintVal         uint32 = 3
	thresholdTimeoutDuration        = metav1.Duration{Duration: time.Duration(5 * time.Second)}
	thresholdBackoffDuration        = metav1.Duration{Duration: time.Duration(1 * time.Second)}
)

func TestBuildVirtualHostStub(t *testing.T) {
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
			host:         httpHostHeaderKey,
			domains:      []string{"domain1", "domain2"},
			expectedName: "inbound_virtual-host|host",
		},
		{
			name:         "outbound virtual host",
			namePrefix:   outboundVirtualHost,
			host:         httpHostHeaderKey,
			domains:      []string{"domain1", "domain2"},
			expectedName: "outbound_virtual-host|host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := buildVirtualHostStub(tc.namePrefix, tc.host, tc.domains)
			assert.Equal(tc.expectedName, actual.Name)
			assert.Equal(tc.domains, actual.Domains)
		})
	}
}
func TestBuildInboundRoutes(t *testing.T) {
	testWeightedCluster := service.WeightedCluster{
		ClusterName: "default/testCluster|80|local",
		Weight:      100,
	}
	testCases := []struct {
		name       string
		inputRules []*trafficpolicy.Rule
		expectFunc func(assert *tassert.Assertions, actual []*xds_route.Route)
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
					AllowedPrincipals: mapset.NewSet("foo.bar.cluster.local"),
				},
			},
			expectFunc: func(assert *tassert.Assertions, actual []*xds_route.Route) {
				assert.Equal(1, len(actual))
				assert.Equal("/hello", actual[0].GetMatch().GetSafeRegex().Regex)
				assert.Equal("GET", actual[0].GetMatch().GetHeaders()[0].GetSafeRegexMatch().Regex)
				assert.Equal(1, len(actual[0].GetRoute().GetWeightedClusters().Clusters))
				assert.Equal(uint32(100), actual[0].GetRoute().GetWeightedClusters().TotalWeight.GetValue())
				assert.Equal("default/testCluster|80|local", actual[0].GetRoute().GetWeightedClusters().Clusters[0].Name)
				assert.Equal(uint32(100), actual[0].GetRoute().GetWeightedClusters().Clusters[0].Weight.GetValue())
				assert.NotNil(actual[0].TypedPerFilterConfig)
			},
		},
		{
			name: "invalid route rule without Rule.AllowedPrincipals",
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
					AllowedPrincipals: nil,
				},
			},
			expectFunc: func(assert *tassert.Assertions, actual []*xds_route.Route) {
				assert.Equal(0, len(actual))
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			actual := buildInboundRoutes(tc.inputRules)
			tc.expectFunc(tassert.New(t), actual)
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
			RetryPolicy: &policyv1alpha1.RetryPolicySpec{
				RetryOn:                  "4xx",
				PerTryTimeout:            &thresholdTimeoutDuration,
				NumRetries:               &thresholdUintVal,
				RetryBackoffBaseInterval: &thresholdBackoffDuration,
			},
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
	retry := &xds_route.RetryPolicy{
		RetryOn:       "4xx",
		PerTryTimeout: durationpb.New(thresholdTimeoutDuration.Duration),
		NumRetries:    &wrapperspb.UInt32Value{Value: thresholdUintVal},
		RetryBackOff: &xds_route.RetryPolicy_RetryBackOff{
			BaseInterval: durationpb.New(thresholdBackoffDuration.Duration),
		},
	}
	assert.Equal(retry, actual[0].GetRoute().GetRetryPolicy())
}

func TestBuildRoute(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name          string
		route         trafficpolicy.RouteWeightedClusters
		method        string
		expectedRoute *xds_route.Route
	}{
		{
			name: "outbound route for regex path match",
			route: trafficpolicy.RouteWeightedClusters{
				HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
					PathMatchType: trafficpolicy.PathMatchRegex,
					Path:          "/somepath",
					Headers:       map[string]string{"header1": "header1-val", "header2": "header2-val"},
				},
				WeightedClusters: mapset.NewSetFromSlice([]interface{}{
					service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1|80|local"), Weight: 30},
					service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2|80|local"), Weight: 70}}),
				RetryPolicy: &policyv1alpha1.RetryPolicySpec{
					RetryOn:                  "4xx",
					PerTryTimeout:            &thresholdTimeoutDuration,
					NumRetries:               &thresholdUintVal,
					RetryBackoffBaseInterval: &thresholdBackoffDuration,
				},
			},
			method: "GET",
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
										Name:   "osm/bookstore-1|80|local",
										Weight: &wrappers.UInt32Value{Value: 30},
									},
									{
										Name:   "osm/bookstore-2|80|local",
										Weight: &wrappers.UInt32Value{Value: 70},
									},
								},
								TotalWeight: &wrappers.UInt32Value{Value: 100},
							},
						},
						Timeout: &duration.Duration{Seconds: 0},
						RetryPolicy: &xds_route.RetryPolicy{
							RetryOn:       "4xx",
							PerTryTimeout: durationpb.New(thresholdTimeoutDuration.Duration),
							NumRetries:    &wrapperspb.UInt32Value{Value: thresholdUintVal},
							RetryBackOff: &xds_route.RetryPolicy_RetryBackOff{
								BaseInterval: durationpb.New(thresholdBackoffDuration.Duration),
							},
						},
					},
				},
			},
		},
		{
			name: "inbound route for regex path match",
			route: trafficpolicy.RouteWeightedClusters{
				HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
					PathMatchType: trafficpolicy.PathMatchRegex,
					Path:          "/somepath",
					Headers:       map[string]string{"header1": "header1-val", "header2": "header2-val"},
				},
				WeightedClusters: mapset.NewSetFromSlice([]interface{}{
					service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1|80|local"), Weight: 100}}),
				RetryPolicy: &policyv1alpha1.RetryPolicySpec{
					RetryOn: "4xx",
				},
			},
			method: "GET",
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
										Name:   "osm/bookstore-1|80|local",
										Weight: &wrappers.UInt32Value{Value: 100},
									},
								},
								TotalWeight: &wrappers.UInt32Value{Value: 100},
							},
						},
						Timeout: &duration.Duration{Seconds: 0},
						RetryPolicy: &xds_route.RetryPolicy{
							RetryOn: "4xx",
						},
					},
				},
			},
		},
		{
			name: "inbound route for exact path match",
			route: trafficpolicy.RouteWeightedClusters{
				HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
					PathMatchType: trafficpolicy.PathMatchExact,
					Path:          "/somepath",
					Headers:       nil,
				},
				WeightedClusters: mapset.NewSetFromSlice([]interface{}{
					service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1|80|local"), Weight: 100}}),
				RetryPolicy: nil,
			},
			method: "GET",
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
										Name:   "osm/bookstore-1|80|local",
										Weight: &wrappers.UInt32Value{Value: 100},
									},
								},
								TotalWeight: &wrappers.UInt32Value{Value: 100},
							},
						},
						Timeout:     &duration.Duration{Seconds: 0},
						RetryPolicy: nil,
					},
				},
			},
		},
		{
			name: "inbound route for prefix path match",
			route: trafficpolicy.RouteWeightedClusters{
				HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
					PathMatchType: trafficpolicy.PathMatchPrefix,
					Path:          "/somepath",
					Headers:       nil,
				},
				WeightedClusters: mapset.NewSetFromSlice([]interface{}{
					service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1|80|local"), Weight: 100}}),
				RetryPolicy: nil,
			},
			method: "GET",
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
										Name:   "osm/bookstore-1|80|local",
										Weight: &wrappers.UInt32Value{Value: 100},
									},
								},
								TotalWeight: &wrappers.UInt32Value{Value: 100},
							},
						},
						Timeout:     &duration.Duration{Seconds: 0},
						RetryPolicy: nil,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := buildRoute(tc.route, tc.method)
			// Assert route.Match
			assert.Equal(tc.expectedRoute.Match.PathSpecifier, actual.Match.PathSpecifier)
			assert.ElementsMatch(tc.expectedRoute.Match.Headers, actual.Match.Headers)
			// Assert route.Action
			assert.Equal(tc.expectedRoute.Action, actual.Action)
		})
	}
}

func TestBuildWeightedCluster(t *testing.T) {
	testCases := []struct {
		name                string
		weightedClusters    mapset.Set
		expectedClusters    int
		expectedTotalWeight int
	}{
		{
			name: "multiple valid clusters",
			weightedClusters: mapset.NewSetFromSlice([]interface{}{
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1|80|local"), Weight: 30},
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2|80|local"), Weight: 70},
			}),
			expectedClusters:    2,
			expectedTotalWeight: 100,
		},
		{
			name: "total cluster weight is invalid (< 1)",
			weightedClusters: mapset.NewSetFromSlice([]interface{}{
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-1|80|local"), Weight: 0},
				service.WeightedCluster{ClusterName: service.ClusterName("osm/bookstore-2|80|local"), Weight: 0},
			}),
			expectedClusters: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := buildWeightedCluster(tc.weightedClusters)
			if tc.expectedClusters == 0 {
				assert.Nil(actual)
				return
			}

			assert.Len(actual.Clusters, tc.expectedClusters)
			assert.EqualValues(tc.expectedTotalWeight, actual.TotalWeight.GetValue())
		})
	}
}

func TestBuildRetryPolicy(t *testing.T) {
	testCases := []struct {
		name        string
		retryPolicy *policyv1alpha1.RetryPolicySpec
		expRetry    *xds_route.RetryPolicy
	}{
		{
			name:        "no retry",
			retryPolicy: nil,
			expRetry:    nil,
		},
		{
			name: "valid retry policy",
			retryPolicy: &policyv1alpha1.RetryPolicySpec{
				RetryOn: "2xx",
			},
			expRetry: &xds_route.RetryPolicy{
				RetryOn: "2xx",
			},
		},
		{
			name: "valid retry policy with all fields",
			retryPolicy: &policyv1alpha1.RetryPolicySpec{
				RetryOn:                  "2xx",
				PerTryTimeout:            &thresholdTimeoutDuration,
				NumRetries:               &thresholdUintVal,
				RetryBackoffBaseInterval: &thresholdBackoffDuration,
			},
			expRetry: &xds_route.RetryPolicy{
				RetryOn:       "2xx",
				PerTryTimeout: durationpb.New(thresholdTimeoutDuration.Duration),
				NumRetries: &wrapperspb.UInt32Value{
					Value: thresholdUintVal,
				},
				RetryBackOff: &xds_route.RetryPolicy_RetryBackOff{
					BaseInterval: durationpb.New(thresholdBackoffDuration.Duration),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := buildRetryPolicy(tc.retryPolicy)
			assert.Equal(tc.expRetry, actual)
		})
	}
}

func TestSanitizeHTTPMethods(t *testing.T) {
	testCases := []struct {
		name                   string
		allowedMethods         []string
		expectedAllowedMethods []string
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
			assert := tassert.New(t)

			actual := sanitizeHTTPMethods(tc.allowedMethods)
			assert.Equal(tc.expectedAllowedMethods, actual)
		})
	}
}

func TestNewRouteConfigurationStub(t *testing.T) {
	assert := tassert.New(t)

	testName := "testing"
	actual := newRouteConfigurationStub(testName)

	assert.Equal(testName, actual.Name)
	assert.Nil(actual.VirtualHosts)
	assert.False(actual.ValidateClusters.Value)
}

func TestGetRegexForMethod(t *testing.T) {
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
			assert := tassert.New(t)

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
	assert.Equal(methodHeaderKey, actual[0].Name)
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
	assert.Equal(methodHeaderKey, actual[0].Name)
	assert.Equal(routePolicy.Methods[1], actual[0].GetSafeRegexMatch().Regex)

	// Returns only HeaderMatcher for the method and host header (:authority)
	routePolicy = trafficpolicy.HTTPRouteMatch{
		Path:          "/books-bought",
		PathMatchType: trafficpolicy.PathMatchRegex,
		Methods:       []string{"GET", "POST"},
		Headers: map[string]string{
			"host": tests.HTTPHostHeader,
		},
	}
	actual = getHeadersForRoute(routePolicy.Methods[0], routePolicy.Headers)
	assert.Equal(2, len(actual))
	assert.Equal(methodHeaderKey, actual[0].Name)
	assert.Equal(routePolicy.Methods[0], actual[0].GetSafeRegexMatch().Regex)
	assert.Equal(authorityHeaderKey, actual[1].Name)
	assert.Equal(tests.HTTPHostHeader, actual[1].GetSafeRegexMatch().Regex)
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

func TestBuildEgressRoute(t *testing.T) {
	testCases := []struct {
		name           string
		routingRules   []*trafficpolicy.EgressHTTPRoutingRule
		expectedRoutes []*xds_route.Route
	}{
		{
			name:           "no routing rules",
			routingRules:   nil,
			expectedRoutes: nil,
		},
		{
			name: "multiple routing rules",
			routingRules: []*trafficpolicy.EgressHTTPRoutingRule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
							PathMatchType: trafficpolicy.PathMatchRegex,
							Path:          "/foo",
							Methods:       []string{"GET"},
						},
						WeightedClusters: mapset.NewSetFromSlice([]interface{}{
							service.WeightedCluster{ClusterName: "foo.com:80", Weight: 100},
						}),
						RetryPolicy: nil,
					},
				},
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
							PathMatchType: trafficpolicy.PathMatchRegex,
							Path:          "/bar",
							Methods:       []string{"POST"},
						},
						WeightedClusters: mapset.NewSetFromSlice([]interface{}{
							service.WeightedCluster{ClusterName: "foo.com:80", Weight: 100},
						}),
						RetryPolicy: nil,
					},
				},
			},
			expectedRoutes: []*xds_route.Route{
				{
					Match: &xds_route.RouteMatch{
						PathSpecifier: &xds_route.RouteMatch_SafeRegex{
							SafeRegex: &xds_matcher.RegexMatcher{
								EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
								Regex:      "/foo",
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
						},
					},
					Action: &xds_route.Route_Route{
						Route: &xds_route.RouteAction{
							ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
								WeightedClusters: &xds_route.WeightedCluster{
									Clusters: []*xds_route.WeightedCluster_ClusterWeight{
										{
											Name:   "foo.com:80",
											Weight: &wrappers.UInt32Value{Value: 100},
										},
									},
									TotalWeight: &wrappers.UInt32Value{Value: 100},
								},
							},
							Timeout:     &duration.Duration{Seconds: 0},
							RetryPolicy: nil,
						},
					},
				},
				{
					Match: &xds_route.RouteMatch{
						PathSpecifier: &xds_route.RouteMatch_SafeRegex{
							SafeRegex: &xds_matcher.RegexMatcher{
								EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
								Regex:      "/bar",
							},
						},
						Headers: []*xds_route.HeaderMatcher{
							{
								Name: ":method",
								HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
									SafeRegexMatch: &xds_matcher.RegexMatcher{
										EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
										Regex:      "POST",
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
											Name:   "foo.com:80",
											Weight: &wrappers.UInt32Value{Value: 100},
										},
									},
									TotalWeight: &wrappers.UInt32Value{Value: 100},
								},
							},
							Timeout:     &duration.Duration{Seconds: 0},
							RetryPolicy: nil,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := buildEgressRoutes(tc.routingRules)
			assert.ElementsMatch(tc.expectedRoutes, actual)
		})
	}
}

func TestGetEgressRouteConfigNameForPort(t *testing.T) {
	testCases := []struct {
		name         string
		port         int
		expectedName string
	}{
		{
			name:         "test 1",
			port:         10,
			expectedName: "rds-egress.10",
		},
		{
			name:         "test 2",
			port:         20,
			expectedName: "rds-egress.20",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := GetEgressRouteConfigNameForPort(tc.port)
			assert.Equal(tc.expectedName, actual)
		})
	}
}
