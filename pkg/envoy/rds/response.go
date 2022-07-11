package rds

import (
	mapset "github.com/deckarep/golang-set"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// NewResponse creates a new Route Discovery Response.
func NewResponse(cataloger catalog.MeshCataloger, proxy *envoy.Proxy, discoveryReq *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, cm *certificate.Manager, proxyRegistry *registry.ProxyRegistry) ([]types.Resource, error) {
	proxyServices, err := proxyRegistry.ListProxyServices(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Msgf("Error looking up services for Envoy with name=%s", proxy.GetName())
		return nil, err
	}

	var rdsResources []types.Resource

	trustDomain := cm.GetTrustDomain()

	// ---
	// Build inbound mesh route configurations. These route configurations allow
	// the services associated with this proxy to accept traffic from downstream
	// clients on allowed routes.
	inboundMeshTrafficPolicy := cataloger.GetInboundMeshTrafficPolicy(proxy.Identity, proxyServices)
	if inboundMeshTrafficPolicy != nil {
		inboundMeshRouteConfig := route.BuildInboundMeshRouteConfiguration(inboundMeshTrafficPolicy.HTTPRouteConfigsPerPort, proxy, cfg, trustDomain)
		for _, config := range inboundMeshRouteConfig {
			rdsResources = append(rdsResources, config)
		}
	}

	// ---
	// Build outbound mesh route configurations. These route configurations allow this proxy
	// to direct traffic to upstream services that it is authorized to connect to on allowed
	// routes.
	outboundMeshTrafficPolicy := cataloger.GetOutboundMeshTrafficPolicy(proxy.Identity)

	if outboundMeshTrafficPolicy != nil {
		outboundMeshRouteConfig := route.BuildOutboundMeshRouteConfiguration(outboundMeshTrafficPolicy.HTTPRouteConfigsPerPort)
		for _, config := range outboundMeshRouteConfig {
			rdsResources = append(rdsResources, config)
		}
	}

	// ---
	// Build ingress route configurations. These route configurations allow the
	// services associated with this proxy to accept ingress traffic from downstream
	// clients on allowed routes.
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
	if len(ingressTrafficPolicies) > 0 {
		ingressRouteConfig := route.BuildIngressConfiguration(ingressTrafficPolicies, trustDomain)
		rdsResources = append(rdsResources, ingressRouteConfig)
	}

	// ---
	// Build egress route configurations. These route configurations allow this
	// proxy to direct traffic to external non-mesh destinations on allowed routes.
	egressTrafficPolicy, err := cataloger.GetEgressTrafficPolicy(proxy.Identity)
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress traffic policies for proxy with identity %s, skipping egress route configuration", proxy.Identity)
	}
	if egressTrafficPolicy != nil {
		egressRouteConfigs := route.BuildEgressRouteConfiguration(egressTrafficPolicy.HTTPRouteConfigsPerPort)
		for _, egressConfig := range egressRouteConfigs {
			rdsResources = append(rdsResources, egressConfig)
		}
	}

	// ---
	// To ensure the XDS state machine converages, it is possible that an LDS configuration
	// references in RDS configuration that does not exist. It is okay for this to happen,
	// but we need to ensure an empty RDS route configuration is returned for the requested
	// RDS resources that OSM cannot fulfill so that the XDS state machine converges. Envoy
	// will ignore any configuration resource that it doesn't require.
	if discoveryReq != nil {
		rdsResources = ensureRDSRequestCompletion(discoveryReq, rdsResources)
	}

	return rdsResources, nil
}

// ensureRDSRequestCompletion computes delta between requested resources and response resources.
// If any resources requested were not responded to, this function will fill those in with empty RouteConfig stubs
func ensureRDSRequestCompletion(discoveryReq *xds_discovery.DiscoveryRequest, rdsResources []types.Resource) []types.Resource {
	requestMapset := mapset.NewSet()
	for _, resourceName := range discoveryReq.ResourceNames {
		requestMapset.Add(resourceName)
	}

	responseMapset := mapset.NewSet()
	for _, resourceName := range rdsResources {
		responseMapset.Add(cache.GetResourceName(resourceName))
	}

	// If there were any requested elements we didn't reply to, create empty RDS resources
	// for those now
	requestDifference := requestMapset.Difference(responseMapset)
	for reqDif := range requestDifference.Iterator().C {
		unfulfilledRequestedResource := reqDif.(string)
		rdsResources = append(rdsResources, route.NewRouteConfigurationStub(unfulfilledRequestedResource))
	}

	log.Info().Msgf("RDS did not fulfill all requested resources (diff: %v). Fulfill with empty RouteConfigs.", requestDifference)

	return rdsResources
}
