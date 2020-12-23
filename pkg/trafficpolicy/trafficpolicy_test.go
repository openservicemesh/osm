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

	testHostnames = []string{"testHostname1", "testHostname2", "testHostname3"}

	testWeightedCluster = service.WeightedCluster{
		ClusterName: "testCluster",
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

func newTestInboundPolicy(name string, rules []*Rule) *InboundTrafficPolicy {
	return &InboundTrafficPolicy{
		Name:      name,
		Hostnames: testHostnames,
		Rules:     rules,
	}
}
