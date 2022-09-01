package route

import (
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_http_rbac "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy/rbac"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestBuildInboundRBACFilterForRule(t *testing.T) {
	testCases := []struct {
		name               string
		rule               *trafficpolicy.Rule
		expectedRBACPolicy *xds_rbac.Policy
		expectError        bool
	}{
		{
			name: "valid trafficpolicy rule with restricted downstream identities",
			rule: &trafficpolicy.Rule{
				Route: trafficpolicy.RouteWeightedClusters{
					HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
					WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
				},
				AllowedPrincipals: mapset.NewSet(
					identity.K8sServiceAccount{Name: "foo", Namespace: "ns-1"}.AsPrincipal("cluster.local"),
					identity.K8sServiceAccount{Name: "bar", Namespace: "ns-2"}.AsPrincipal("cluster.local"),
				),
			},
			expectedRBACPolicy: &xds_rbac.Policy{
				Principals: []*xds_rbac.Principal{
					rbac.GetAuthenticatedPrincipal("foo.ns-1.cluster.local"),
					rbac.GetAuthenticatedPrincipal("bar.ns-2.cluster.local"),
				},
				Permissions: []*xds_rbac.Permission{
					{
						Rule: &xds_rbac.Permission_Any{Any: true},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid trafficpolicy rule which allows all downstream identities",
			rule: &trafficpolicy.Rule{
				Route: trafficpolicy.RouteWeightedClusters{
					HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
					WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
				},
				AllowedPrincipals: mapset.NewSet(
					identity.WildcardPrincipal, // setting a wildcard will result in all downstream identities being allowed
				),
			},
			expectedRBACPolicy: &xds_rbac.Policy{
				Principals: []*xds_rbac.Principal{
					{
						Identifier: &xds_rbac.Principal_Any{Any: true},
					},
				},
				Permissions: []*xds_rbac.Permission{
					{
						Rule: &xds_rbac.Permission_Any{Any: true},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid trafficpolicy rule with Rule.AllowedPrincipals not specified",
			rule: &trafficpolicy.Rule{
				Route: trafficpolicy.RouteWeightedClusters{
					HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
					WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
				},
			},
			expectedRBACPolicy: nil,
			expectError:        true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			rbacFilter, err := buildInboundRBACFilterForRule(tc.rule, "cluster.local")

			assert.Equal(tc.expectError, err != nil)
			if err != nil {
				assert.Nil(rbacFilter)
				return
			}

			httpRBACPerRoute := &xds_http_rbac.RBACPerRoute{}
			err = rbacFilter.UnmarshalTo(httpRBACPerRoute)
			assert.Nil(err)

			rbacRules := httpRBACPerRoute.Rbac.Rules
			assert.Equal(xds_rbac.RBAC_ALLOW, rbacRules.Action)

			rbacPolicy := rbacRules.Policies[rbacPerRoutePolicyName]

			// Match principals regardless of their order in the generated RBAC policy
			assert.ElementsMatch(tc.expectedRBACPolicy.Principals, rbacPolicy.Principals)
			assert.Equal(tc.expectedRBACPolicy.Permissions, rbacPolicy.Permissions)
		})
	}
}
