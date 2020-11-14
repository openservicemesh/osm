package lds

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestBuildInboundRBACPolicies(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	proxySvcAccount := service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}

	lb := &listenerBuilder{
		meshCatalog: mockCatalog,
		svcAccount:  proxySvcAccount,
	}

	testCases := []struct {
		name                      string
		allowedInboundSvcAccounts []service.K8sServiceAccount
		expectedPrincipals        []string
		expectErr                 bool
	}{
		{
			name: "multiple client allowed",
			allowedInboundSvcAccounts: []service.K8sServiceAccount{
				{Name: "sa-2", Namespace: "ns-2"},
				{Name: "sa-3", Namespace: "ns-3"},
			},
			expectedPrincipals: []string{
				"sa-2.ns-2.cluster.local",
				"sa-3.ns-3.cluster.local",
			},
			expectErr: false, // no error
		},
		{
			name:                      "no clients allowed",
			allowedInboundSvcAccounts: []service.K8sServiceAccount{},
			expectedPrincipals:        []string{},
			expectErr:                 false, // no error
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			// Mock the calls to catalog
			mockCatalog.EXPECT().ListAllowedInboundServiceAccounts(lb.svcAccount).Return(tc.allowedInboundSvcAccounts, nil).Times(1)

			// Test the RBAC policies
			networkRBAC, err := lb.buildInboundRBACPolicies()
			assert.Equal(err != nil, tc.expectErr)

			assert.Equal(networkRBAC.Rules.GetAction(), xds_rbac.RBAC_ALLOW)

			rbacPolicies := networkRBAC.Rules.Policies

			// Expect 1 policy per client principal
			assert.Len(rbacPolicies, len(tc.expectedPrincipals))

			// Loop through the policies and ensure there is a policy corresponding to each principal
			var actualPrincipals []string
			for _, policy := range rbacPolicies {
				principalName := policy.Principals[0].GetOrIds().Ids[0].GetAuthenticated().PrincipalName.GetExact()
				actualPrincipals = append(actualPrincipals, principalName)

				assert.Len(policy.Permissions, 1) // Any permission
				assert.True(policy.Permissions[0].GetAny())
			}
			assert.ElementsMatch(tc.expectedPrincipals, actualPrincipals)
		})
	}
}

func TestBuildRBACFilter(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	proxySvcAccount := service.K8sServiceAccount{Name: "sa-1", Namespace: "ns-1"}

	lb := &listenerBuilder{
		meshCatalog: mockCatalog,
		svcAccount:  proxySvcAccount,
	}

	testCases := []struct {
		name                      string
		allowedInboundSvcAccounts []service.K8sServiceAccount
		expectErr                 bool
	}{
		{
			name: "multiple clients allowed",
			allowedInboundSvcAccounts: []service.K8sServiceAccount{
				{Name: "sa-2", Namespace: "ns-2"},
				{Name: "sa-3", Namespace: "ns-3"},
			},
			expectErr: false, // no error
		},
		{
			name:                      "no clients allowed",
			allowedInboundSvcAccounts: []service.K8sServiceAccount{},
			expectErr:                 false, // no error
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			// Mock the calls to catalog
			mockCatalog.EXPECT().ListAllowedInboundServiceAccounts(lb.svcAccount).Return(tc.allowedInboundSvcAccounts, nil).Times(1)

			// Test the RBAC filter
			rbacFilter, err := lb.buildRBACFilter()
			assert.Equal(err != nil, tc.expectErr)

			assert.Equal(rbacFilter.Name, wellknown.RoleBasedAccessControl)
		})
	}
}

func TestGetPolicyName(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		downstream   service.K8sServiceAccount
		upstream     service.K8sServiceAccount
		expectedName string
	}{
		{
			downstream:   service.K8sServiceAccount{Name: "foo", Namespace: "ns-1"},
			upstream:     service.K8sServiceAccount{Name: "bar", Namespace: "ns-2"},
			expectedName: "ns-1/foo to ns-2/bar",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			actual := getPolicyName(tc.downstream, tc.upstream)
			assert.Equal(actual, tc.expectedName)
		})
	}
}
