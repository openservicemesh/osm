package rds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/route"

	"github.com/open-service-mesh/osm/pkg/smi"
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
	sourceAggregatedRoutesByDomain := make(map[string][]endpoint.RoutePolicyWeightedClusters)
	destinationAggregatedRoutesByDomain := make(map[string][]endpoint.RoutePolicyWeightedClusters)

	for _, trafficPolicies := range allTrafficPolicies {
		isSourceService := envoy.Contains(proxyServiceName, trafficPolicies.Source.Services)
		isDestinationService := envoy.Contains(proxyServiceName, trafficPolicies.Destination.Services)
		for _, service := range trafficPolicies.Destination.Services {
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
				aggregateRoutesByDomain(sourceAggregatedRoutesByDomain, trafficPolicies.RoutePolicies, weightedCluster, domain)
			}
			if isDestinationService {
				aggregateRoutesByDomain(destinationAggregatedRoutesByDomain, trafficPolicies.RoutePolicies, weightedCluster, domain)
			}
		}
	}

	// Process ingress policy if applicable
	if err = updateRoutesForIngress(proxyServiceName, catalog, destinationAggregatedRoutesByDomain); err != nil {
		return nil, err
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

func aggregateRoutesByDomain(domainRoutesMap map[string][]endpoint.RoutePolicyWeightedClusters, routePolicies []endpoint.RoutePolicy, weightedCluster endpoint.WeightedCluster, domain string) {
	routesList, exists := domainRoutesMap[domain]
	if !exists {
		// no domain found, create a new route and cluster mapping and add the domain
		var routeWeightedClustersList []endpoint.RoutePolicyWeightedClusters
		for _, route := range routePolicies {
			routeWeightedClustersList = append(routeWeightedClustersList, createRoutePolicyWeightedClusters(route, weightedCluster))
		}
		domainRoutesMap[domain] = routeWeightedClustersList
	} else {
		for _, route := range routePolicies {
			routeIndex, routeFound := routeExits(routesList, route)
			if routeFound {
				// add the cluster to the existing route
				routesList[routeIndex].WeightedClusters = append(routesList[routeIndex].WeightedClusters, weightedCluster)
				routesList[routeIndex].RoutePolicy.Methods = append(routesList[routeIndex].RoutePolicy.Methods, route.Methods...)
			} else {
				// no route found, create a new route and cluster mapping on domain
				routesList = append(routesList, createRoutePolicyWeightedClusters(route, weightedCluster))
			}
			domainRoutesMap[domain] = routesList
		}
	}
}

func createRoutePolicyWeightedClusters(routePolicy endpoint.RoutePolicy, weightedCluster endpoint.WeightedCluster) endpoint.RoutePolicyWeightedClusters {
	return endpoint.RoutePolicyWeightedClusters{
		RoutePolicy:      routePolicy,
		WeightedClusters: []endpoint.WeightedCluster{weightedCluster},
	}
}

func routeExits(routesList []endpoint.RoutePolicyWeightedClusters, routePolicy endpoint.RoutePolicy) (int, bool) {
	routeExists := false
	index := -1
	// check if the route is already in list
	for i, routeWeightedClusters := range routesList {
		if routeWeightedClusters.RoutePolicy.PathRegex == routePolicy.PathRegex {
			routeExists = true
			index = i
			return index, routeExists
		}
	}
	return index, routeExists
}

func updateRoutesForIngress(proxyServiceName endpoint.NamespacedService, catalog catalog.MeshCataloger, domainRoutesMap map[string][]endpoint.RoutePolicyWeightedClusters) error {
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
