package lds

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rbac"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestBuildRBACPolicyFromTrafficTarget(t *testing.T) {
	testCases := []struct {
		name          string
		trafficTarget trafficpolicy.TrafficTargetWithRoutes

		expectedPolicy *xds_rbac.Policy
	}{
		{
			// Test 1
			name: "traffic target without TCP routes",
			trafficTarget: trafficpolicy.TrafficTargetWithRoutes{
				Name:        "ns-1/test-1",
				Destination: identity.ServiceIdentity("sa-1.ns-1"),
				Sources: []identity.ServiceIdentity{
					identity.ServiceIdentity("sa-2.ns-2"),
					identity.ServiceIdentity("sa-3.ns-3"),
				},
				TCPRouteMatches: nil,
			},

			expectedPolicy: &xds_rbac.Policy{
				Permissions: []*xds_rbac.Permission{
					{
						Rule: &xds_rbac.Permission_Any{Any: true},
					},
				},
				Principals: []*xds_rbac.Principal{
					rbac.GetAuthenticatedPrincipal("sa-2.ns-2.cluster.local"),
					rbac.GetAuthenticatedPrincipal("sa-3.ns-3.cluster.local"),
				},
			},
		},

		{
			// Test 2
			name: "traffic target with TCP routes",
			trafficTarget: trafficpolicy.TrafficTargetWithRoutes{
				Name:        "ns-1/test-1",
				Destination: identity.ServiceIdentity("sa-1.ns-1"),
				Sources: []identity.ServiceIdentity{
					identity.ServiceIdentity("sa-2.ns-2"),
					identity.ServiceIdentity("sa-3.ns-3"),
				},
				TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
					{
						Ports: []uint16{1000, 2000},
					},
					{
						Ports: []uint16{3000},
					},
				},
			},

			expectedPolicy: &xds_rbac.Policy{
				Permissions: []*xds_rbac.Permission{
					rbac.GetDestinationPortPermission(1000),
					rbac.GetDestinationPortPermission(2000),
					rbac.GetDestinationPortPermission(3000),
				},
				Principals: []*xds_rbac.Principal{
					rbac.GetAuthenticatedPrincipal("sa-2.ns-2.cluster.local"),
					rbac.GetAuthenticatedPrincipal("sa-3.ns-3.cluster.local"),
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			// Test the RBAC policies
			policy := buildRBACPolicyFromTrafficTarget(tc.trafficTarget, "cluster.local")

			assert.Equal(tc.expectedPolicy, policy)
		})
	}
}

func TestBuildInboundRBACPolicies(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	proxySvcAccount := identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}

	lb := &listenerBuilder{
		meshCatalog:     mockCatalog,
		serviceIdentity: proxySvcAccount.ToServiceIdentity(),
	}

	testCases := []struct {
		name           string
		trafficTargets []trafficpolicy.TrafficTargetWithRoutes

		expectedPolicyKeys []string
		expectErr          bool
	}{
		{
			// Test 1
			name: "traffic target without TCP routes",
			trafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2"),
						identity.ServiceIdentity("sa-3.ns-3"),
					},
					TCPRouteMatches: nil,
				},
			},

			expectedPolicyKeys: []string{"ns-1/test-1"},

			expectErr: false, // no error
		},

		{
			// Test 2
			name: "traffic target with TCP routes",
			trafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2"),
						identity.ServiceIdentity("sa-3.ns-3"),
					},
				},
				{
					Name:        "ns-1/test-2",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-4.ns-2"),
					},
				},
			},

			expectedPolicyKeys: []string{"ns-1/test-1", "ns-1/test-2"},
			expectErr:          false, // no error
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			// Mock catalog calls
			mockCatalog.EXPECT().ListInboundTrafficTargetsWithRoutes(proxySvcAccount.ToServiceIdentity()).Return(tc.trafficTargets, nil).Times(1)

			// Test the RBAC policies
			policy, err := lb.buildInboundRBACPolicies()

			assert.Equal(tc.expectErr, err != nil)
			assert.Equal(xds_rbac.RBAC_ALLOW, policy.Rules.Action)
			assert.Len(policy.Rules.Policies, len(tc.expectedPolicyKeys))

			var actualPolicyKeys []string
			for key := range policy.Rules.Policies {
				actualPolicyKeys = append(actualPolicyKeys, key)
			}
			assert.ElementsMatch(tc.expectedPolicyKeys, actualPolicyKeys)
		})
	}
}

func TestBuildRBACFilter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	proxySvcAccount := identity.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}.ToServiceIdentity()

	lb := &listenerBuilder{
		meshCatalog:     mockCatalog,
		serviceIdentity: proxySvcAccount,
	}

	testCases := []struct {
		name           string
		trafficTargets []trafficpolicy.TrafficTargetWithRoutes

		expectErr bool
	}{
		{
			// Test 1
			name: "traffic target without TCP routes",
			trafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2"),
						identity.ServiceIdentity("sa-3.ns-3"),
					},
					TCPRouteMatches: nil,
				},
			},

			expectErr: false, // no error
		},

		{
			// Test 2
			name: "traffic target with TCP routes",
			trafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2"),
						identity.ServiceIdentity("sa-3.ns-3"),
					},
				},
				{
					Name:        "ns-1/test-2",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-4.ns-2"),
					},
				},
			},

			expectErr: false, // no error
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			// Mock catalog calls
			mockCatalog.EXPECT().ListInboundTrafficTargetsWithRoutes(proxySvcAccount).Return(tc.trafficTargets, nil).Times(1)

			rbacFilter, err := lb.buildRBACFilter()
			assert.Equal(err != nil, tc.expectErr)

			assert.Equal(envoy.L4RBACFilterName, rbacFilter.Name)
		})
	}
}
