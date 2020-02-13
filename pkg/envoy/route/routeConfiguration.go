package route

import (
	"strings"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/golang/glog"

	smcEndpoint "github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log"
)

const (
	// TypeRDS is the string constant of the Route Configuration URI

	// DestinationRouteConfig is the name of the route config that the envoy will identify
	DestinationRouteConfig = "RDS_Destination"
	// SourceRouteConfig is the name of the route config that the envoy will identify
	SourceRouteConfig = "RDS_Source"
)

//NewRouteConfiguration consrtucts the Envoy construct necessary for TrafficTarget implementation
// todo (snchh) : need to figure out linking to name spaces
func NewRouteConfiguration(trafficPolicies smcEndpoint.TrafficTargetPolicies) []v2.RouteConfiguration {

	routeConfigurations := []v2.RouteConfiguration{}
	serverRouteConfig := getServerRouteConfiguration(trafficPolicies)
	routeConfigurations = append(routeConfigurations, serverRouteConfig)
	clientRouteConfig := getClientRouteConfiguration(trafficPolicies)
	routeConfigurations = append(routeConfigurations, clientRouteConfig)
	return routeConfigurations
}

func getServerRouteConfiguration(trafficPolicies smcEndpoint.TrafficTargetPolicies) v2.RouteConfiguration {

	routeConfig := v2.RouteConfiguration{
		Name: DestinationRouteConfig,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "backend",
			Domains: []string{"*"},
			Routes:  []*route.Route{},
			Cors:    &route.CorsPolicy{},
		}},
	}

	for _, routePaths := range trafficPolicies.PolicyRoutePaths {
		rt := route.Route{
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: routePaths.RoutePathRegex,
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: trafficPolicies.Destination,
					},
				},
			},
		}
		cors := &route.CorsPolicy{
			AllowMethods: strings.Join(routePaths.RouteMethods, ", "),
		}

		routeConfig.VirtualHosts[0].Cors = cors
		routeConfig.VirtualHosts[0].Routes = append(routeConfig.VirtualHosts[0].Routes, &rt)
	}
	glog.V(log.LvlTrace).Infof("[RDS] Constructed Server RouteConfiguration: %+v", routeConfig)
	return routeConfig
}

func getClientRouteConfiguration(trafficPolicies smcEndpoint.TrafficTargetPolicies) v2.RouteConfiguration {
	routeConfig := v2.RouteConfiguration{
		Name: SourceRouteConfig,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "envoy_admin",
			Domains: []string{"*"},
			Routes:  []*route.Route{},
			Cors:    &route.CorsPolicy{},
		}},
	}

	for _, routePaths := range trafficPolicies.PolicyRoutePaths {
		rt := route.Route{
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: routePaths.RoutePathRegex,
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: trafficPolicies.Source,
					},
				},
			},
		}

		cors := &route.CorsPolicy{
			AllowMethods: strings.Join(routePaths.RouteMethods, ", "),
		}

		routeConfig.VirtualHosts[0].Cors = cors
		routeConfig.VirtualHosts[0].Routes = append(routeConfig.VirtualHosts[0].Routes, &rt)
	}
	glog.Infof("[RDS] Constructed Client RouteConfiguration: %+v", routeConfig)
	return routeConfig
}
