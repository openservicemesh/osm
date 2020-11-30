package rbac

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
)

func TestGenerate(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name                string
		p                   *Policy
		expectedPrincipals  []*xds_rbac.Principal
		expectedPermissions []*xds_rbac.Permission
		expectError         bool
	}{
		{
			name: "testing AND rules for single principal",
			p: &Policy{
				Principals: []RulesList{
					{
						AndRules: []Rule{
							{Attribute: DownstreamAuthPrincipal, Value: "foo.domain"},
							{Attribute: DownstreamAuthPrincipal, Value: "bar.domain"},
						},
					},
				},
			},
			expectedPrincipals: []*xds_rbac.Principal{
				{
					Identifier: &xds_rbac.Principal_AndIds{
						AndIds: &xds_rbac.Principal_Set{
							Ids: []*xds_rbac.Principal{
								getPrincipalAuthenticated("foo.domain"),
								getPrincipalAuthenticated("bar.domain"),
							},
						},
					},
				},
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_Any{Any: true},
				},
			},
			expectError: false,
		},

		{
			name: "testing OR rules for single principal",
			p: &Policy{
				Principals: []RulesList{
					{
						OrRules: []Rule{
							{Attribute: DownstreamAuthPrincipal, Value: "foo.domain"},
							{Attribute: DownstreamAuthPrincipal, Value: "bar.domain"},
						},
					},
				},
			},
			expectedPrincipals: []*xds_rbac.Principal{
				{
					Identifier: &xds_rbac.Principal_OrIds{
						OrIds: &xds_rbac.Principal_Set{
							Ids: []*xds_rbac.Principal{
								getPrincipalAuthenticated("foo.domain"),
								getPrincipalAuthenticated("bar.domain"),
							},
						},
					},
				},
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_Any{Any: true},
				},
			},
			expectError: false,
		},

		{
			name: "testing rule for ANY principal when no AND/OR rules specified",
			p: &Policy{
				Principals: []RulesList{
					{}, // No AND/OR rules
				},
			},
			expectedPrincipals: []*xds_rbac.Principal{
				getPrincipalAny(),
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_Any{Any: true},
				},
			},
			expectError: false,
		},

		{
			name: "testing rule for no principal specified",
			p:    &Policy{},
			expectedPrincipals: []*xds_rbac.Principal{
				getPrincipalAny(),
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_Any{Any: true},
				},
			},
			expectError: false,
		},

		{
			name: "testing AND/OR rules for multiple principals",
			p: &Policy{
				Principals: []RulesList{
					{
						AndRules: []Rule{
							{Attribute: DownstreamAuthPrincipal, Value: "foo.domain"},
							{Attribute: DownstreamAuthPrincipal, Value: "bar.domain"},
						},
					},
					{
						OrRules: []Rule{
							{Attribute: DownstreamAuthPrincipal, Value: "foo.domain"},
							{Attribute: DownstreamAuthPrincipal, Value: "bar.domain"},
						},
					},
				},
			},
			expectedPrincipals: []*xds_rbac.Principal{
				{
					Identifier: &xds_rbac.Principal_AndIds{
						AndIds: &xds_rbac.Principal_Set{
							Ids: []*xds_rbac.Principal{
								getPrincipalAuthenticated("foo.domain"),
								getPrincipalAuthenticated("bar.domain"),
							},
						},
					},
				},
				{
					Identifier: &xds_rbac.Principal_OrIds{
						OrIds: &xds_rbac.Principal_Set{
							Ids: []*xds_rbac.Principal{
								getPrincipalAuthenticated("foo.domain"),
								getPrincipalAuthenticated("bar.domain"),
							},
						},
					},
				},
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_Any{Any: true},
				},
			},
			expectError: false,
		},

		{
			name: "testing error when both AND and OR rules are specified for a single principal",
			p: &Policy{
				Principals: []RulesList{
					{
						AndRules: []Rule{
							{Attribute: DownstreamAuthPrincipal, Value: "foo.domain"},
						},
						OrRules: []Rule{
							{Attribute: DownstreamAuthPrincipal, Value: "bar.domain"},
						},
					},
				},
			},
			expectedPrincipals:  nil,
			expectedPermissions: nil,
			expectError:         true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			policy, err := tc.p.Generate()

			assert.Equal(err != nil, tc.expectError)
			if err != nil {
				assert.Nil(policy)
			} else {
				assert.NotNil(policy)
				assert.Equal(policy.Principals, tc.expectedPrincipals)
				assert.Equal(policy.Permissions, tc.expectedPermissions)
			}
		})
	}
}
