package rds

import (
<<<<<<< HEAD
	"fmt"

	set "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
=======
	mapset "github.com/deckarep/golang-set"
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// NewResponse creates a new Route Discovery Response.
func NewResponse(cataloger catalog.MeshCataloger, proxy *envoy.Proxy, discoveryReq *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, _ certificate.Manager, proxyRegistry *registry.ProxyRegistry) ([]types.Resource, error) {
	var inboundTrafficPolicies []*trafficpolicy.InboundTrafficPolicy
	var outboundTrafficPolicies []*trafficpolicy.OutboundTrafficPolicy
	var ingressTrafficPolicies []*trafficpolicy.InboundTrafficPolicy

	proxyIdentity, err := envoy.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service Account for Envoy with serial number=%q", proxy.GetCertificateSerialNumber())
		return nil, err
	}

<<<<<<< HEAD

	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("Failed listing routes for proxyServiceName:%+v", proxyServiceName))
		return nil, err
	}
	//log.Debug().Msgf("RDS proxy:%+v trafficPolicies:%+v", proxy, allTrafficPolicies)
=======
	services, err := proxyRegistry.ListProxyServices(proxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up services for Envoy with serial number=%q", proxy.GetCertificateSerialNumber())
		return nil, err
	}
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7

	// Build traffic policies from  either SMI Traffic Target and Traffic Split or service discovery
	// depending on whether permissive mode is enabled or not
	inboundTrafficPolicies = cataloger.ListInboundTrafficPolicies(proxyIdentity.ToServiceIdentity(), services)
	outboundTrafficPolicies = cataloger.ListOutboundTrafficPolicies(proxyIdentity.ToServiceIdentity())

	routeConfiguration := route.BuildRouteConfiguration(inboundTrafficPolicies, outboundTrafficPolicies, proxy)
	var rdsResources []types.Resource

	for _, config := range routeConfiguration {
		rdsResources = append(rdsResources, config)
	}

<<<<<<< HEAD
	allTrafficSplits, _, _, _, _ := catalog.ListSMIPolicies()
	var routeConfiguration []*xds_route.RouteConfiguration
	outboundRouteConfig := route.NewRouteConfigurationStub(route.OutboundRouteConfigName)
	inboundRouteConfig := route.NewRouteConfigurationStub(route.InboundRouteConfigName)
	outboundAggregatedRoutesByHostnames := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)
	inboundAggregatedRoutesByHostnames := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)

	for _, trafficPolicy := range allTrafficPolicies {
		isSourceService := trafficPolicy.Source.Equals(proxyServiceName)
		isDestinationService := trafficPolicy.Destination.GetMeshService().Equals(proxyServiceName)
		svc := trafficPolicy.Destination.GetMeshService()
		hostnames, err := catalog.GetResolvableHostnamesForUpstreamService(proxyServiceName, svc)
		//filter out traffic split service, reference to pkg/catalog/xds_certificates.go:74
		if isTrafficSplitService(svc, allTrafficSplits) {
			continue
		}
		if err != nil {
			log.Error().Err(err).Msg("Failed listing domains")
			return nil, err
		}
		log.Debug().Msgf("RDS hostnames: %+v", hostnames)

		// multiple targets exist per service
		var weightedCluster service.WeightedCluster
		target := trafficPolicy.Destination
		if target.Port != 0 {
			hostnames = filterOnTargetPort(hostnames, target.Port)
			log.Debug().Msgf("RDS filtered hostnames: %+v", hostnames)
			weightedCluster, err = catalog.GetWeightedClusterForServicePort(target)
			if err != nil {
				log.Error().Err(err).Msg("Failed listing clusters")
				return nil, err
			}
		} else {

			weightedCluster, err = catalog.GetWeightedClusterForService(svc)
			if err != nil {
				log.Error().Err(err).Msg("Failed listing clusters")
				return nil, err
			}
		}
		log.Debug().Msgf("RDS weightedCluster: %+v", weightedCluster)

		// All routes from a given source to destination are part of 1 traffic policy between the source and destination.
		for _, hostname := range hostnames {
			for _, httpRoute := range trafficPolicy.HTTPRouteMatches {
				if isSourceService {
					aggregateRoutesByHost(outboundAggregatedRoutesByHostnames, httpRoute, weightedCluster, hostname, target.Port)
				}

				if isDestinationService {
					aggregateRoutesByHost(inboundAggregatedRoutesByHostnames, httpRoute, weightedCluster, hostname, target.Port)
				}
			}
		}
	}

	/* do not include ingress routes for now as iptables should take care of it
	if err = updateRoutesForIngress(proxyServiceName, catalog, inboundAggregatedRoutesByHostnames); err != nil {
		return nil, err
=======
	// Build Ingress inbound policies for the services associated with this proxy
	for _, svc := range services {
		ingressInboundPolicies, err := cataloger.GetIngressPoliciesForService(svc)
		if err != nil {
			log.Error().Err(err).Msgf("Error looking up ingress policies for service=%s", svc)
			return nil, err
		}
		ingressTrafficPolicies = trafficpolicy.MergeInboundPolicies(catalog.AllowPartialHostnamesMatch, ingressTrafficPolicies, ingressInboundPolicies...)
	}
	if len(ingressTrafficPolicies) > 0 {
		ingressRouteConfig := route.BuildIngressConfiguration(ingressTrafficPolicies, proxy)
		rdsResources = append(rdsResources, ingressRouteConfig)
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
	}
	*/

<<<<<<< HEAD
	route.UpdateRouteConfiguration(catalog, outboundAggregatedRoutesByHostnames, outboundRouteConfig, route.OutboundRoute)
	route.UpdateRouteConfiguration(catalog, inboundAggregatedRoutesByHostnames, inboundRouteConfig, route.InboundRoute)
	routeConfiguration = append(routeConfiguration, inboundRouteConfig)
	routeConfiguration = append(routeConfiguration, outboundRouteConfig)

	//log.Debug().Msgf("RDS proxy: %+v routeConfiguration: %+v", proxy, routeConfiguration)

	for _, config := range routeConfiguration {
		marshalledRouteConfig, err := ptypes.MarshalAny(config)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal route config for proxy")
			return nil, err
=======
	// Build Egress route configurations based on Egress HTTP routing rules associated with this proxy
	egressTrafficPolicy, err := cataloger.GetEgressTrafficPolicy(proxyIdentity.ToServiceIdentity())
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress traffic policies for proxy with identity %s, skipping egress route configuration", proxyIdentity)
	}
	if egressTrafficPolicy != nil {
		egressRouteConfigs := route.BuildEgressRouteConfiguration(egressTrafficPolicy.HTTPRouteConfigsPerPort)
		for _, egressConfig := range egressRouteConfigs {
			rdsResources = append(rdsResources, egressConfig)
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
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

<<<<<<< HEAD
func aggregateRoutesByHost(routesPerHost map[string]map[string]trafficpolicy.RouteWeightedClusters, routePolicy trafficpolicy.HTTPRouteMatch, weightedCluster service.WeightedCluster, hostname string, targetPort int) {
	host := kubernetes.GetServiceFromHostname(hostname)
	if targetPort != 0 {
		host = fmt.Sprintf("%s:%d", host, targetPort)
	}
	_, exists := routesPerHost[host]
	if !exists {
		// no host found, create a new route map
		routesPerHost[host] = make(map[string]trafficpolicy.RouteWeightedClusters)
=======
// ensureRDSRequestCompletion computes delta between requested resources and response resources.
// If any resources requested were not responded to, this function will fill those in with empty RouteConfig stubs
func ensureRDSRequestCompletion(discoveryReq *xds_discovery.DiscoveryRequest, rdsResources []types.Resource) []types.Resource {
	requestMapset := mapset.NewSet()
	for _, resourceName := range discoveryReq.ResourceNames {
		requestMapset.Add(resourceName)
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
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
