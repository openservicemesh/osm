package lds

import (
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_network_rbac "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rbac"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// buildRBACFilter builds an RBAC filter based on SMI TrafficTarget policies.
// The returned RBAC filter has policies that gives downstream principals full access to the local service.
func buildRBACFilter(trafficTargets []trafficpolicy.TrafficTargetWithRoutes, trustDomain certificate.TrustDomain) (*xds_listener.Filter, error) {
	networkRBACPolicy, err := buildInboundRBACPolicies(trafficTargets, trustDomain)
	if err != nil {
		return nil, err
	}

	marshalledNetworkRBACPolicy, err := anypb.New(networkRBACPolicy)
	if err != nil {
		return nil, err
	}

	rbacFilter := &xds_listener.Filter{
		Name:       envoy.L4RBACFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledNetworkRBACPolicy},
	}

	return rbacFilter, nil
}

// buildInboundRBACPolicies builds the RBAC policies based on allowed principals
func buildInboundRBACPolicies(trafficTargets []trafficpolicy.TrafficTargetWithRoutes, trustDomain certificate.TrustDomain) (*xds_network_rbac.RBAC, error) {
	rbacPolicies := make(map[string]*xds_rbac.Policy)
	// Build an RBAC policies based on SMI TrafficTarget policies
	for _, targetPolicy := range trafficTargets {
		rbacPolicies[targetPolicy.Name] = buildRBACPolicyFromTrafficTarget(targetPolicy, trustDomain)
	}

	// Create an inbound RBAC policy that denies a request by default, unless a policy explicitly allows it
	networkRBACPolicy := &xds_network_rbac.RBAC{
		StatPrefix: "network-", // will be displayed as network-rbac.<path>
		Rules: &xds_rbac.RBAC{
			Action:   xds_rbac.RBAC_ALLOW, // Allows the request if and only if there is a policy that matches the request
			Policies: rbacPolicies,
		},
	}

	return networkRBACPolicy, nil
}

// buildRBACPolicyFromTrafficTarget creates an XDS RBAC policy from the given traffic target policy
func buildRBACPolicyFromTrafficTarget(trafficTarget trafficpolicy.TrafficTargetWithRoutes, trustDomains certificate.TrustDomain) *xds_rbac.Policy {
	pb := &rbac.PolicyBuilder{}

	// Create the list of identities for this policy
	for _, downstreamIdentity := range trafficTarget.Sources {
		pb.AddPrincipal(downstreamIdentity.AsPrincipal(trustDomains.Signing))
		if trustDomains.AreDifferent() {
			pb.AddPrincipal(downstreamIdentity.AsPrincipal(trustDomains.Validating))
		}
	}
	// Create the list of permissions for this policy
	for _, tcpRouteMatch := range trafficTarget.TCPRouteMatches {
		// Matching ports have an OR relationship
		for _, port := range tcpRouteMatch.Ports {
			pb.AddAllowedDestinationPort(port)
		}
	}

	return pb.Build()
}
