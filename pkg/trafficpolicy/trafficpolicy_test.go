package trafficpolicy

import (
	"testing"

	mapset "github.com/deckarep/golang-set"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	testHTTPRouteMatch = HTTPRouteMatch{
		Path:          "/hello",
		PathMatchType: PathMatchRegex,
		Methods:       []string{"GET"},
		Headers:       map[string]string{"hello": "world"},
	}

	testHTTPRouteMatch2 = HTTPRouteMatch{
		Path:          "/goodbye",
		PathMatchType: PathMatchRegex,
		Methods:       []string{"GET"},
		Headers:       map[string]string{"later": "alligator"},
	}

	testHostnames = []string{"testHostname1", "testHostname2", "testHostname3"}

	testHostnames2 = []string{"testing1", "testing2", "testing3"}

	testWeightedCluster = service.WeightedCluster{
		ClusterName: "testCluster",
		Weight:      100,
	}
	testWeightedCluster2 = service.WeightedCluster{
		ClusterName: "testCluster2",
		Weight:      100,
	}

	testServiceAccount1 = identity.K8sServiceAccount{
		Name:      "testServiceAccount1",
		Namespace: "testNamespace1",
	}

	testServiceAccount2 = identity.K8sServiceAccount{
		Name:      "testServiceAccount2",
		Namespace: "testNamespace2",
	}

	testRoute = RouteWeightedClusters{
		HTTPRouteMatch:   testHTTPRouteMatch,
		WeightedClusters: mapset.NewSet(testWeightedCluster),
	}

	testRoute2 = RouteWeightedClusters{
		HTTPRouteMatch:   testHTTPRouteMatch2,
		WeightedClusters: mapset.NewSet(testWeightedCluster),
	}
)

func TestAddRule(t *testing.T) {
	testCases := []struct {
		name                  string
		existingRules         []*Rule
		allowedServiceAccount identity.K8sServiceAccount
		route                 RouteWeightedClusters
		expectedRules         []*Rule
	}{
		{
			name:                  "rule for route does not exist",
			existingRules:         []*Rule{},
			allowedServiceAccount: testServiceAccount1,
			route:                 testRoute,
			expectedRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
				},
			},
		},
		{
			name: "rule exists for route but not for given service account",
			existingRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
				},
			},
			allowedServiceAccount: testServiceAccount2,
			route:                 testRoute,
			expectedRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity(), testServiceAccount2.ToServiceIdentity()),
				},
			},
		},
		{
			name: "rule exists for route and for given service account",
			existingRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
				},
			},
			allowedServiceAccount: testServiceAccount1,
			route:                 testRoute,
			expectedRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			inboundPolicy := newTestInboundPolicy(tc.name, tc.existingRules)
			inboundPolicy.AddRule(tc.route, tc.allowedServiceAccount.ToServiceIdentity())
			assert.Equal(tc.expectedRules, inboundPolicy.Rules)
		})
	}
}

func TestAddRoute(t *testing.T) {
	testCases := []struct {
		name                  string
		existingRoutes        []*RouteWeightedClusters
		expectedRoutes        []*RouteWeightedClusters
		givenRouteMatch       HTTPRouteMatch
		givenWeightedClusters []service.WeightedCluster
		givenRetryPolicy      *v1alpha1.RetryPolicySpec
		expectedErr           bool
	}{
		{
			name:                  "no routes exist",
			existingRoutes:        []*RouteWeightedClusters{},
			givenRouteMatch:       testHTTPRouteMatch,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster},
			givenRetryPolicy:      &v1alpha1.RetryPolicySpec{},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: mapset.NewSet(testWeightedCluster),
					RetryPolicy:      &v1alpha1.RetryPolicySpec{},
				},
			},
			expectedErr: false,
		},
		{
			name: "add route to existing routes",
			existingRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: mapset.NewSet(testWeightedCluster),
				},
			},
			givenRouteMatch:       testHTTPRouteMatch2,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster2},
			givenRetryPolicy: &v1alpha1.RetryPolicySpec{
				RetryOn: "5xx",
			},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: mapset.NewSet(testWeightedCluster),
				},
				{
					HTTPRouteMatch:   testHTTPRouteMatch2,
					WeightedClusters: mapset.NewSet(testWeightedCluster2),
					RetryPolicy: &v1alpha1.RetryPolicySpec{
						RetryOn: "5xx",
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "add route with multiple weighted clusters to existing routes",
			existingRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: mapset.NewSet(testWeightedCluster),
				},
			},
			givenRouteMatch:       testHTTPRouteMatch2,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster, testWeightedCluster2},
			givenRetryPolicy: &v1alpha1.RetryPolicySpec{
				RetryOn:       "5xx",
				PerTryTimeout: "5s",
			},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: mapset.NewSet(testWeightedCluster),
				},
				{
					HTTPRouteMatch:   testHTTPRouteMatch2,
					WeightedClusters: mapset.NewSet(testWeightedCluster, testWeightedCluster2),
					RetryPolicy: &v1alpha1.RetryPolicySpec{
						RetryOn:       "5xx",
						PerTryTimeout: "5s",
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "route already exists, same weighted cluster",
			existingRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: mapset.NewSet(testWeightedCluster),
				},
			},
			givenRouteMatch:       testHTTPRouteMatch,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster},
			givenRetryPolicy: &v1alpha1.RetryPolicySpec{
				RetryOn:       "5xx",
				NumRetries:    3,
				PerTryTimeout: "5s",
			},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: mapset.NewSet(testWeightedCluster),
					RetryPolicy: &v1alpha1.RetryPolicySpec{
						RetryOn:       "5xx",
						NumRetries:    3,
						PerTryTimeout: "5s",
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "route already exists, different weighted cluster",
			existingRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: mapset.NewSet(testWeightedCluster),
				},
			},
			givenRouteMatch:       testHTTPRouteMatch,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster2},
			givenRetryPolicy: &v1alpha1.RetryPolicySpec{
				RetryOn:                  "5xx",
				RetryBackoffBaseInterval: "4s",
			},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: mapset.NewSet(testWeightedCluster),
				},
			},
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			outboundPolicy := newTestOutboundPolicy(tc.name, tc.existingRoutes)
			err := outboundPolicy.AddRoute(tc.givenRouteMatch, tc.givenRetryPolicy, tc.givenWeightedClusters...)
			if tc.expectedErr {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
			assert.Equal(tc.expectedRoutes, outboundPolicy.Routes)
		})
	}
}

func TestMergeInboundPolicies(t *testing.T) {
	testRule1 := Rule{
		Route:                    testRoute,
		AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
	}
	testRule2 := Rule{
		Route:                    testRoute2,
		AllowedServiceIdentities: mapset.NewSet(testServiceAccount2.ToServiceIdentity()),
	}
	testRule1Modified := Rule{
		Route: RouteWeightedClusters{
			HTTPRouteMatch: HTTPRouteMatch{
				Path:          "/hello",
				PathMatchType: PathMatchRegex,
				Methods:       []string{"*"},
			},
			WeightedClusters: mapset.NewSet(testWeightedCluster),
		},
	}
	testCases := []struct {
		name            string
		originalInbound []*InboundTrafficPolicy
		newInbound      []*InboundTrafficPolicy
		expectedInbound []*InboundTrafficPolicy
	}{
		{
			name: "hostnames match",
			originalInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2},
				},
			},
			newInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule2},
				},
			},
			expectedInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2},
				},
			},
		},
		{
			name: "hostnames do not match",
			originalInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2},
				},
			},
			newInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames2,
					Rules:     []*Rule{&testRule2},
				},
			},
			expectedInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames2,
					Rules:     []*Rule{&testRule2},
				},
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2},
				},
			},
		},
		{
			name: "hostnames match but rules differ",
			originalInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2},
				},
			},
			newInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1Modified},
				},
			},
			expectedInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2, &testRule1Modified},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := MergeInboundPolicies(false, tc.originalInbound, tc.newInbound...)
			assert.ElementsMatch(tc.expectedInbound, actual)
		})
	}
}

func TestMergeInboundPoliciesWithPartialHostnames(t *testing.T) {
	testRule1 := Rule{
		Route:                    testRoute,
		AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
	}
	testRule2 := Rule{
		Route:                    testRoute2,
		AllowedServiceIdentities: mapset.NewSet(testServiceAccount2.ToServiceIdentity()),
	}
	testRule1Modified := Rule{
		Route: RouteWeightedClusters{
			HTTPRouteMatch: HTTPRouteMatch{
				Path:          "/hello",
				PathMatchType: PathMatchRegex,
				Methods:       []string{"*"},
			},
			WeightedClusters: mapset.NewSet(testWeightedCluster),
		},
	}
	testCases := []struct {
		name            string
		originalInbound []*InboundTrafficPolicy
		newInbound      []*InboundTrafficPolicy
		expectedInbound []*InboundTrafficPolicy
	}{
		{
			name: "hostnames is a subset",
			originalInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2},
				},
			},
			newInbound: []*InboundTrafficPolicy{
				{
					Hostnames: []string{"testHostname1"},
					Rules:     []*Rule{&testRule2},
				},
			},
			expectedInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2},
				},
			},
		},
		{
			name: "hostnames is a subset but rules differ",
			originalInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2},
				},
			},
			newInbound: []*InboundTrafficPolicy{
				{
					Hostnames: []string{"testHostname1"},
					Rules:     []*Rule{&testRule1Modified},
				},
			},
			expectedInbound: []*InboundTrafficPolicy{
				{
					Hostnames: testHostnames,
					Rules:     []*Rule{&testRule1, &testRule2, &testRule1Modified},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := MergeInboundPolicies(true, tc.originalInbound, tc.newInbound...)
			assert.ElementsMatch(actual, tc.expectedInbound)
		})
	}
}

func TestMergeRules(t *testing.T) {
	testCases := []struct {
		name          string
		originalRules []*Rule
		newRules      []*Rule
		expectedRules []*Rule
	}{
		{
			name: "routes match",
			originalRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
				},
			},
			newRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount2.ToServiceIdentity()),
				},
			},
			expectedRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSetWith(testServiceAccount1.ToServiceIdentity(), testServiceAccount2.ToServiceIdentity()),
				},
			},
		},
		{
			name: "routes match but with duplicate allowed service accounts",
			originalRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
				},
			},
			newRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
				},
			},
			expectedRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSetWith(testServiceAccount1.ToServiceIdentity()),
				},
			},
		},
		{
			name: "routes don't match, add rule",
			originalRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
				},
			},
			newRules: []*Rule{
				{
					Route:                    testRoute2,
					AllowedServiceIdentities: mapset.NewSet(testServiceAccount1.ToServiceIdentity()),
				},
			},
			expectedRules: []*Rule{
				{
					Route:                    testRoute,
					AllowedServiceIdentities: mapset.NewSetWith(testServiceAccount1.ToServiceIdentity()),
				},
				{
					Route:                    testRoute2,
					AllowedServiceIdentities: mapset.NewSetWith(testServiceAccount1.ToServiceIdentity()),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := MergeRules(tc.originalRules, tc.newRules)
			assert.ElementsMatch(tc.expectedRules, actual)
		})
	}
}

func TestMergeRouteWeightedClusters(t *testing.T) {
	testCases := []struct {
		name                                         string
		originalRoutes, latestRoutes, expectedRoutes []*RouteWeightedClusters
	}{
		{
			name:           "merge routes with different match conditions",
			originalRoutes: []*RouteWeightedClusters{&testRoute},
			latestRoutes:   []*RouteWeightedClusters{&testRoute2},
			expectedRoutes: []*RouteWeightedClusters{&testRoute, &testRoute2},
		},
		{
			name:           "collapse routes with same match conditions and weighted clusters",
			originalRoutes: []*RouteWeightedClusters{&testRoute},
			latestRoutes:   []*RouteWeightedClusters{&testRoute},
			expectedRoutes: []*RouteWeightedClusters{&testRoute},
		},
		{
			name:           "routes have same match conditions but different weighted clusters, union the weighted clusters",
			originalRoutes: []*RouteWeightedClusters{&testRoute},
			latestRoutes: []*RouteWeightedClusters{{
				HTTPRouteMatch:   testHTTPRouteMatch,
				WeightedClusters: mapset.NewSet(testWeightedCluster2),
			}},
			expectedRoutes: []*RouteWeightedClusters{{
				HTTPRouteMatch:   testHTTPRouteMatch,
				WeightedClusters: mapset.NewSet(testWeightedCluster, testWeightedCluster2),
			}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := mergeRoutesWeightedClusters(tc.originalRoutes, tc.latestRoutes)
			assert.Equal(tc.expectedRoutes, actual)
		})
	}
}
func TestNewInboundTrafficPolicy(t *testing.T) {
	assert := tassert.New(t)

	name := "name"
	hostnames := []string{"hostname1", "hostname2"}
	expected := &InboundTrafficPolicy{Name: name, Hostnames: hostnames}

	actual := NewInboundTrafficPolicy(name, hostnames)
	assert.Equal(expected, actual)
}

func TestNewRouteWeightedCluster(t *testing.T) {
	testCases := []struct {
		name             string
		route            HTTPRouteMatch
		weightedClusters []service.WeightedCluster
		expected         *RouteWeightedClusters
	}{
		{
			name:             "single weighted cluster in set",
			route:            testHTTPRouteMatch,
			weightedClusters: []service.WeightedCluster{testWeightedCluster},
			expected:         &RouteWeightedClusters{HTTPRouteMatch: testHTTPRouteMatch, WeightedClusters: mapset.NewSet(testWeightedCluster)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := NewRouteWeightedCluster(tc.route, tc.weightedClusters)
			assert.Equal(tc.expected, actual)
		})
	}
}

func TestNewOutboundPolicy(t *testing.T) {
	assert := tassert.New(t)

	name := "name"
	hostnames := []string{"hostname1", "hostname2"}
	expected := &OutboundTrafficPolicy{Name: name, Hostnames: hostnames}

	actual := NewOutboundTrafficPolicy(name, hostnames)
	assert.Equal(expected, actual)
}

func TestTotalClustersWeight(t *testing.T) {
	testCases := []struct {
		name           string
		route          RouteWeightedClusters
		expectedWeight int
	}{
		{
			name:           "route with single cluster",
			route:          testRoute2,
			expectedWeight: 100,
		},
		{
			name: "route with multiple clusters",
			route: RouteWeightedClusters{
				HTTPRouteMatch:   testHTTPRouteMatch2,
				WeightedClusters: mapset.NewSetFromSlice([]interface{}{testWeightedCluster, testWeightedCluster2}),
			},
			expectedWeight: 200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := tc.route.TotalClustersWeight()
			assert.Equal(tc.expectedWeight, actual)
		})
	}
}

func newTestInboundPolicy(name string, rules []*Rule) *InboundTrafficPolicy {
	return &InboundTrafficPolicy{
		Name:      name,
		Hostnames: testHostnames,
		Rules:     rules,
	}
}

func newTestOutboundPolicy(name string, routes []*RouteWeightedClusters) *OutboundTrafficPolicy {
	return &OutboundTrafficPolicy{
		Name:      name,
		Hostnames: testHostnames,
		Routes:    routes,
	}
}

func TestSlicesUnionIfSubset(t *testing.T) {
	first := []string{"bookstore.bookstore",
		"bookstore.bookstore.svc.cluster.local",
		"bookstore:80",
		"bookstore.bookstore.svc:80",
		"bookstore.bookstore.svc.cluster.local:80",
		"bookstore",
		"bookstore.bookstore.svc",
		"bookstore.bookstore.svc.cluster",
		"bookstore.bookstore:80",
		"bookstore.bookstore.svc.cluster:80",
	}

	second := []string{"bookstore.bookstore.svc.cluster.local"}
	assert := tassert.New(t)
	hostsUnion := slicesUnionIfSubset(first, second)
	assert.NotEqual(len(hostsUnion), 0)
	assert.ElementsMatch(first, hostsUnion)

	hostsUnion = slicesUnionIfSubset(second, first)
	assert.NotEqual(len(hostsUnion), 0)
	assert.ElementsMatch(first, hostsUnion)

	third := []string{"bookstore.bookstore.svc.cluster.local", "foo.com"}
	hostsUnion = slicesUnionIfSubset(first, third)
	assert.Equal(len(hostsUnion), 0)

	hostsUnion = slicesUnionIfSubset(third, first)
	assert.Equal(len(hostsUnion), 0)
}

func TestDeduplicateTrafficMatches(t *testing.T) {
	testCases := []struct {
		name     string
		input    []*TrafficMatch
		expected []*TrafficMatch
	}{
		{
			name: "Duplicate HTTP port based traffic match",
			input: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "http",
				},
				{
					DestinationPort:     80,
					DestinationProtocol: "http",
				},
			},
			expected: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "http",
				},
			},
		},
		{
			name: "Unique HTTP port based traffic match",
			input: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "http",
				},
				{
					DestinationPort:     90,
					DestinationProtocol: "http",
				},
			},
			expected: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "http",
				},
				{
					DestinationPort:     90,
					DestinationProtocol: "http",
				},
			},
		},
		{
			name: "HTTP and TCP traffic match",
			input: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "http",
				},
				{
					DestinationPort:     90,
					DestinationProtocol: "tcp",
				},
			},
			expected: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "http",
				},
				{
					DestinationPort:     90,
					DestinationProtocol: "tcp",
				},
			},
		},
		{
			name: "Order of IP ranges for the same port should be ignored during deduplication",
			input: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "tcp",
					DestinationIPRanges: []string{"1.1.1.1/1", "2.2.2.2/2"},
				},
				{
					DestinationPort:     80,
					DestinationProtocol: "tcp",
					DestinationIPRanges: []string{"2.2.2.2/2", "1.1.1.1/1"},
				},
			},
			expected: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "tcp",
					DestinationIPRanges: []string{"1.1.1.1/1", "2.2.2.2/2"},
				},
			},
		},
		{
			name: "HTTPS and TCP traffic matches on the same port should not collide",
			input: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "tcp",
					Cluster:             "80",
				},
				{
					DestinationPort:     80,
					DestinationProtocol: "https",
					ServerNames:         []string{"foo.com"},
					Cluster:             "80",
				},
			},
			expected: []*TrafficMatch{
				{
					DestinationPort:     80,
					DestinationProtocol: "tcp",
				},
				{
					DestinationPort:     80,
					DestinationProtocol: "https",
					ServerNames:         []string{"foo.com"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual, err := DeduplicateTrafficMatches(tc.input)
			assert.Nil(err)
			assert.Len(actual, len(tc.expected))
		})
	}
}

func TestDeduplicateClusterConfigs(t *testing.T) {
	testCases := []struct {
		name     string
		input    []*EgressClusterConfig
		expected []*EgressClusterConfig
	}{
		{
			name: "Duplicate TCP clusters",
			input: []*EgressClusterConfig{
				{
					Name: "80",
					Port: 80,
				},
				{
					Name: "80",
					Port: 80,
				},
			},
			expected: []*EgressClusterConfig{
				{
					Name: "80",
					Port: 80,
				},
			},
		},
		{
			name: "Duplicate HTTP clusters",
			input: []*EgressClusterConfig{
				{
					Name: "foo.com:80",
					Port: 80,
					Host: "foo.com",
				},
				{
					Name: "foo.com:80",
					Port: 80,
					Host: "foo.com",
				},
			},
			expected: []*EgressClusterConfig{
				{
					Name: "foo.com:80",
					Port: 80,
					Host: "foo.com",
				},
			},
		},
		{
			name: "HTTP clusters with same port different Host are not duplicates",
			input: []*EgressClusterConfig{
				{
					Name: "foo.com:80",
					Port: 80,
					Host: "foo.com",
				},
				{
					Name: "bar.com:80",
					Port: 80,
					Host: "bar.com",
				},
			},
			expected: []*EgressClusterConfig{
				{
					Name: "foo.com:80",
					Port: 80,
					Host: "foo.com",
				},
				{
					Name: "bar.com:80",
					Port: 80,
					Host: "bar.com",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual, err := DeduplicateClusterConfigs(tc.input)
			assert.Nil(err)
			assert.Len(actual, len(tc.expected))
		})
	}
}
