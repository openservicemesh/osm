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
func NewResponse(cataloger catalog.MeshCataloger, proxy *envoy.Proxy, discoveryReq *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, _ certificate.Manager, proxyRegistry *registry.ProxyRegistry) ([]types.Resource, error) {
	var inboundTrafficPolicies []*trafficpolicy.InboundTrafficPolicy
	var outboundTrafficPolicies []*trafficpolicy.OutboundTrafficPolicy
	var ingressTrafficPolicies []*trafficpolicy.InboundTrafficPolicy

	proxyIdentity, err := envoy.GetServiceIdentityFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingServiceIdentity)).
			Msgf("Error looking up Service Account for Envoy with serial number=%q", proxy.GetCertificateSerialNumber())
		return nil, err
	}

	services, err := proxyRegistry.ListProxyServices(proxy)

	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Msgf("Error looking up services for Envoy with serial number=%q", proxy.GetCertificateSerialNumber())
		return nil, err
	}

	// Build traffic policies from  either SMI Traffic Target and Traffic Split or service discovery
	// depending on whether permissive mode is enabled or not
	inboundTrafficPolicies = cataloger.ListInboundTrafficPolicies(proxyIdentity, services)
	outboundTrafficPolicies = cataloger.ListOutboundTrafficPolicies(proxyIdentity)

	routeConfiguration := route.BuildRouteConfiguration(inboundTrafficPolicies, outboundTrafficPolicies, proxy, cfg)
	var rdsResources []types.Resource

	for _, config := range routeConfiguration {
		rdsResources = append(rdsResources, config)
	}

	// Build Ingress inbound policies for the services associated with this proxy
	for _, svc := range services {
		ingressPolicy, err := cataloger.GetIngressTrafficPolicy(svc)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting ingress traffic policy for service %s, skipping", svc)
			continue
		}
		if ingressPolicy == nil {
			log.Trace().Msgf("No ingress policy confiugred for service %s", svc)
			continue
		}
		ingressTrafficPolicies = trafficpolicy.MergeInboundPolicies(catalog.AllowPartialHostnamesMatch, ingressTrafficPolicies, ingressPolicy.HTTPRoutePolicies...)
	}
	if len(ingressTrafficPolicies) > 0 {
		ingressRouteConfig := route.BuildIngressConfiguration(ingressTrafficPolicies)
		rdsResources = append(rdsResources, ingressRouteConfig)
	}

	// Build Egress route configurations based on Egress HTTP routing rules associated with this proxy
	egressTrafficPolicy, err := cataloger.GetEgressTrafficPolicy(proxyIdentity)
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress traffic policies for proxy with identity %s, skipping egress route configuration", proxyIdentity)
	}
	if egressTrafficPolicy != nil {
		egressRouteConfigs := route.BuildEgressRouteConfiguration(egressTrafficPolicy.HTTPRouteConfigsPerPort)
		for _, egressConfig := range egressRouteConfigs {
			rdsResources = append(rdsResources, egressConfig)
		}
	}

	if discoveryReq != nil {
		// Ensure all RDS resources are responded to a given non-nil and non-empty request
		// Empty RDS RouteConfig will be provided for resources requested that our logic did not fulfill
		// due to policy logic
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
