package rds

import (
	"context"

	set "github.com/deckarep/golang-set"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/route"
	"github.com/open-service-mesh/osm/pkg/featureflags"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
)

// NewResponse creates a new Route Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	proxyServiceName := proxy.GetService()
	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msg("Failed listing routes")
		return nil, err
	}
	log.Debug().Msgf("trafficPolicies: %+v", allTrafficPolicies)

	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	routeConfiguration := []xds.RouteConfiguration{}
	sourceRouteConfig := route.NewRouteConfigurationStub(route.OutboundRouteConfig)
	destinationRouteConfig := route.NewRouteConfigurationStub(route.InboundRouteConfig)
	sourceAggregatedRoutesByDomain := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)
	destinationAggregatedRoutesByDomain := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)

	for _, trafficPolicies := range allTrafficPolicies {
		isSourceService := trafficPolicies.Source.Services.Contains(proxyServiceName)
		isDestinationService := trafficPolicies.Destination.Services.Contains(proxyServiceName)
		for serviceInterface := range trafficPolicies.Destination.Services.Iter() {
			service := serviceInterface.(service.NamespacedService)
			domain, err := catalog.GetDomainForService(service)
			if err != nil {
				log.Error().Err(err).Msg("Failed listing domains")
				return nil, err
			}
			weightedCluster, err := catalog.GetWeightedClusterForService(service)
			if err != nil {
				log.Error().Err(err).Msg("Failed listing clusters")
				return nil, err
			}
			if isSourceService {
				aggregateRoutesByDomain(sourceAggregatedRoutesByDomain, trafficPolicies.Routes, weightedCluster, domain)
			}
			if isDestinationService {
				aggregateRoutesByDomain(destinationAggregatedRoutesByDomain, trafficPolicies.Routes, weightedCluster, domain)
			}
		}
	}

	if featureflags.IsIngressEnabled() {
		// Process ingress policy if applicable
		if err = updateRoutesForIngress(proxyServiceName, catalog, destinationAggregatedRoutesByDomain); err != nil {
			return nil, err
		}
	}

	sourceRouteConfig = route.UpdateRouteConfiguration(sourceAggregatedRoutesByDomain, sourceRouteConfig, true, false)
	destinationRouteConfig = route.UpdateRouteConfiguration(destinationAggregatedRoutesByDomain, destinationRouteConfig, false, true)
	routeConfiguration = append(routeConfiguration, sourceRouteConfig)
	routeConfiguration = append(routeConfiguration, destinationRouteConfig)

	for _, config := range routeConfiguration {
		marshalledRouteConfig, err := ptypes.MarshalAny(&config)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal route config for proxy")
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledRouteConfig)
	}
	return resp, nil
}

func aggregateRoutesByDomain(domainRoutesMap map[string]map[string]trafficpolicy.RouteWeightedClusters, routePolicies []trafficpolicy.Route, weightedCluster service.WeightedCluster, domain string) {
	_, exists := domainRoutesMap[domain]
	if !exists {
		// no domain found, create a new route map
		domainRoutesMap[domain] = make(map[string]trafficpolicy.RouteWeightedClusters)
	}
	for _, route := range routePolicies {
		routePolicyWeightedCluster, routeFound := domainRoutesMap[domain][route.PathRegex]
		if routeFound {
			// add the cluster to the existing route
			routePolicyWeightedCluster.WeightedClusters.Add(weightedCluster)
			routePolicyWeightedCluster.Route.Methods = append(routePolicyWeightedCluster.Route.Methods, route.Methods...)
			domainRoutesMap[domain][route.PathRegex] = routePolicyWeightedCluster
		} else {
			// no route found, create a new route and cluster mapping on domain
			domainRoutesMap[domain][route.PathRegex] = createRoutePolicyWeightedClusters(route, weightedCluster)
		}
	}
}

func createRoutePolicyWeightedClusters(routePolicy trafficpolicy.Route, weightedCluster service.WeightedCluster) trafficpolicy.RouteWeightedClusters {
	return trafficpolicy.RouteWeightedClusters{
		Route:            routePolicy,
		WeightedClusters: set.NewSet(weightedCluster),
	}
}

func updateRoutesForIngress(proxyServiceName service.NamespacedService, catalog catalog.MeshCataloger, domainRoutesMap map[string]map[string]trafficpolicy.RouteWeightedClusters) error {
	domainRoutePoliciesMap, err := catalog.GetIngressRoutePoliciesPerDomain(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress route configuration for proxy %s", proxyServiceName)
		return err
	}
	if len(domainRoutePoliciesMap) == 0 {
		return nil
	}

	ingressWeightedCluster, err := catalog.GetIngressWeightedCluster(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get weighted ingress clusters for proxy %s", proxyServiceName)
		return err
	}
	for domain, routePolicies := range domainRoutePoliciesMap {
		aggregateRoutesByDomain(domainRoutesMap, routePolicies, ingressWeightedCluster, domain)
	}
	return nil
}
