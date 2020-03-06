package rds

import (
	"context"
	"reflect"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/route"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

// NewResponse creates a new Route Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	proxyServiceName := proxy.GetService()
	allTrafficPolicies, err := catalog.ListTrafficRoutes(proxyServiceName)
	if err != nil {
		glog.Errorf("[%s] Failed listing routes: %+v", packageName, err)
		return nil, err
	}
	glog.V(level.Debug).Infof("[%s] trafficPolicies: %+v", packageName, allTrafficPolicies)

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
			glog.Errorf("[%s] Failed to marshal route config for proxy %v", packageName, err)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledRouteConfig)
	}
	return resp, nil
}
