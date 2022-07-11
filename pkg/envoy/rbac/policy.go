package rbac

import (
	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"

	"github.com/openservicemesh/osm/pkg/identity"
)

// PolicyBuilder is a utility for constructing *xds_rbac.Policy's
type PolicyBuilder struct {
	allowedPorts       []uint32
	allowedIdentities  []identity.ServiceIdentity
	allowAllPrincipals bool

	// All permissions are applied using OR semantics by default. If applyPermissionsAsAnd is set to true, then
	// permissions are applied using AND semantics.
	applyPermissionsAsAnd bool
	trustDomain           string
}

// Build constructs an RBAC policy for the policy object on which this method is called
func (p *PolicyBuilder) Build() *xds_rbac.Policy {
	policy := &xds_rbac.Policy{}

	// Each RuleList follows OR semantics with other RuleList in the list of RuleList
	prinicipals := make([]*xds_rbac.Principal, 0, len(p.allowedIdentities))
	for _, svcIdentity := range p.allowedIdentities {
		prinicipals = append(prinicipals, GetAuthenticatedPrincipal(svcIdentity.AsPrincipal(p.trustDomain)))
	}
	if len(prinicipals) == 0 {
		// No principals specified for this policy, allow ANY
		prinicipals = []*xds_rbac.Principal{getAnyPrincipal()}
	}

	// Policies are applied with OR semantics.
	// See comments on the xds_rbac.Policy.Permissions field for more details.
	policy.Principals = prinicipals

	// Construct the Permissions ---------------------------
	// By default, permissions are applied with OR semantics.
	permissions := make([]*xds_rbac.Permission, 0, len(p.allowedPorts))
	for _, port := range p.allowedPorts {
		perm := GetDestinationPortPermission(port)
		permissions = append(permissions, perm)
	}
	if len(permissions) == 0 {
		// No principals specified for this policy, allow ANY
		permissions = []*xds_rbac.Permission{getAnyPermission()}
	}

	if p.applyPermissionsAsAnd {
		// Permissions are applied with OR semantics by default
		// See comments on the xds_rbac.Policy.Permissions field for more details.
		policy.Permissions = []*xds_rbac.Permission{andPermission(permissions)}
	} else {
		policy.Permissions = permissions
	}

	return policy
}

// UseANDForPermissions will apply all permissions with AND semantics.
func (p *PolicyBuilder) UseANDForPermissions(val bool) {
	p.applyPermissionsAsAnd = val
}

// AddIdentity adds an identity, later to be converted to a principal, to the list of allowed identities.
func (p *PolicyBuilder) AddIdentity(svcIdentity identity.ServiceIdentity) {
	// We need this extra defense in depth because it is currently possible to configure a wildcard principal
	// in addition to specific principals. Future changes may look to avoid this.
	if svcIdentity.IsWildcard() {
		p.AllowAnyIdentity()
	}
	if !p.allowAllPrincipals {
		p.allowedIdentities = append(p.allowedIdentities, svcIdentity)
	}
}

// AllowAnyIdentity allows any principal to access the permissions.
func (p *PolicyBuilder) AllowAnyIdentity() {
	p.allowedIdentities = nil
	p.allowAllPrincipals = true
}

// AddAllowedDestinationPort adds the allowed destination port to the list of allowed ports.
func (p *PolicyBuilder) AddAllowedDestinationPort(port uint16) {
	// envoy uses uint32 for ports.
	p.allowedPorts = append(p.allowedPorts, uint32(port))
}

// SetTrustDomain sets the trust domain for the policy, which is used when converting a ServiceIdentity to a Principal.
func (p *PolicyBuilder) SetTrustDomain(td string) {
	p.trustDomain = td
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

func andPermission(permissions []*xds_rbac.Permission) *xds_rbac.Permission {
	return &xds_rbac.Permission{
		Rule: &xds_rbac.Permission_AndRules{
			AndRules: &xds_rbac.Permission_Set{
				Rules: permissions,
			},
		},
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
