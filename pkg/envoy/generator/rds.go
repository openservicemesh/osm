package generator

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/envoy/generator/rds"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// generateRDS creates a new Route Discovery Response.
func (g *EnvoyConfigGenerator) generateRDS(ctx context.Context, proxy *models.Proxy) ([]types.Resource, error) {
	proxyServices, err := g.catalog.ListServicesForProxy(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Msgf("Error looking up services for Envoy with name=%s", proxy)
		return nil, err
	}

	statsHeaders := map[string]string{}
	if g.catalog.GetMeshConfig().Spec.FeatureFlags.EnableWASMStats {
		statsHeaders, err = g.catalog.GetProxyStatsHeaders(proxy)
		if err != nil {
			log.Err(err).Msgf("Error getting proxy stats headers for proxy %s", proxy)
		}
	}

	routesBuilder := rds.RoutesBuilder().
		Proxy(proxy).
		StatsHeaders(statsHeaders)

	inboundTPBuilder := trafficpolicy.InboundTrafficPolicyBuilder()
	inboundTPBuilder.UpstreamServices(proxyServices)
	inboundTPBuilder.UpstreamIdentity(proxy.Identity)
	inboundTPBuilder.EnablePermissiveTrafficPolicyMode(g.catalog.GetMeshConfig().Spec.Traffic.EnablePermissiveTrafficPolicyMode)
	inboundTPBuilder.TrustDomain(g.certManager.GetTrustDomains())
	inboundTPBuilder.HttpTrafficSpecsList(g.catalog.ListHTTPTrafficSpecs())

	destinationFilter := smi.WithTrafficTargetDestination(proxy.Identity.ToK8sServiceAccount())
	inboundTPBuilder.TrafficTargetsByOptions(g.catalog.ListTrafficTargetsByOptions(destinationFilter))

	allUpstreamSvcIncludeApex := g.catalog.GetUpstreamServicesIncludeApex(proxyServices)
	inboundTPBuilder.UpstreamServicesIncludeApex(allUpstreamSvcIncludeApex)

	var upstreamTrafficSettingsPerService map[*service.MeshService]*policyv1alpha1.UpstreamTrafficSetting
	var hostnamesPerService map[*service.MeshService][]string

	for _, upstreamSvc := range allUpstreamSvcIncludeApex {
		upstreamSvc := upstreamSvc // To prevent loop variable memory aliasing in for loop
		upstreamTrafficSettingsPerService[&upstreamSvc] = g.catalog.GetUpstreamTrafficSettingByService(&upstreamSvc)
		hostnamesPerService[&upstreamSvc] = g.catalog.GetHostnamesForService(upstreamSvc, true /* local namespace FQDN should always be allowed for inbound routes*/)
	}

	inboundTPBuilder.UpstreamTrafficSettingsPerService(upstreamTrafficSettingsPerService)
	inboundTPBuilder.HostnamesPerService(hostnamesPerService)

	// Get HTTP route configs per port from inbound mesh traffic policy and pass to builder
	routesBuilder.InboundPortSpecificRouteConfigs(inboundTPBuilder.GetInboundMeshHTTPRouteConfigsPerPort())

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
