package rbac

import (
	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

// PolicyBuilder is a utility for constructing *xds_rbac.Policy's
type PolicyBuilder struct {
	allowedPorts       []uint32
	allowedPrincipals  []string
	allowAllPrincipals bool
}

// Build constructs an RBAC policy for the policy object on which this method is called
func (p *PolicyBuilder) Build() *xds_rbac.Policy {
	policy := &xds_rbac.Policy{}

	// Each RuleList follows OR semantics with other RuleList in the list of RuleList
	prinicipals := make([]*xds_rbac.Principal, 0, len(p.allowedPrincipals))
	for _, principal := range p.allowedPrincipals {
		prinicipals = append(prinicipals, GetAuthenticatedPrincipal(principal))
	}
	if len(prinicipals) == 0 {
		// No principals specified for this policy, allow ANY
		prinicipals = []*xds_rbac.Principal{getAnyPrincipal()}
	}

	policy.Principals = prinicipals

	// Construct the Permissions ---------------------------
	permissions := make([]*xds_rbac.Permission, 0, len(p.allowedPorts))
	for _, port := range p.allowedPorts {
		permissions = append(permissions, GetDestinationPortPermission(port))
	}
	if len(permissions) == 0 {
		// No principals specified for this policy, allow ANY
		permissions = []*xds_rbac.Permission{getAnyPermission()}
	}

	policy.Permissions = permissions

	return policy
}

// AddPrincipal adds a principal to the list of allowed principals
func (p *PolicyBuilder) AddPrincipal(principal string) {
	// We need this extra defense in depth because it is currently possible to configure a wildcard principal
	// in addition to specific principals. We should instead not allow this, preventing this misconfiguration from
	// happening through validating webhooks.
	if principal == "*" {
		p.AllowAnyPrincipal()
	}
	if !p.allowAllPrincipals {
		// TODO(4754) return the trust domain of the first MRC, ***while grabbing the lock***
		p.allowedPrincipals = append(p.allowedPrincipals, principal)
	}
}

// AllowAnyPrincipal allows any principal to access the permissions.
func (p *PolicyBuilder) AllowAnyPrincipal() {
	p.allowedPrincipals = nil
	p.allowAllPrincipals = true
}

// AddAllowedDestinationPort adds the allowed destination port to the list of allowed ports.
func (p *PolicyBuilder) AddAllowedDestinationPort(port uint32) {
	p.allowedPorts = append(p.allowedPorts, port)
}

// GetAuthenticatedPrincipal returns an authenticated RBAC principal object for the given principal
func GetAuthenticatedPrincipal(principalName string) *xds_rbac.Principal {
	return &xds_rbac.Principal{
		Identifier: &xds_rbac.Principal_Authenticated_{
			Authenticated: &xds_rbac.Principal_Authenticated{
				PrincipalName: &xds_matcher.StringMatcher{
					MatchPattern: &xds_matcher.StringMatcher_Exact{
						Exact: principalName,
					},
				},
			},
		},
	}
}

func getAnyPrincipal() *xds_rbac.Principal {
	return &xds_rbac.Principal{
		Identifier: &xds_rbac.Principal_Any{Any: true},
	}
}

func getAnyPermission() *xds_rbac.Permission {
	return &xds_rbac.Permission{
		Rule: &xds_rbac.Permission_Any{Any: true},
	}
}

// GetDestinationPortPermission returns an RBAC permission for the given destination port
func GetDestinationPortPermission(port uint32) *xds_rbac.Permission {
	return &xds_rbac.Permission{
		Rule: &xds_rbac.Permission_DestinationPort{
			DestinationPort: port,
		},
	}
}
