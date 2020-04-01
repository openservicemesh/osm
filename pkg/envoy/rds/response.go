package rds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/route"

	"github.com/open-service-mesh/osm/pkg/smi"
)

type empty struct{}

// NewResponse creates a new Route Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	proxyServiceName := proxy.GetService()
	allTrafficPolicies, err := catalog.ListTrafficRoutes(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("[%s] Failed listing routes", packageName)
		return nil, err
	}
	log.Debug().Msgf("[%s] trafficPolicies: %+v", packageName, allTrafficPolicies)

	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	routeConfiguration := []xds.RouteConfiguration{}
	sourceRouteConfig := route.NewOutboundRouteConfiguration()
	destinationRouteConfig := route.NewInboundRouteConfiguration()
	for _, trafficPolicies := range allTrafficPolicies {
		isSourceService := envoy.Contains(proxyServiceName, trafficPolicies.Source.Services)
		isDestinationService := envoy.Contains(proxyServiceName, trafficPolicies.Destination.Services)
		if isSourceService {
			sourceRouteConfig = route.UpdateRouteConfiguration(trafficPolicies, sourceRouteConfig, isSourceService, isDestinationService)
		} else if isDestinationService {
			destinationRouteConfig = route.UpdateRouteConfiguration(trafficPolicies, destinationRouteConfig, isSourceService, isDestinationService)
		}
	}
	routeConfiguration = append(routeConfiguration, sourceRouteConfig)
	routeConfiguration = append(routeConfiguration, destinationRouteConfig)

	for _, config := range routeConfiguration {
		marshalledRouteConfig, err := ptypes.MarshalAny(&config)
		if err != nil {
			log.Error().Err(err).Msgf("[%s] Failed to marshal route config for proxy", packageName)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledRouteConfig)
	}
	return resp, nil
}
