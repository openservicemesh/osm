package rds

import (
	"context"

	set "github.com/deckarep/golang-set"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"
	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/route"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
)

// NewResponse creates a new Route Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest, cfg configurator.Configurator) (*xds.DiscoveryResponse, error) {
	svc, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	proxyServiceName := *svc

	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msg("Failed listing routes")
		return nil, err
	}
	log.Debug().Msgf("trafficPolicies: %+v", allTrafficPolicies)

	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	routeConfiguration := []*xds.RouteConfiguration{}
	sourceRouteConfig := route.NewRouteConfigurationStub(route.OutboundRouteConfigName)
	destinationRouteConfig := route.NewRouteConfigurationStub(route.InboundRouteConfigName)
	sourceAggregatedRoutesByDomain := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)
	destinationAggregatedRoutesByDomain := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)

	for _, trafficPolicies := range allTrafficPolicies {
		isSourceService := trafficPolicies.Source.Service.Equals(proxyServiceName)
		isDestinationService := trafficPolicies.Destination.Service.Equals(proxyServiceName)
		svc := trafficPolicies.Destination.Service
		domain, err := catalog.GetDomainForService(svc, trafficPolicies.Route.Headers)
		if err != nil {
			log.Error().Err(err).Msg("Failed listing domains")
			return nil, err
		}
		weightedCluster, err := catalog.GetWeightedClusterForService(svc)
		if err != nil {
			log.Error().Err(err).Msg("Failed listing clusters")
			return nil, err
		}

		if isSourceService {
			aggregateRoutesByHost(sourceAggregatedRoutesByDomain, trafficPolicies.Route, weightedCluster, domain)
		}

		if isDestinationService {
			aggregateRoutesByHost(destinationAggregatedRoutesByDomain, trafficPolicies.Route, weightedCluster, domain)
		}
	}

	if err = updateRoutesForIngress(proxyServiceName, catalog, destinationAggregatedRoutesByDomain); err != nil {
		return nil, err
	}

	route.UpdateRouteConfiguration(sourceAggregatedRoutesByDomain, sourceRouteConfig, true, false)
	route.UpdateRouteConfiguration(destinationAggregatedRoutesByDomain, destinationRouteConfig, false, true)
	routeConfiguration = append(routeConfiguration, sourceRouteConfig)
	routeConfiguration = append(routeConfiguration, destinationRouteConfig)

	for _, config := range routeConfiguration {
		marshalledRouteConfig, err := ptypes.MarshalAny(config)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal route config for proxy")
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledRouteConfig)
	}
	return resp, nil
}

func aggregateRoutesByHost(routesPerHost map[string]map[string]trafficpolicy.RouteWeightedClusters, routePolicy trafficpolicy.Route, weightedCluster service.WeightedCluster, host string) {
	_, exists := routesPerHost[host]
	if !exists {
		// no host found, create a new route map
		routesPerHost[host] = make(map[string]trafficpolicy.RouteWeightedClusters)
	}
	routePolicyWeightedCluster, routeFound := routesPerHost[host][routePolicy.PathRegex]
	if routeFound {
		// add the cluster to the existing route
		routePolicyWeightedCluster.WeightedClusters.Add(weightedCluster)
		routePolicyWeightedCluster.Route.Methods = append(routePolicyWeightedCluster.Route.Methods, routePolicy.Methods...)
		if routePolicyWeightedCluster.Route.Headers == nil {
			routePolicyWeightedCluster.Route.Headers = make(map[string]string)
		}
		for headerKey, headerValue := range routePolicy.Headers {
			routePolicyWeightedCluster.Route.Headers[headerKey] = headerValue
		}
		routesPerHost[host][routePolicy.PathRegex] = routePolicyWeightedCluster
	} else {
		// no route found, create a new route and cluster mapping on host
		routesPerHost[host][routePolicy.PathRegex] = createRoutePolicyWeightedClusters(routePolicy, weightedCluster)
	}
}

func createRoutePolicyWeightedClusters(routePolicy trafficpolicy.Route, weightedCluster service.WeightedCluster) trafficpolicy.RouteWeightedClusters {
	return trafficpolicy.RouteWeightedClusters{
		Route:            routePolicy,
		WeightedClusters: set.NewSet(weightedCluster),
	}
}
