package rbac

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/identity"

	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

func TestBuild(t *testing.T) {
	testCases := []struct {
		name                  string
		identities            []identity.ServiceIdentity
		ports                 []uint16
		applyPermissionsAsAND bool
		trustDomain           string
		expectedPolicy        *xds_rbac.Policy
	}{
		{
			name:        "testing rules for single principal",
			identities:  []identity.ServiceIdentity{identity.New("foo", "domain"), identity.New("bar", "domain")},
			ports:       []uint16{80},
			trustDomain: "cluster.local",
			expectedPolicy: &xds_rbac.Policy{
				Principals: []*xds_rbac.Principal{
					{
						Identifier: &xds_rbac.Principal_Authenticated_{
							Authenticated: &xds_rbac.Principal_Authenticated{
								PrincipalName: &xds_matcher.StringMatcher{
									MatchPattern: &xds_matcher.StringMatcher_Exact{
										Exact: "foo.domain.cluster.local",
									},
								},
							},
						},
					},
					{
						Identifier: &xds_rbac.Principal_Authenticated_{
							Authenticated: &xds_rbac.Principal_Authenticated{
								PrincipalName: &xds_matcher.StringMatcher{
									MatchPattern: &xds_matcher.StringMatcher_Exact{
										Exact: "bar.domain.cluster.local",
									},
								},
							},
						},
					},
				},
				Permissions: []*xds_rbac.Permission{
					{
						Rule: &xds_rbac.Permission_DestinationPort{
							DestinationPort: 80,
						},
					},
				},
			},
		},
		{
			name:        "testing rules for single principal",
			identities:  []identity.ServiceIdentity{identity.New("foo", "domain")},
			trustDomain: "cluster.local",
			ports:       []uint16{80, 443},
			expectedPolicy: &xds_rbac.Policy{
				Principals: []*xds_rbac.Principal{
					{
						Identifier: &xds_rbac.Principal_Authenticated_{
							Authenticated: &xds_rbac.Principal_Authenticated{
								PrincipalName: &xds_matcher.StringMatcher{
									MatchPattern: &xds_matcher.StringMatcher_Exact{
										Exact: "foo.domain.cluster.local",
									},
								},
							},
						},
					},
				},
				Permissions: []*xds_rbac.Permission{
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
		},
		{
			// Note that AND ports wouldn't make sense, since you can't have 2 ports at once, but we use it to test
			// the logic.
			name:                  "testing rules for AND ports",
			identities:            []identity.ServiceIdentity{identity.New("foo", "domain")},
			ports:                 []uint16{80, 443},
			applyPermissionsAsAND: true,
			trustDomain:           "cluster.local",
			expectedPolicy: &xds_rbac.Policy{
				Principals: []*xds_rbac.Principal{
					{
						Identifier: &xds_rbac.Principal_Authenticated_{
							Authenticated: &xds_rbac.Principal_Authenticated{
								PrincipalName: &xds_matcher.StringMatcher{
									MatchPattern: &xds_matcher.StringMatcher_Exact{
										Exact: "foo.domain.cluster.local",
									},
								},
							},
						},
					},
				},
				Permissions: []*xds_rbac.Permission{
					{
						Rule: &xds_rbac.Permission_AndRules{
							AndRules: &xds_rbac.Permission_Set{
								Rules: []*xds_rbac.Permission{
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
						},
					},
				},
			},
		},
		{
			name:       "testing rule for ANY principal when no ports specified",
			identities: []identity.ServiceIdentity{identity.New("foo", "domain"), identity.WildcardServiceIdentity},
			expectedPolicy: &xds_rbac.Policy{
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
		},
		{
			name: "testing rule for no principal specified",
			expectedPolicy: &xds_rbac.Policy{
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
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			pb := &PolicyBuilder{}
			for _, svcIdentity := range tc.identities {
				pb.AddIdentity(svcIdentity)
			}
			for _, port := range tc.ports {
				pb.AddAllowedDestinationPort(port)
			}

			pb.UseANDForPermissions(tc.applyPermissionsAsAND)
			pb.SetTrustDomain(tc.trustDomain)

			policy := pb.Build()
			assert.Equal(tc.expectedPolicy, policy)
		})
	}
}
