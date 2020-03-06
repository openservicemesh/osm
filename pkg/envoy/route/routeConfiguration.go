package route

import (
	"sort"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/deislabs/smc/pkg/endpoint"
	smcEndpoint "github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
)

const (
	//InboundRouteConfig is the name of the route config that the envoy will identify
	InboundRouteConfig = "RDS_Inbound"

	//OutboundRouteConfig is the name of the route config that the envoy will identify
	OutboundRouteConfig = "RDS_Outbound"
)

//UpdateRouteConfiguration consrtucts the Envoy construct necessary for TrafficTarget implementation
func UpdateRouteConfiguration(trafficPolicies smcEndpoint.TrafficTargetPolicies, routeConfig v2.RouteConfiguration, isSourceService bool, isDestinationService bool) v2.RouteConfiguration {
	glog.V(level.Trace).Infof("[RDS] Updating Route Configuration")
	var routeConfiguration v2.RouteConfiguration
	if isSourceService {
		routeConfiguration = updateOutboundRouteConfiguration(trafficPolicies, routeConfig)
	} else if isDestinationService {
		routeConfiguration = updateInboundRouteConfiguration(trafficPolicies, routeConfig)
	}
	return routeConfiguration
}

func updateInboundRouteConfiguration(trafficPolicies smcEndpoint.TrafficTargetPolicies, routeConfig v2.RouteConfiguration) v2.RouteConfiguration {
	// todo (sneha) : update cors policy
	//allowedMethods := string.Trim(strings.Split(routeConfig.VirtualHosts[0].Cors.AllowMethods, ","))
	glog.V(level.Trace).Infof("[RDS] Updating InboundRouteConfiguration for policy %v", trafficPolicies)
	numberofRoutes := len(routeConfig.VirtualHosts[0].Routes)
	for _, routePaths := range trafficPolicies.PolicyRoutePaths {
		if numberofRoutes == 0 {
			route := createRoute(routePaths.RoutePathRegex, trafficPolicies.Destination.Clusters, true)
			//allowedMethods = append(allowedMethods, routePaths.RouteMethods...)
			routeConfig.VirtualHosts[0].Routes = append(routeConfig.VirtualHosts[0].Routes, &route)
			continue
		}
		for i := 0; i < numberofRoutes; i++ {
			if routePaths.RoutePathRegex == routeConfig.VirtualHosts[0].Routes[i].GetMatch().GetPrefix() {
				routeConfig.VirtualHosts[0].Routes[i].Action = updateRouteActionWeightedClusters(*routeConfig.VirtualHosts[0].Routes[i].GetRoute().GetWeightedClusters(), trafficPolicies.Source.Clusters, false)
			} else {
				route := createRoute(routePaths.RoutePathRegex, trafficPolicies.Destination.Clusters, false)
				//allowedMethods = append(allowedMethods, routePaths.RouteMethods...)
				routeConfig.VirtualHosts[0].Routes = append(routeConfig.VirtualHosts[0].Routes, &route)
			}
		}
	}
	routeConfig.VirtualHosts[0].Cors = &route.CorsPolicy{
		AllowMethods: "GET",
	}
	glog.V(level.Debug).Infof("[RDS] Constructed InboundRouteConfiguration %+v", routeConfig)
	return routeConfig
}

func updateOutboundRouteConfiguration(trafficPolicies smcEndpoint.TrafficTargetPolicies, routeConfig v2.RouteConfiguration) v2.RouteConfiguration {
	// todo (sneha) : update cors policy
	//allowedMethods := strings.Split(routeConfig.VirtualHosts[0].Cors.AllowMethods, ",")
	glog.V(level.Trace).Infof("[RDS] Updating OutboundRouteConfiguration for policy %v", trafficPolicies)
	for _, routePaths := range trafficPolicies.PolicyRoutePaths {
		if len(routeConfig.VirtualHosts[0].Routes) == 0 {
			route := createRoute(routePaths.RoutePathRegex, trafficPolicies.Source.Clusters, false)
			//allowedMethods = append(allowedMethods, routePaths.RouteMethods...)
			routeConfig.VirtualHosts[0].Routes = append(routeConfig.VirtualHosts[0].Routes, &route)
			continue
		}
		for i := 0; i < len(routeConfig.VirtualHosts[0].Routes); i++ {
			if routePaths.RoutePathRegex == routeConfig.VirtualHosts[0].Routes[i].GetMatch().GetPrefix() {
				routeConfig.VirtualHosts[0].Routes[i].Action = updateRouteActionWeightedClusters(*routeConfig.VirtualHosts[0].Routes[i].GetRoute().GetWeightedClusters(), trafficPolicies.Source.Clusters, false)
			} else {
				route := createRoute(routePaths.RoutePathRegex, trafficPolicies.Source.Clusters, false)
				//allowedMethods = append(allowedMethods, routePaths.RouteMethods...)
				routeConfig.VirtualHosts[0].Routes = append(routeConfig.VirtualHosts[0].Routes, &route)
			}
		}
	}
	routeConfig.VirtualHosts[0].Cors = &route.CorsPolicy{
		AllowMethods: "GET",
	}
	glog.V(level.Debug).Infof("[RDS] Constructed OutboundRouteConfiguration %+v", routeConfig)
	return routeConfig
}

func createRoute(pathPrefix string, weightedClusters []endpoint.WeightedCluster, isLocalCluster bool) route.Route {
	route := route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: pathPrefix,
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_WeightedClusters{
					WeightedClusters: getWeightedCluster(weightedClusters, isLocalCluster),
				},
			},
		},
	}
	return route
}

func getWeightedCluster(weightedClusters []endpoint.WeightedCluster, isLocalCluster bool) *route.WeightedCluster {
	var wc route.WeightedCluster
	var total int
	for _, cluster := range weightedClusters {
		clusterName := string(cluster.ClusterName)
		total += cluster.Weight
		if isLocalCluster {
			clusterName += envoy.LocalCluster
		}
		wc.Clusters = append(wc.Clusters, &route.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: &wrappers.UInt32Value{Value: uint32(cluster.Weight)},
		})
	}
	wc.TotalWeight = &wrappers.UInt32Value{Value: uint32(total)}
	sort.Stable(clusterWeightByName(wc.Clusters))
	return &wc
}

func updateRouteActionWeightedClusters(existingWeightedCluster route.WeightedCluster, newWeightedClusters []endpoint.WeightedCluster, isLocalCluster bool) *route.Route_Route {
	total := int(existingWeightedCluster.TotalWeight.GetValue())
	for _, cluster := range newWeightedClusters {
		clusterName := string(cluster.ClusterName)
		total += cluster.Weight
		if isLocalCluster {
			clusterName += envoy.LocalCluster
		}
		existingWeightedCluster.Clusters = append(existingWeightedCluster.Clusters, &route.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: &wrappers.UInt32Value{Value: uint32(cluster.Weight)},
		})
	}
	existingWeightedCluster.TotalWeight = &wrappers.UInt32Value{Value: uint32(total)}
	sort.Stable(clusterWeightByName(existingWeightedCluster.Clusters))
	action := &route.Route_Route{
		Route: &route.RouteAction{
			ClusterSpecifier: &route.RouteAction_WeightedClusters{
				WeightedClusters: &existingWeightedCluster,
			},
		},
	}
	return action
}

type clusterWeightByName []*route.WeightedCluster_ClusterWeight

func (c clusterWeightByName) Len() int      { return len(c) }
func (c clusterWeightByName) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c clusterWeightByName) Less(i, j int) bool {
	if c[i].Name == c[j].Name {
		return c[i].Weight.Value < c[j].Weight.Value
	}
	return c[i].Name < c[j].Name
}

//NewOutboundRouteConfiguration creates the outbound route configurations
func NewOutboundRouteConfiguration() v2.RouteConfiguration {
	routeConfiguration := v2.RouteConfiguration{
		Name: OutboundRouteConfig,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "envoy_admin",
			Domains: []string{"*"},
			Routes:  []*route.Route{},
			//Cors:    &route.CorsPolicy{},
		}},
		ValidateClusters: &wrappers.BoolValue{Value: true},
	}
	return routeConfiguration
}

//NewInboundRouteConfiguration creates the inbound route configurations
func NewInboundRouteConfiguration() v2.RouteConfiguration {
	routeConfiguration := v2.RouteConfiguration{
		Name: InboundRouteConfig,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "backend",
			Domains: []string{"*"},
			Routes:  []*route.Route{},
			//Cors:    &route.CorsPolicy{},
		}},
		ValidateClusters: &wrappers.BoolValue{Value: true},
	}
	return routeConfiguration
}
