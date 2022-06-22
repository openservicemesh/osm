package lds

import (
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_rbac "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	xds_network_rbac "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rbac"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// buildRBACFilter builds an RBAC filter based on SMI TrafficTarget policies.
// The returned RBAC filter has policies that gives downstream principals full access to the local service.
func (lb *listenerBuilder) buildRBACFilter() (*xds_listener.Filter, error) {
	networkRBACPolicy, err := lb.buildInboundRBACPolicies()
	if err != nil {
		log.Error().Err(err).Msgf("Error building inbound RBAC policies for principal %q", lb.serviceIdentity)
		return nil, err
	}

	marshalledNetworkRBACPolicy, err := anypb.New(networkRBACPolicy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling RBAC policy: %v", networkRBACPolicy)
		return nil, err
	}

	rbacFilter := &xds_listener.Filter{
		Name:       envoy.L4RBACFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledNetworkRBACPolicy},
	}

	return rbacFilter, nil
}

// buildInboundRBACPolicies builds the RBAC policies based on allowed principals
func (lb *listenerBuilder) buildInboundRBACPolicies() (*xds_network_rbac.RBAC, error) {
	proxyIdentity := identity.ServiceIdentity(lb.serviceIdentity.String())
	trafficTargets, err := lb.meshCatalog.ListInboundTrafficTargetsWithRoutes(lb.serviceIdentity)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingInboundTrafficTargets)).
			Msgf("Error listing allowed inbound traffic targets for proxy identity %s", proxyIdentity)
		return nil, err
	}

	rbacPolicies := make(map[string]*xds_rbac.Policy)
	// Build an RBAC policies based on SMI TrafficTarget policies
	for _, targetPolicy := range trafficTargets {
		rbacPolicies[targetPolicy.Name] = buildRBACPolicyFromTrafficTarget(targetPolicy, lb.trustDomain)
	}

	log.Debug().Msgf("RBAC policy for proxy with identity %s: %+v", proxyIdentity, rbacPolicies)

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
func buildRBACPolicyFromTrafficTarget(trafficTarget trafficpolicy.TrafficTargetWithRoutes, trustDomain string) *xds_rbac.Policy {
	pb := &rbac.PolicyBuilder{}

	// Create the list of identities for this policy
	for _, downstreamIdentity := range trafficTarget.Sources {
		pb.AddIdentity(downstreamIdentity)
	}
	// Create the list of permissions for this policy
	for _, tcpRouteMatch := range trafficTarget.TCPRouteMatches {
		// Matching ports have an OR relationship
		for _, port := range tcpRouteMatch.Ports {
			pb.AddAllowedDestinationPort(port)
		}
	}

	pb.SetTrustDomain(trustDomain)

	return pb.Build()
}
