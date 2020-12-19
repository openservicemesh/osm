package lds

import (
	"strconv"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_network_rbac "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/envoy/rbac"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// buildRBACFilter builds an RBAC filter based on SMI TrafficTarget policies.
// The returned RBAC filter has policies that gives downstream principals full access to the local service.
func (lb *listenerBuilder) buildRBACFilter() (*xds_listener.Filter, error) {
	networkRBACPolicy, err := lb.buildInboundRBACPolicies()
	if err != nil {
		log.Error().Err(err).Msgf("Error building inbound RBAC policies for principal %q", lb.svcAccount)
		return nil, err
	}

	marshalledNetworkRBACPolicy, err := ptypes.MarshalAny(networkRBACPolicy)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling RBAC policy: %v", networkRBACPolicy)
		return nil, err
	}

	rbacFilter := &xds_listener.Filter{
		Name:       wellknown.RoleBasedAccessControl,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledNetworkRBACPolicy},
	}

	return rbacFilter, nil
}

// buildInboundRBACPolicies builds the RBAC policies based on allowed principals
func (lb *listenerBuilder) buildInboundRBACPolicies() (*xds_network_rbac.RBAC, error) {
	proxyIdentity := identity.ServiceIdentity(lb.svcAccount.String())
	trafficTargets, err := lb.meshCatalog.ListInboundTrafficTargetsWithRoutes(lb.svcAccount)
	if err != nil {
		log.Error().Err(err).Msgf("Error listing allowed inbound traffic targets for proxy identity %s", proxyIdentity)
		return nil, err
	}

	rbacPolicies := make(map[string]*xds_rbac.Policy)
	// Build an RBAC policies based on SMI TrafficTarget policies
	for _, targetPolicy := range trafficTargets {
		if policy, err := buildRBACPolicyFromTrafficTarget(targetPolicy); err != nil {
			log.Error().Err(err).Msgf("Error building RBAC policy for proxy identity %s from TrafficTarget %s", proxyIdentity, targetPolicy.Name)
		} else {
			rbacPolicies[targetPolicy.Name] = policy
		}
	}

	log.Debug().Msgf("RBAC policy for proxy with identity %s: %+v", proxyIdentity, rbacPolicies)

	// Create an inbound RBAC policy that denies a request by default, unless a policy explicitly allows it
	networkRBACPolicy := &xds_network_rbac.RBAC{
		StatPrefix: "RBAC",
		Rules: &xds_rbac.RBAC{
			Action:   xds_rbac.RBAC_ALLOW, // Allows the request if and only if there is a policy that matches the request
			Policies: rbacPolicies,
		},
	}

	return networkRBACPolicy, nil
}

// buildRBACPolicyFromTrafficTarget creates an XDS RBAC policy from the given traffic target policy
func buildRBACPolicyFromTrafficTarget(trafficTarget trafficpolicy.TrafficTargetWithRoutes) (*xds_rbac.Policy, error) {
	policy := &rbac.Policy{}

	// Create the list of principals for this policy
	var principalRuleList []rbac.RulesList
	for _, downstreamPrincipal := range trafficTarget.Sources {
		principalRule := rbac.RulesList{
			OrRules: []rbac.Rule{
				{Attribute: rbac.DownstreamAuthPrincipal, Value: downstreamPrincipal.String()},
			},
		}
		principalRuleList = append(principalRuleList, principalRule)
	}
	policy.Principals = principalRuleList

	// Create the list of permissions for this policy
	var permissionRuleList []rbac.RulesList
	for _, tcpRouteMatch := range trafficTarget.TCPRouteMatches {
		// Matching ports have an OR relationship
		var orPortRules []rbac.Rule
		for _, port := range tcpRouteMatch.Ports {
			portRule := rbac.Rule{
				Attribute: rbac.DestinationPort, Value: strconv.Itoa(port),
			}
			orPortRules = append(orPortRules, portRule)
		}

		// Each TCP route match is its own permission in an RBAC policy
		permissionRule := rbac.RulesList{
			OrRules: orPortRules,
		}

		permissionRuleList = append(permissionRuleList, permissionRule)
	}
	policy.Permissions = permissionRuleList

	return policy.Generate()
}
