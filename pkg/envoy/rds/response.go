package rds

import (
	"context"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	set "github.com/deckarep/golang-set"

	"github.com/golang/protobuf/ptypes"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// NewResponse creates a new Route Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, cfg configurator.Configurator) (*xds_discovery.DiscoveryResponse, error) {
	svc, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	proxyServiceName := *svc

	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msg("Failed listing routes")
		return nil, err
	}
	log.Debug().Msgf("trafficPolicies: %+v", allTrafficPolicies)

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	routeConfiguration := []*xds_route.RouteConfiguration{}
	sourceRouteConfig := route.NewRouteConfigurationStub(route.OutboundRouteConfigName)
	destinationRouteConfig := route.NewRouteConfigurationStub(route.InboundRouteConfigName)
	sourceAggregatedRoutesByDomain := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)
	destinationAggregatedRoutesByDomain := make(map[string]map[string]trafficpolicy.RouteWeightedClusters)

	for _, trafficPolicies := range allTrafficPolicies {
		isSourceService := trafficPolicies.Source.Equals(proxyServiceName)
		isDestinationService := trafficPolicies.Destination.Equals(proxyServiceName)
		svc := trafficPolicies.Destination
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
