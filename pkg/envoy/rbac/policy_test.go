package rbac

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
)

func TestBuild(t *testing.T) {
	testCases := []struct {
		name                string
		p                   *PolicyBuilder
		expectedPrincipals  []*xds_rbac.Principal
		expectedPermissions []*xds_rbac.Permission
	}{
		{
			name: "testing AND rules for single principal",
			p: &PolicyBuilder{
				allowedPrincipals: []string{"foo.domain", "bar.domain"},
			},
			expectedPrincipals: []*xds_rbac.Principal{
				GetAuthenticatedPrincipal("foo.domain"),
				GetAuthenticatedPrincipal("bar.domain"),
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_Any{Any: true},
				},
			},
		},

		{
			name: "testing OR rules for single principal",
			p: &PolicyBuilder{
				allowedPrincipals: []string{"foo.domain"},
				allowedPorts:      []uint32{80, 443},
			},
			expectedPrincipals: []*xds_rbac.Principal{
				GetAuthenticatedPrincipal("foo.domain"),
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_DestinationPort{
						DestinationPort: 80,
					},
				},
				{
					Rule: &xds_rbac.Permission_DestinationPort{
						DestinationPort: 443,
					},
				},
			},
		},

		{
			name: "testing rule for ANY principal when no AND/OR rules specified",
			p:    &PolicyBuilder{},
			expectedPrincipals: []*xds_rbac.Principal{
				getAnyPrincipal(),
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_Any{Any: true},
				},
			},
		},

		{
			name: "testing rule for no principal specified",
			p:    &PolicyBuilder{},
			expectedPrincipals: []*xds_rbac.Principal{
				getAnyPrincipal(),
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_Any{Any: true},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			policy := tc.p.Build()

			assert.NotNil(policy)
			assert.Equal(policy.Principals, tc.expectedPrincipals)
			assert.Equal(policy.Permissions, tc.expectedPermissions)
		})
	}
}
