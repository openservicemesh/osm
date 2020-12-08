package trafficpolicy

import (
	"testing"

	set "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/service"
)

var (
	testHTTPRouteMatch = HTTPRouteMatch{
		PathRegex: "/hello",
		Methods:   []string{"GET"},
		Headers:   map[string]string{"hello": "world"},
	}

	testHTTPRouteMatch2 = HTTPRouteMatch{
		PathRegex: "/goodbye",
		Methods:   []string{"GET"},
		Headers:   map[string]string{"later": "alligator"},
	}

	testHostnames = []string{"testHostname1", "testHostname2", "testHostname3"}

	testWeightedCluster = service.WeightedCluster{
		ClusterName: "testCluster",
		Weight:      100,
	}
	testWeightedCluster2 = service.WeightedCluster{
		ClusterName: "testCluster2",
		Weight:      100,
	}

	testServiceAccount1 = service.K8sServiceAccount{
		Name:      "testServiceAccount1",
		Namespace: "testNamespace1",
	}

	testServiceAccount2 = service.K8sServiceAccount{
		Name:      "testServiceAccount2",
		Namespace: "testNamespace2",
	}

	testRoute = RouteWeightedClusters{
		HTTPRouteMatch:   testHTTPRouteMatch,
		WeightedClusters: set.NewSet(testWeightedCluster),
	}
)

func TestAddRule(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name                  string
		existingRules         []*Rule
		allowedServiceAccount service.K8sServiceAccount
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
					Route:                  testRoute,
					AllowedServiceAccounts: set.NewSet(testServiceAccount1),
				},
			},
		},
		{
			name: "rule exists for route but not for given service account",
			existingRules: []*Rule{
				{
					Route:                  testRoute,
					AllowedServiceAccounts: set.NewSet(testServiceAccount1),
				},
			},
			allowedServiceAccount: testServiceAccount2,
			route:                 testRoute,
			expectedRules: []*Rule{
				{
					Route:                  testRoute,
					AllowedServiceAccounts: set.NewSet(testServiceAccount1, testServiceAccount2),
				},
			},
		},
		{
			name: "rule exists for route and for given service account",
			existingRules: []*Rule{
				{
					Route:                  testRoute,
					AllowedServiceAccounts: set.NewSet(testServiceAccount1),
				},
			},
			allowedServiceAccount: testServiceAccount1,
			route:                 testRoute,
			expectedRules: []*Rule{
				{
					Route:                  testRoute,
					AllowedServiceAccounts: set.NewSet(testServiceAccount1),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inboundPolicy := newTestInboundPolicy(tc.name, tc.existingRules)
			inboundPolicy.AddRule(tc.route, tc.allowedServiceAccount)
			assert.Equal(tc.expectedRules, inboundPolicy.Rules)
		})
	}
}

func TestAddRoute(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name                  string
		existingRoutes        []*RouteWeightedClusters
		expectedRoutes        []*RouteWeightedClusters
		givenRouteMatch       HTTPRouteMatch
		givenWeightedClusters []service.WeightedCluster
		expectedErr           bool
	}{
		{
			name:                  "no routes exist",
			existingRoutes:        []*RouteWeightedClusters{},
			givenRouteMatch:       testHTTPRouteMatch,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: set.NewSet(testWeightedCluster),
				},
			},
			expectedErr: false,
		},
		{
			name: "add route to existing routes",
			existingRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: set.NewSet(testWeightedCluster),
				},
			},
			givenRouteMatch:       testHTTPRouteMatch2,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster2},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: set.NewSet(testWeightedCluster),
				},
				{
					HTTPRouteMatch:   testHTTPRouteMatch2,
					WeightedClusters: set.NewSet(testWeightedCluster2),
				},
			},
			expectedErr: false,
		},
		{
			name: "add route with multiple weighted clusters to existing routes",
			existingRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: set.NewSet(testWeightedCluster),
				},
			},
			givenRouteMatch:       testHTTPRouteMatch2,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster, testWeightedCluster2},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: set.NewSet(testWeightedCluster),
				},
				{
					HTTPRouteMatch:   testHTTPRouteMatch2,
					WeightedClusters: set.NewSet(testWeightedCluster, testWeightedCluster2),
				},
			},
			expectedErr: false,
		},
		{
			name: "route already exists, same weighted cluster",
			existingRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: set.NewSet(testWeightedCluster),
				},
			},
			givenRouteMatch:       testHTTPRouteMatch,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: set.NewSet(testWeightedCluster),
				},
			},
			expectedErr: false,
		},
		{
			name: "route already exists, different weighted cluster",
			existingRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: set.NewSet(testWeightedCluster),
				},
			},
			givenRouteMatch:       testHTTPRouteMatch,
			givenWeightedClusters: []service.WeightedCluster{testWeightedCluster2},
			expectedRoutes: []*RouteWeightedClusters{
				{
					HTTPRouteMatch:   testHTTPRouteMatch,
					WeightedClusters: set.NewSet(testWeightedCluster),
				},
			},
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outboundPolicy := newTestOutboundPolicy(tc.name, tc.existingRoutes)
			err := outboundPolicy.AddRoute(tc.givenRouteMatch, tc.givenWeightedClusters...)
			if tc.expectedErr {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
			assert.Equal(tc.expectedRoutes, outboundPolicy.Routes)
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
