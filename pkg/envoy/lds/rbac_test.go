package lds

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/envoy/rbac"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestBuildRBACPolicyFromTrafficTarget(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name          string
		trafficTarget trafficpolicy.TrafficTargetWithRoutes

		expectedPolicy *xds_rbac.Policy
		expectErr      bool
	}{
		{
			// Test 1
			name: "traffic target without TCP routes",
			trafficTarget: trafficpolicy.TrafficTargetWithRoutes{
				Name:        "ns-1/test-1",
				Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
				Sources: []identity.ServiceIdentity{
					identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
					identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
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
					{
						Identifier: &xds_rbac.Principal_OrIds{
							OrIds: &xds_rbac.Principal_Set{
								Ids: []*xds_rbac.Principal{
									rbac.GetAuthenticatedPrincipal("sa-2.ns-2.cluster.local"),
								},
							},
						},
					},
					{
						Identifier: &xds_rbac.Principal_OrIds{
							OrIds: &xds_rbac.Principal_Set{
								Ids: []*xds_rbac.Principal{
									rbac.GetAuthenticatedPrincipal("sa-3.ns-3.cluster.local"),
								},
							},
						},
					},
				},
			},
			expectErr: false, // no error
		},

		{
			// Test 2
			name: "traffic target with TCP routes",
			trafficTarget: trafficpolicy.TrafficTargetWithRoutes{
				Name:        "ns-1/test-1",
				Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
				Sources: []identity.ServiceIdentity{
					identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
					identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
				},
				TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
					{
						Ports: []int{1000, 2000},
					},
					{
						Ports: []int{3000},
					},
				},
			},

			expectedPolicy: &xds_rbac.Policy{
				Permissions: []*xds_rbac.Permission{
					{
						Rule: &xds_rbac.Permission_OrRules{
							OrRules: &xds_rbac.Permission_Set{
								Rules: []*xds_rbac.Permission{
									rbac.GetDestinationPortPermission(1000),
									rbac.GetDestinationPortPermission(2000),
								},
							},
						},
					},
					{
						Rule: &xds_rbac.Permission_OrRules{
							OrRules: &xds_rbac.Permission_Set{
								Rules: []*xds_rbac.Permission{
									rbac.GetDestinationPortPermission(3000),
								},
							},
						},
					},
				},
				Principals: []*xds_rbac.Principal{
					{
						Identifier: &xds_rbac.Principal_OrIds{
							OrIds: &xds_rbac.Principal_Set{
								Ids: []*xds_rbac.Principal{
									rbac.GetAuthenticatedPrincipal("sa-2.ns-2.cluster.local"),
								},
							},
						},
					},
					{
						Identifier: &xds_rbac.Principal_OrIds{
							OrIds: &xds_rbac.Principal_Set{
								Ids: []*xds_rbac.Principal{
									rbac.GetAuthenticatedPrincipal("sa-3.ns-3.cluster.local"),
								},
							},
						},
					},
				},
			},
			expectErr: false, // no error
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			// Test the RBAC policies
			policy, err := buildRBACPolicyFromTrafficTarget(tc.trafficTarget)

			assert.Equal(tc.expectErr, err != nil)
			assert.Equal(tc.expectedPolicy, policy)
		})
	}
}

func TestBuildInboundRBACPolicies(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	proxySvcAccount := service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}

	lb := &listenerBuilder{
		meshCatalog: mockCatalog,
		svcAccount:  proxySvcAccount,
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
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
						identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
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
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
						identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
					},
				},
				{
					Name:        "ns-1/test-2",
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-4.ns-2.cluster.local"),
					},
				},
			},

			expectedPolicyKeys: []string{"ns-1/test-1", "ns-1/test-2"},
			expectErr:          false, // no error
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			// Mock catalog calls
			mockCatalog.EXPECT().ListInboundTrafficTargetsWithRoutes(proxySvcAccount).Return(tc.trafficTargets, nil).Times(1)

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
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	proxySvcAccount := service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}

	lb := &listenerBuilder{
		meshCatalog: mockCatalog,
		svcAccount:  proxySvcAccount,
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
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
						identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
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
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
						identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
					},
				},
				{
					Name:        "ns-1/test-2",
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-4.ns-2.cluster.local"),
					},
				},
			},

			expectErr: false, // no error
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			// Mock catalog calls
			mockCatalog.EXPECT().ListInboundTrafficTargetsWithRoutes(proxySvcAccount).Return(tc.trafficTargets, nil).Times(1)

			rbacFilter, err := lb.buildRBACFilter()
			assert.Equal(err != nil, tc.expectErr)

			assert.Equal(rbacFilter.Name, wellknown.RoleBasedAccessControl)
		})
	}
}
