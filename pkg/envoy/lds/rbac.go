package lds

import (
	"fmt"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_network_rbac "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/openservicemesh/osm/pkg/envoy"
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

	marshalledNetworkRBACPolicy, err := envoy.MessageToAny(networkRBACPolicy)
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
		rbacPolicies[policyName] = buildAllowAllPermissionsPolicy(principal)
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
func buildAllowAllPermissionsPolicy(clientPrincipal identity.ServiceIdentity) *xds_rbac.Policy {
	return &xds_rbac.Policy{
		Permissions: []*xds_rbac.Permission{
			{
				// Grant the given principal all access
				Rule: &xds_rbac.Permission_Any{Any: true},
			},
		},
		Principals: []*xds_rbac.Principal{
			{
				Identifier: &xds_rbac.Principal_OrIds{
					OrIds: &xds_rbac.Principal_Set{
						Ids: []*xds_rbac.Principal{
							getPrincipalAuthenticated(clientPrincipal.String()),
						},
					},
				},
			},
		},
	}
}

// getPolicyName returns a policy name for the policy used to authorize a downstream service account by the upstream
func getPolicyName(downstream, upstream service.K8sServiceAccount) string {
	return fmt.Sprintf("%s to %s", downstream, upstream)
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
