package generator

import (
	"context"
	"fmt"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/envoy/generator/rds"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// generateRDS creates a new Route Discovery Response.
func (g *EnvoyConfigGenerator) generateRDS(ctx context.Context, proxy *models.Proxy) ([]types.Resource, error) {
	proxyServices, err := g.catalog.ListServicesForProxy(proxy)
	fmt.Println("\nGenerateRDS called, proxyservices: ", proxyServices)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Msgf("Error looking up services for Envoy with name=%s", proxy)
		return nil, err
	}

	trustDomain := g.certManager.GetTrustDomain()

	statsHeaders := map[string]string{}
	if g.catalog.GetMeshConfig().Spec.FeatureFlags.EnableWASMStats {
		statsHeaders, err = g.catalog.GetProxyStatsHeaders(proxy)
		if err != nil {
			log.Err(err).Msgf("Error getting proxy stats headers for proxy %s", proxy)
		}
	}

	routesBuilder := rds.RoutesBuilder().
		Proxy(proxy).
		StatsHeaders(statsHeaders).
		TrustDomain(trustDomain)

	// Get HTTP route configs per port from inbound mesh traffic policy and pass to builder
	routesBuilder.InboundPortSpecificRouteConfigs(g.catalog.GetInboundMeshHTTPRouteConfigsPerPort(proxy.Identity, proxyServices))

	// Get HTTP route configs per port from outbound mesh traffic policy and pass to builder
	routesBuilder.OutboundPortSpecificRouteConfigs(g.catalog.GetOutboundMeshHTTPRouteConfigsPerPort(proxy.Identity))

	// Get ingress http route policies and pass to builder
	var ingressTrafficPolicies []*trafficpolicy.InboundTrafficPolicy
	for _, svc := range proxyServices {
		ingressHTTPRoutePolicies := g.catalog.GetIngressHTTPRoutePoliciesForSvc(svc)
		if ingressHTTPRoutePolicies == nil {
			log.Trace().Msgf("No ingress policy configured for service %s", svc)
			continue
		}
		ingressTrafficPolicies = trafficpolicy.MergeInboundPolicies(ingressTrafficPolicies, ingressHTTPRoutePolicies...)
	}
	routesBuilder.IngressTrafficPolicies(ingressTrafficPolicies)

	// Get HTTP route configs per port from egress traffic policy and pass to builder
	routesBuilder.EgressPortSpecificRouteConfigs(g.catalog.GetEgressHTTPRouteConfigsPerPort(proxy.Identity))

	rdsResources, err := routesBuilder.Build()

	if err != nil {
		log.Error().Err(err).Msgf("Error building route config.")
		return nil, err
	}

	return rdsResources, nil
}
