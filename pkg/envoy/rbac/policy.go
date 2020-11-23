package rbac

import (
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

		if len(principalRuleList.AndRules) != 0 {
			// Combine all the AND rules for this Principal rule with AND semantics
			var andPrincipalRules []*xds_rbac.Principal
			for _, andPrincipalRule := range principalRuleList.AndRules {
				// Fill in the authenticated principal types
				if andPrincipalRule.Attribute == DownstreamAuthPrincipal {
					authPrincipal := getPrincipalAuthenticated(andPrincipalRule.Value)
					andPrincipalRules = append(andPrincipalRules, authPrincipal)
				}
			}
			currentPrincipal = getPrincipalAnd(andPrincipalRules)
		} else if len(principalRuleList.OrRules) != 0 {
			// Combine all the OR rules for this Principal rule with OR semantics
			var orPrincipalRules []*xds_rbac.Principal
			for _, orPrincipalRule := range principalRuleList.OrRules {
				// Fill in the authenticated principal types
				if orPrincipalRule.Attribute == DownstreamAuthPrincipal {
					authPrincipal := getPrincipalAuthenticated(orPrincipalRule.Value)
					orPrincipalRules = append(orPrincipalRules, authPrincipal)
				}
			}
			currentPrincipal = getPrincipalOr(orPrincipalRules)
		} else {
			// Neither AND/OR rules set, set principal to Any
			currentPrincipal = getPrincipalAny()
		}

		finalPrincipals = append(finalPrincipals, currentPrincipal)
	}
	if len(p.Principals) == 0 {
		// No principals specified for this policy, allow ANY
		finalPrincipals = append(finalPrincipals, getPrincipalAny())
	}

	policy.Principals = finalPrincipals

	// Construct the Permissions ---------------------------
	// Currently an RBAC policy grants all permissions.
	// This will be extended in the future as a part of TCP policy support
	// where permission will only be granted to select ports.
	policy.Permissions = []*xds_rbac.Permission{
		{
			// Grant the given principal all access
			Rule: &xds_rbac.Permission_Any{Any: true},
		},
	}

	return policy, nil
}

func getPrincipalAuthenticated(principalName string) *xds_rbac.Principal {
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

func getPrincipalOr(principals []*xds_rbac.Principal) *xds_rbac.Principal {
	return &xds_rbac.Principal{
		Identifier: &xds_rbac.Principal_OrIds{
			OrIds: &xds_rbac.Principal_Set{
				Ids: principals,
			},
		},
	}
}

func getPrincipalAnd(principals []*xds_rbac.Principal) *xds_rbac.Principal {
	return &xds_rbac.Principal{
		Identifier: &xds_rbac.Principal_AndIds{
			AndIds: &xds_rbac.Principal_Set{
				Ids: principals,
			},
		},
	}
}

func getPrincipalAny() *xds_rbac.Principal {
	return &xds_rbac.Principal{
		Identifier: &xds_rbac.Principal_Any{Any: true},
	}
}
