package lds

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy/rbac"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestBuildRBACPolicyFromTrafficTarget(t *testing.T) {
	testCases := []struct {
		name              string
		trafficTarget     trafficpolicy.TrafficTargetWithRoutes
		configuredIssuers certificate.IssuerInfo
		expectedPolicy    *xds_rbac.Policy
	}{
		{
			// Test 1
			name: "traffic target without TCP routes",
			configuredIssuers: certificate.IssuerInfo{
				Signing: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
				Validating: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
			},
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
			configuredIssuers: certificate.IssuerInfo{
				Signing: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
				Validating: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
			},
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

		{
			name: "traffic target without TCP routes and multiple trust domains",
			configuredIssuers: certificate.IssuerInfo{
				Signing: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
				Validating: certificate.PrincipalInfo{
					TrustDomain:   "cluster.new",
					SpiffeEnabled: false,
				},
			},
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
					rbac.GetAuthenticatedPrincipal("sa-2.ns-2.cluster.new"),
					rbac.GetAuthenticatedPrincipal("sa-3.ns-3.cluster.local"),
					rbac.GetAuthenticatedPrincipal("sa-3.ns-3.cluster.new"),
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i+1, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			// Test the RBAC policies
			fb := filterBuilder{
				issuers: tc.configuredIssuers,
			}

			policy := fb.buildRBACPolicyFromTrafficTarget(tc.trafficTarget)

			assert.Equal(tc.expectedPolicy, policy)
		})
	}
}

func TestBuildInboundRBACPolicies(t *testing.T) {
	testCases := []struct {
		name               string
		trafficTargets     []trafficpolicy.TrafficTargetWithRoutes
		configuredIssuers  certificate.IssuerInfo
		expectedPolicyKeys map[string][]string
		expectErr          bool
	}{
		{
			// Test 1
			name: "traffic target without TCP routes",
			configuredIssuers: certificate.IssuerInfo{
				Signing: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
				Validating: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
			},
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

			expectedPolicyKeys: map[string][]string{
				"ns-1/test-1": {"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local"},
			},

			expectErr: false, // no error
		},

		{
			// Test 2
			name: "traffic target with TCP routes",
			configuredIssuers: certificate.IssuerInfo{
				Signing: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
				Validating: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
			},
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

			expectedPolicyKeys: map[string][]string{
				"ns-1/test-1": {
					"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local",
				},
				"ns-1/test-2": {
					"sa-4.ns-2.cluster.local",
				},
			},
			expectErr: false, // no error
		},
		{
			name: "traffic target without TCP routes and different trust domains",
			configuredIssuers: certificate.IssuerInfo{
				Signing: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
				Validating: certificate.PrincipalInfo{
					TrustDomain:   "cluster.new",
					SpiffeEnabled: false,
				},
			},
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

			expectedPolicyKeys: map[string][]string{
				"ns-1/test-1": {"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local", "sa-2.ns-2.cluster.new", "sa-3.ns-3.cluster.new"},
			},

			expectErr: false, // no error
		},
		{
			name: "traffic target without TCP routes and spiffe is enabled for one",
			configuredIssuers: certificate.IssuerInfo{
				Signing: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: false,
				},
				Validating: certificate.PrincipalInfo{
					TrustDomain:   "cluster.local",
					SpiffeEnabled: true,
				},
			},
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

			expectedPolicyKeys: map[string][]string{
				"ns-1/test-1": {"sa-2.ns-2.cluster.local", "sa-3.ns-3.cluster.local", "spiffe://cluster.local/sa-2/ns-2", "spiffe://cluster.local/sa-3/ns-3"},
			},

			expectErr: false, // no error
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			// Test the RBAC policies
			fb := filterBuilder{
				trafficTargets: tc.trafficTargets,
				issuers:        tc.configuredIssuers,
			}
			policy, err := fb.buildInboundRBACPolicies()

			assert.Equal(tc.expectErr, err != nil)
			assert.Equal(xds_rbac.RBAC_ALLOW, policy.Rules.Action)
			assert.Len(policy.Rules.Policies, len(tc.expectedPolicyKeys))

			var actualPolicyKeys []string
			for key, v := range policy.Rules.Policies {
				actualPolicyKeys = append(actualPolicyKeys, key)

				expectedPrincipals := tc.expectedPolicyKeys[key]
				var actualPrincipals []string
				for _, v := range v.GetPrincipals() {
					p := v.GetAuthenticated().GetPrincipalName().GetExact()
					actualPrincipals = append(actualPrincipals, p)
				}
				assert.ElementsMatch(expectedPrincipals, actualPrincipals)
			}

			var expectedKeys []string
			for key := range tc.expectedPolicyKeys {
				expectedKeys = append(expectedKeys, key)
			}
			assert.ElementsMatch(expectedKeys, actualPolicyKeys)
		})
	}
}
