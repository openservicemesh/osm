package rds

import (
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// NewResponse creates a new Route Discovery Response.
func NewResponse(cataloger catalog.MeshCataloger, proxy *envoy.Proxy, cm *certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	proxyServices, err := cataloger.ListServicesForProxy(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Msgf("Error looking up services for Envoy with name=%s", proxy)
		return nil, err
	}

	trustDomain := cm.GetTrustDomain()

	statsHeaders := map[string]string{}
	if cataloger.GetMeshConfig().Spec.FeatureFlags.EnableWASMStats {
		statsHeaders, err = cataloger.GetProxyStatsHeaders(proxy)
		if err != nil {
			log.Err(err).Msgf("Error getting proxy stats headers for proxy %s", proxy)
		}
	}

	routesBuilder := RoutesBuilder().
		Proxy(proxy).
		StatsHeaders(statsHeaders).
		TrustDomain(trustDomain)

	// Get inbound mesh traffic policy and pass to builder
	inboundMeshTrafficPolicy := cataloger.GetInboundMeshTrafficPolicy(proxy.Identity, proxyServices)
	if inboundMeshTrafficPolicy != nil {
		routesBuilder.InboundPortSpecificRouteConfigs(inboundMeshTrafficPolicy.HTTPRouteConfigsPerPort)
	}

	// Get outbound mesh traffic policy and pass to builder
	outboundMeshTrafficPolicy := cataloger.GetOutboundMeshTrafficPolicy(proxy.Identity)
	if outboundMeshTrafficPolicy != nil {
		routesBuilder.OutboundPortSpecificRouteConfigs(outboundMeshTrafficPolicy.HTTPRouteConfigsPerPort)
	}

	// Get ingress traffic policies and pass to builder
	var ingressTrafficPolicies []*trafficpolicy.InboundTrafficPolicy
	for _, svc := range proxyServices {
		ingressPolicy, err := cataloger.GetIngressTrafficPolicy(svc)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting ingress traffic policy for service %s, skipping", svc)
			continue
		}
		if ingressPolicy == nil {
			log.Trace().Msgf("No ingress policy configured for service %s", svc)
			continue
		}
		ingressTrafficPolicies = trafficpolicy.MergeInboundPolicies(ingressTrafficPolicies, ingressPolicy.HTTPRoutePolicies...)
	}
	routesBuilder.IngressTrafficPolicies(ingressTrafficPolicies)

	// Get egress traffic policy and pass to builder
	egressTrafficPolicy, err := cataloger.GetEgressTrafficPolicy(proxy.Identity)

	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress traffic policies for proxy with identity %s, skipping egress route configuration", proxy.Identity)
	}
	if egressTrafficPolicy != nil {
		routesBuilder.EgressPortSpecificRouteConfigs(egressTrafficPolicy.HTTPRouteConfigsPerPort)
	}

	rdsResources, err := routesBuilder.Build()

	if err != nil {
		log.Error().Err(err).Msgf("Error building route config.")
		return nil, err
	}

	return rdsResources, nil
}
