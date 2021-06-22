package rbac

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
)

func TestGenerate(t *testing.T) {
	assert := tassert.New(t)

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
								GetAuthenticatedPrincipal("foo.domain"),
								GetAuthenticatedPrincipal("bar.domain"),
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
								GetAuthenticatedPrincipal("foo.domain"),
								GetAuthenticatedPrincipal("bar.domain"),
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
				getAnyPrincipal(),
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
				getAnyPrincipal(),
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
								GetAuthenticatedPrincipal("foo.domain"),
								GetAuthenticatedPrincipal("bar.domain"),
							},
						},
					},
				},
				{
					Identifier: &xds_rbac.Principal_OrIds{
						OrIds: &xds_rbac.Principal_Set{
							Ids: []*xds_rbac.Principal{
								GetAuthenticatedPrincipal("foo.domain"),
								GetAuthenticatedPrincipal("bar.domain"),
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

		{
			name: "testing single principal AND rules with single permission AND rules",
			p: &Policy{
				Principals: []RulesList{
					{
						AndRules: []Rule{
							{Attribute: DownstreamAuthPrincipal, Value: "foo.domain"},
							{Attribute: DownstreamAuthPrincipal, Value: "bar.domain"},
						},
					},
				},
				Permissions: []RulesList{
					{
						AndRules: []Rule{
							{Attribute: DestinationPort, Value: "80"},
							{Attribute: DestinationPort, Value: "90"},
						},
					},
				},
			},
			expectedPrincipals: []*xds_rbac.Principal{
				{
					Identifier: &xds_rbac.Principal_AndIds{
						AndIds: &xds_rbac.Principal_Set{
							Ids: []*xds_rbac.Principal{
								GetAuthenticatedPrincipal("foo.domain"),
								GetAuthenticatedPrincipal("bar.domain"),
							},
						},
					},
				},
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_AndRules{
						AndRules: &xds_rbac.Permission_Set{
							Rules: []*xds_rbac.Permission{
								GetDestinationPortPermission(80),
								GetDestinationPortPermission(90),
							},
						},
					},
				},
			},
			expectError: false,
		},

		{
			name: "testing single principal AND rules with single permission OR rule",
			p: &Policy{
				Principals: []RulesList{
					{
						AndRules: []Rule{
							{Attribute: DownstreamAuthPrincipal, Value: "foo.domain"},
							{Attribute: DownstreamAuthPrincipal, Value: "bar.domain"},
						},
					},
				},
				Permissions: []RulesList{
					{
						OrRules: []Rule{
							{Attribute: DestinationPort, Value: "80"},
						},
					},
				},
			},
			expectedPrincipals: []*xds_rbac.Principal{
				{
					Identifier: &xds_rbac.Principal_AndIds{
						AndIds: &xds_rbac.Principal_Set{
							Ids: []*xds_rbac.Principal{
								GetAuthenticatedPrincipal("foo.domain"),
								GetAuthenticatedPrincipal("bar.domain"),
							},
						},
					},
				},
			},
			expectedPermissions: []*xds_rbac.Permission{
				{
					Rule: &xds_rbac.Permission_OrRules{
						OrRules: &xds_rbac.Permission_Set{
							Rules: []*xds_rbac.Permission{
								GetDestinationPortPermission(80),
							},
						},
					},
				},
			},
			expectError: false,
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
