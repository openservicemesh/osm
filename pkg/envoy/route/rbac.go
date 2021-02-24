package route

import (
	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_http_rbac "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/envoy/rbac"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	rbacPerRoutePolicyName = "rbac-for-route"
)

// buildInboundRBACFilterForRule builds an HTTP RBAC per route filter based on the given traffic policy rule.
// The principals in the RBAC policy are derived from the allowed service accounts specified in the given rule.
// The permissions in the RBAC policy are implicitly set to ANY (all permissions).
func buildInboundRBACFilterForRule(rule *trafficpolicy.Rule) (map[string]*any.Any, error) {
	if rule.AllowedServiceAccounts == nil {
		return nil, errors.Errorf("traffipolicy.Rule.AllowedServiceAccounts not set")
	}

	policy := &rbac.Policy{}

	// Create the list of principals for this policy
	var principalRuleList []rbac.RulesList
	for downstream := range rule.AllowedServiceAccounts.Iter() {
		var principalRule rbac.RulesList
		downstreamIdentity := downstream.(service.K8sServiceAccount)

		if downstreamIdentity.IsEmpty() {
			// When the downstream identity in a traffic policy rule is set to be empty, it implies
			// we must allow all downstream principals. This can be accomplished by setting an empty
			// principal rules list to generate an RBAC policy with principals set to ANY (all downstreams).
			principalRule = rbac.RulesList{}
		} else {
			// The downstream principal in an RBAC policy is an authenticated principal type, which
			// means the principal must correspond to the fully qualified SAN in the certificate presented
			// by the downstream.
			downstreamPrincipal := identity.GetKubernetesServiceIdentity(downstreamIdentity, identity.ClusterLocalTrustDomain)
			principalRule = rbac.RulesList{
				OrRules: []rbac.Rule{
					{Attribute: rbac.DownstreamAuthPrincipal, Value: downstreamPrincipal.String()},
				},
			}
		}

		principalRuleList = append(principalRuleList, principalRule)
	}

	policy.Principals = principalRuleList

	rbacPolicy, err := policy.Generate()
	if err != nil {
		return nil, err
	}

	// A single RBAC policy per route
	rbacPolicyMap := map[string]*xds_rbac.Policy{rbacPerRoutePolicyName: rbacPolicy}

	// Map generic RBAC policy to HTTP RBAC policy
	httpRBAC := &xds_http_rbac.RBAC{
		Rules: &xds_rbac.RBAC{
			Action:   xds_rbac.RBAC_ALLOW, // Allows the request if and only if there is a policy that matches the request
			Policies: rbacPolicyMap,
		},
	}
	httpRBACPerRoute := &xds_http_rbac.RBACPerRoute{
		Rbac: httpRBAC,
	}

	marshalled, err := ptypes.MarshalAny(httpRBACPerRoute)
	if err != nil {
		return nil, err
	}

	rbacFilter := map[string]*any.Any{wellknown.HTTPRoleBasedAccessControl: marshalled}
	return rbacFilter, nil
}
