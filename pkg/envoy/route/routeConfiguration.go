package route

import (
	"sort"
	"strings"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

const (
	//InboundRouteConfig is the name of the route config that the envoy will identify
	InboundRouteConfig = "RDS_Inbound"

	//OutboundRouteConfig is the name of the route config that the envoy will identify
	OutboundRouteConfig = "RDS_Outbound"
)

//UpdateRouteConfiguration consrtucts the Envoy construct necessary for TrafficTarget implementation
func UpdateRouteConfiguration(trafficPolicies endpoint.TrafficTargetPolicies, routeConfig v2.RouteConfiguration, isSourceService bool, isDestinationService bool) v2.RouteConfiguration {
	log.Trace().Msgf("[RDS] Updating Route Configuration")
	var routeConfiguration v2.RouteConfiguration
	var isLocalCluster bool
	if isSourceService {
		log.Trace().Msgf("[RDS] Updating OutboundRouteConfiguration for policy %v", trafficPolicies)
		isLocalCluster = false
		routeConfiguration = updateRoutes(trafficPolicies.PolicyRoutePaths, trafficPolicies.Source.Clusters, routeConfig, isLocalCluster)
	} else if isDestinationService {
		log.Trace().Msgf("[RDS] Updating InboundRouteConfiguration for policy %v", trafficPolicies)
		isLocalCluster = true
		routeConfiguration = updateRoutes(trafficPolicies.PolicyRoutePaths, trafficPolicies.Destination.Clusters, routeConfig, isLocalCluster)
	}
	return routeConfiguration
}

func updateRoutes(routePaths []endpoint.RoutePaths, cluster []endpoint.WeightedCluster, routeConfig v2.RouteConfiguration, isLocalCluster bool) v2.RouteConfiguration {
	allowedMethods := strings.Split(routeConfig.VirtualHosts[0].Cors.AllowMethods, ",")
	for _, path := range routePaths {
		routedMatched := false
		allowedMethods = append(allowedMethods, path.RouteMethods...)
		for i := 0; i < len(routeConfig.VirtualHosts[0].Routes); i++ {
			if path.RoutePathRegex == routeConfig.VirtualHosts[0].Routes[i].GetMatch().GetPrefix() {
				routedMatched = true
				routeConfig.VirtualHosts[0].Routes[i].Action = updateRouteActionWeightedClusters(*routeConfig.VirtualHosts[0].Routes[i].GetRoute().GetWeightedClusters(), cluster, isLocalCluster)
				continue
			}
		}
		if len(routeConfig.VirtualHosts[0].Routes) == 0 || !routedMatched {
			route := createRoute(path.RoutePathRegex, cluster, isLocalCluster)
			routeConfig.VirtualHosts[0].Routes = append(routeConfig.VirtualHosts[0].Routes, &route)
		}
	}
	allowedMethods = updateAllowedMethods(allowedMethods)
	routeConfig.VirtualHosts[0].Cors = &route.CorsPolicy{
		AllowMethods: strings.Join(allowedMethods, ","),
	}
	log.Debug().Msgf("[RDS] Constructed OutboundRouteConfiguration %+v", routeConfig)
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
			clusterName += envoy.LocalClusterSuffix
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
			clusterName += envoy.LocalClusterSuffix
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

func updateAllowedMethods(allowedMethods []string) []string {
	newAllowedMethods := []string{}
	keys := make(map[string]interface{})
	for _, method := range allowedMethods {
		if method != "" {
			if method == "*" {
				newAllowedMethods = []string{"*"}
				return newAllowedMethods
			}
			if _, value := keys[method]; !value {
				keys[method] = nil
				newAllowedMethods = append(newAllowedMethods, method)
			}
		}
	}
	return newAllowedMethods
}

//NewOutboundRouteConfiguration creates the outbound route configurations
func NewOutboundRouteConfiguration() v2.RouteConfiguration {
	routeConfiguration := v2.RouteConfiguration{
		Name: OutboundRouteConfig,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "envoy_admin",
			Domains: []string{"*"},
			Routes:  []*route.Route{},
			Cors:    &route.CorsPolicy{},
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
			Cors:    &route.CorsPolicy{},
		}},
		ValidateClusters: &wrappers.BoolValue{Value: true},
	}
	return routeConfiguration
}
