package lds

import (
	"fmt"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_network_rbac "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/envoy/rbac"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
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
	allowsInboundSvcAccounts, err := lb.meshCatalog.ListAllowedInboundServiceAccounts(lb.svcAccount)
	if err != nil {
		log.Error().Err(err).Msgf("Error listing allowed inbound ServiceAccounts for ServiceAccount %q", lb.svcAccount)
		return nil, err
	}

	log.Trace().Msgf("Building RBAC policies for ServiceAccount %q with allowed inbound %v", lb.svcAccount, allowsInboundSvcAccounts)

	// Each downstream is a principal in the RBAC policy, which will have its own permissions
	// based on SMI TrafficTarget policies.
	rbacPolicies := make(map[string]*xds_rbac.Policy)
	for _, downstreamSvcAccount := range allowsInboundSvcAccounts {
		policyName := getPolicyName(downstreamSvcAccount, lb.svcAccount)
		principal := identity.GetKubernetesServiceIdentity(downstreamSvcAccount, identity.ClusterLocalTrustDomain)
		if policy, err := buildAllowAllPermissionsPolicy(principal); err != nil {
			log.Error().Err(err).Msgf("Error building RBAC policy for ServiceAccount %q and downstream %q", lb.svcAccount, downstreamSvcAccount)
		} else {
			rbacPolicies[policyName] = policy
		}
	}

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

// buildAllowAllPermissionsPolicy creates an XDS RBAC policy for the given client principal to be granted all access
func buildAllowAllPermissionsPolicy(clientPrincipal identity.ServiceIdentity) (*xds_rbac.Policy, error) {
	policy := &rbac.Policy{
		Principals: []rbac.RulesList{
			{
				OrRules: []rbac.Rule{
					{Attribute: rbac.DownstreamAuthPrincipal, Value: clientPrincipal.String()},
				},
			},
		},
		// Permissions set to ANY if not specified, which grants all access for the given Principals
	}

	return policy.Generate()
}

// getPolicyName returns a policy name for the policy used to authorize a downstream service account by the upstream
func getPolicyName(downstream, upstream service.K8sServiceAccount) string {
	return fmt.Sprintf("%s to %s", downstream, upstream)
}
