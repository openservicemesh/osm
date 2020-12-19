package rbac

import (
	"strconv"

	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/pkg/errors"
)

// Generate constructs an RBAC policy for the policy object on which this method is called
func (p *Policy) Generate() (*xds_rbac.Policy, error) {
	policy := &xds_rbac.Policy{}

	// Construct the Principals ------------------------
	var finalPrincipals []*xds_rbac.Principal

	// Each RuleList follows OR semantics with other RuleList in the list of RuleList
	for _, principalRuleList := range p.Principals {
		// 'principalRuleList' corresponds to a single Principal in an RBAC policy.
		// This Principal can be defined in terms of one of AND or OR rules.
		// When AND/OR semantics are not required to define multiple rules corresponding
		// to this principal, a single Rule in either the AndRules or OrRules will suffice.

		if len(principalRuleList.AndRules) != 0 && len(principalRuleList.OrRules) != 0 {
			return nil, errors.New("Principal rule cannot have both AND & OR rules at the same time")
		}

		var currentPrincipal *xds_rbac.Principal

		switch {
		case len(principalRuleList.AndRules) != 0:
			// Combine all the AND rules for this Principal rule with AND semantics
			var andPrincipalRules []*xds_rbac.Principal
			for _, andPrincipalRule := range principalRuleList.AndRules {
				// Fill in the authenticated principal types
				if andPrincipalRule.Attribute == DownstreamAuthPrincipal {
					authPrincipal := GetAuthenticatedPrincipal(andPrincipalRule.Value)
					andPrincipalRules = append(andPrincipalRules, authPrincipal)
				}
			}
			currentPrincipal = andPrincipals(andPrincipalRules)

		case len(principalRuleList.OrRules) != 0:
			// Combine all the OR rules for this Principal rule with OR semantics
			var orPrincipalRules []*xds_rbac.Principal
			for _, orPrincipalRule := range principalRuleList.OrRules {
				// Fill in the authenticated principal types
				if orPrincipalRule.Attribute == DownstreamAuthPrincipal {
					authPrincipal := GetAuthenticatedPrincipal(orPrincipalRule.Value)
					orPrincipalRules = append(orPrincipalRules, authPrincipal)
				}
			}
			currentPrincipal = orPrincipals(orPrincipalRules)

		default:
			// Neither AND/OR rules set, set principal to Any
			currentPrincipal = getAnyPrincipal()
		}

		finalPrincipals = append(finalPrincipals, currentPrincipal)
	}
	if len(p.Principals) == 0 {
		// No principals specified for this policy, allow ANY
		finalPrincipals = append(finalPrincipals, getAnyPrincipal())
	}

	policy.Principals = finalPrincipals

	// Construct the Permissions ---------------------------
	var finalPermissions []*xds_rbac.Permission

	// Each RuleList follows OR semantics with other RuleList in the list of RuleList
	for _, permissionRuleList := range p.Permissions {
		// 'principalRuleList' corresponds to a single Principal in an RBAC policy.
		// This Principal can be defined in terms of one of AND or OR rules.
		// When AND/OR semantics are not required to define multiple rules corresponding
		// to this principal, a single Rule in either the AndRules or OrRules will suffice.

		if len(permissionRuleList.AndRules) != 0 && len(permissionRuleList.OrRules) != 0 {
			return nil, errors.New("Permission rule cannot have both AND & OR rules at the same time")
		}

		var currentPermission *xds_rbac.Permission

		switch {
		case len(permissionRuleList.AndRules) != 0:
			// Combine all the AND rules for this Permission rule with AND semantics
			var andPermissionRules []*xds_rbac.Permission
			for _, andPermissionRule := range permissionRuleList.AndRules {
				// Fill in the destination port permission
				if andPermissionRule.Attribute == DestinationPort {
					port, err := strconv.ParseUint(andPermissionRule.Value, 10, 32)
					if err != nil {
						return nil, errors.Errorf("Error parsing destination port value %s", andPermissionRule.Value)
					}
					portPermission := GetDestinationPortPermission(uint32(port))
					andPermissionRules = append(andPermissionRules, portPermission)
				}
			}
			currentPermission = andPermissions(andPermissionRules)

		case len(permissionRuleList.OrRules) != 0:
			// Combine all the OR rules for this Permission rule with OR semantics
			var orPermissionRules []*xds_rbac.Permission
			for _, orPermissionRule := range permissionRuleList.OrRules {
				// Fill in the destination port permission
				if orPermissionRule.Attribute == DestinationPort {
					port, err := strconv.ParseUint(orPermissionRule.Value, 10, 32)
					if err != nil {
						return nil, errors.Errorf("Error parsing destination port value %s", orPermissionRule.Value)
					}
					portPermission := GetDestinationPortPermission(uint32(port))
					orPermissionRules = append(orPermissionRules, portPermission)
				}
			}
			currentPermission = orPermissions(orPermissionRules)

		default:
			// Neither AND/OR rules set, set permission to Any
			currentPermission = getAnyPermission()
		}

		finalPermissions = append(finalPermissions, currentPermission)
	}
	if len(p.Permissions) == 0 {
		// No permissions specified for this policy, allow ANY
		finalPermissions = append(finalPermissions, getAnyPermission())
	}

	policy.Permissions = finalPermissions

	return policy, nil
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

func orPrincipals(principals []*xds_rbac.Principal) *xds_rbac.Principal {
	return &xds_rbac.Principal{
		Identifier: &xds_rbac.Principal_OrIds{
			OrIds: &xds_rbac.Principal_Set{
				Ids: principals,
			},
		},
	}
}

func andPrincipals(principals []*xds_rbac.Principal) *xds_rbac.Principal {
	return &xds_rbac.Principal{
		Identifier: &xds_rbac.Principal_AndIds{
			AndIds: &xds_rbac.Principal_Set{
				Ids: principals,
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

func orPermissions(permissions []*xds_rbac.Permission) *xds_rbac.Permission {
	return &xds_rbac.Permission{
		Rule: &xds_rbac.Permission_OrRules{
			OrRules: &xds_rbac.Permission_Set{
				Rules: permissions,
			},
		},
	}
}

func andPermissions(permissions []*xds_rbac.Permission) *xds_rbac.Permission {
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
