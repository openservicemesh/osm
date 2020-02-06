package route

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/wrappers"

	smcEndpoint "github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log"
)

const (
	// RouteConfigurationURI is the string constant of the Route Configuration URI
	RouteConfigurationURI = "type.googleapis.com/envoy.api.v2.RouteConfiguration"
)

//NewRouteConfiguration consrtucts the Envoy construct necessary for TrafficTarget implementation
// todo (snchh) : need to figure out linking to name spaces
// todo (snchh) : trafficPolicies.PolicyRoutePaths.RoutePathMethods not used
func NewRouteConfiguration(trafficPolicies smcEndpoint.TrafficTargetPolicies) v2.RouteConfiguration {

	routeConfiguration := v2.RouteConfiguration{
		Name: trafficPolicies.PolicyName,
		VirtualHosts: []*route.VirtualHost{{
			Name:    trafficPolicies.Source,
			Domains: []string{"*"},
			Routes:  []*route.Route{},
		}},
	}

	for _, routePaths := range trafficPolicies.PolicyRoutePaths {
		rt := route.Route{
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: routePaths.RoutePathRegex,
				},
				// TODO - should we keep this?? -- Grpc: &route.RouteMatch_GrpcRouteMatchOptions{},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					// TODO -- should we keep this -- ClusterNotFoundResponseCode: route.RouteAction_SERVICE_UNAVAILABLE,
					/*
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: trafficPolicies.Destination,
						},
					*/
					ClusterSpecifier: &route.RouteAction_WeightedClusters{
						WeightedClusters: &route.WeightedCluster{
							Clusters: []*route.WeightedCluster_ClusterWeight{{
								Name: "bookstore.mesh",
								Weight: &wrappers.UInt32Value{
									Value: 100,
								},
							}},
						},
					},
				},
			},
		}
		routeConfiguration.VirtualHosts[0].Routes = append(routeConfiguration.VirtualHosts[0].Routes, &rt)
	}

	glog.V(log.LvlTrace).Infof("[RDS] Constructed RouteConfiguration: %+v", routeConfiguration)
	return routeConfiguration
}

func GetServerRouteConfiguration() v2.RouteConfiguration {
	routeConfig := v2.RouteConfiguration{
		Name: "TODO_RDS",
		VirtualHosts: []*route.VirtualHost{{
			Name:    "backend",
			Domains: []string{"*"},
			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: "bookstore-local",
						},
						// PrefixRewrite: "/stats/prometheus", // well-known Admin API endpoint
					},
				},
			}},
		}},
	}
	return routeConfig
}

func GetClientRouteConfiguration() v2.RouteConfiguration {
	routeConfig := v2.RouteConfiguration{
		Name: "TODO_RDS",
		VirtualHosts: []*route.VirtualHost{{
			Name:    "envoy_admin",
			Domains: []string{"*"},
			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: "bookstore.mesh",
						},
						// PrefixRewrite: "/stats/prometheus", // well-known Admin API endpoint
					},
				},
			}},
		}},
	}

	return routeConfig
}
