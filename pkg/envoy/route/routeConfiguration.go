package route

import (
	"sort"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v2route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

const (
	//InboundRouteConfig is the name of the route config that the envoy will identify
	InboundRouteConfig = "RDS_Inbound"

	//OutboundRouteConfig is the name of the route config that the envoy will identify
	OutboundRouteConfig = "RDS_Outbound"

	// maxRegexProgramSize is the max supported regex complexity
	maxRegexProgramSize = 1024
)

var (
	regexEngine = &matcher.RegexMatcher_GoogleRe2{GoogleRe2: &matcher.RegexMatcher_GoogleRE2{
		MaxProgramSize: &wrappers.UInt32Value{
			Value: uint32(maxRegexProgramSize),
		},
	}}
)

//UpdateRouteConfiguration consrtucts the Envoy construct necessary for TrafficTarget implementation
func UpdateRouteConfiguration(trafficPolicies endpoint.TrafficPolicy, routeConfig v2.RouteConfiguration, isSourceService bool, isDestinationService bool) v2.RouteConfiguration {
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
	for _, path := range routePaths {
		routedMatched := false
		for i := 0; i < len(routeConfig.VirtualHosts[0].Routes); i++ {
			if path.RoutePathRegex == routeConfig.VirtualHosts[0].Routes[i].GetMatch().GetPrefix() {
				routedMatched = true
				routeConfig.VirtualHosts[0].Routes[i].Action = updateRouteActionWeightedClusters(*routeConfig.VirtualHosts[0].Routes[i].GetRoute().GetWeightedClusters(), cluster, isLocalCluster)
				continue
			}
		}
		if len(routeConfig.VirtualHosts[0].Routes) == 0 || !routedMatched {
			route := createRoute(&path, cluster, isLocalCluster)
			routeConfig.VirtualHosts[0].Routes = append(routeConfig.VirtualHosts[0].Routes, &route)
		}
	}
	log.Debug().Msgf("[RDS] Constructed OutboundRouteConfiguration %+v", routeConfig)
	return routeConfig
}

func createRoute(path *endpoint.RoutePaths, weightedClusters []endpoint.WeightedCluster, isLocalCluster bool) v2route.Route {
	route := v2route.Route{
		Match: &v2route.RouteMatch{
			PathSpecifier: &v2route.RouteMatch_SafeRegex{
				SafeRegex: &matcher.RegexMatcher{
					EngineType: regexEngine,
					Regex:      path.RoutePathRegex,
				},
			},
		},
		Action: &v2route.Route_Route{
			Route: &v2route.RouteAction{
				ClusterSpecifier: &v2route.RouteAction_WeightedClusters{
					WeightedClusters: getWeightedCluster(weightedClusters, isLocalCluster),
				},
			},
		},
	}

	// For a given route path, sanitize the methods in case there
	// is wildcard or if there are duplicates
	allowedMethods := sanitizeHTTPMethods(path.RouteMethods)
	for _, method := range allowedMethods {
		headerMatcher := &v2route.HeaderMatcher{
			Name: ":method",
			HeaderMatchSpecifier: &v2route.HeaderMatcher_ExactMatch{
				ExactMatch: method,
			},
		}
		route.Match.Headers = append(route.Match.Headers, headerMatcher)
	}
	return route
}

func getWeightedCluster(weightedClusters []endpoint.WeightedCluster, isLocalCluster bool) *v2route.WeightedCluster {
	var wc v2route.WeightedCluster
	var total int
	for _, cluster := range weightedClusters {
		clusterName := string(cluster.ClusterName)
		total += cluster.Weight
		if isLocalCluster {
			clusterName += envoy.LocalClusterSuffix
		}
		wc.Clusters = append(wc.Clusters, &v2route.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: &wrappers.UInt32Value{Value: uint32(cluster.Weight)},
		})
	}
	wc.TotalWeight = &wrappers.UInt32Value{Value: uint32(total)}
	sort.Stable(clusterWeightByName(wc.Clusters))
	return &wc
}

func updateRouteActionWeightedClusters(existingWeightedCluster v2route.WeightedCluster, newWeightedClusters []endpoint.WeightedCluster, isLocalCluster bool) *v2route.Route_Route {
	total := int(existingWeightedCluster.TotalWeight.GetValue())
	for _, cluster := range newWeightedClusters {
		clusterName := string(cluster.ClusterName)
		total += cluster.Weight
		if isLocalCluster {
			clusterName += envoy.LocalClusterSuffix
		}
		existingWeightedCluster.Clusters = append(existingWeightedCluster.Clusters, &v2route.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: &wrappers.UInt32Value{Value: uint32(cluster.Weight)},
		})
	}
	existingWeightedCluster.TotalWeight = &wrappers.UInt32Value{Value: uint32(total)}
	sort.Stable(clusterWeightByName(existingWeightedCluster.Clusters))
	action := &v2route.Route_Route{
		Route: &v2route.RouteAction{
			ClusterSpecifier: &v2route.RouteAction_WeightedClusters{
				WeightedClusters: &existingWeightedCluster,
			},
		},
	}
	return action
}

type clusterWeightByName []*v2route.WeightedCluster_ClusterWeight

func (c clusterWeightByName) Len() int      { return len(c) }
func (c clusterWeightByName) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c clusterWeightByName) Less(i, j int) bool {
	if c[i].Name == c[j].Name {
		return c[i].Weight.Value < c[j].Weight.Value
	}
	return c[i].Name < c[j].Name
}

// sanitizeHTTPMethods takes in a list of HTTP methods including a wildcard (*) and returns a wildcard if any of
// the methods is a wildcard or sanitizes the input list to avoid duplicates.
func sanitizeHTTPMethods(allowedMethods []string) []string {
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
		VirtualHosts: []*v2route.VirtualHost{{
			Name:    "envoy_admin",
			Domains: []string{"*"},
			Routes:  []*v2route.Route{},
			Cors:    &v2route.CorsPolicy{},
		}},
		ValidateClusters: &wrappers.BoolValue{Value: true},
	}
	return routeConfiguration
}

//NewInboundRouteConfiguration creates the inbound route configurations
func NewInboundRouteConfiguration() v2.RouteConfiguration {
	routeConfiguration := v2.RouteConfiguration{
		Name: InboundRouteConfig,
		VirtualHosts: []*v2route.VirtualHost{{
			Name:    "backend",
			Domains: []string{"*"},
			Routes:  []*v2route.Route{},
			Cors:    &v2route.CorsPolicy{},
		}},
		ValidateClusters: &wrappers.BoolValue{Value: true},
	}
	return routeConfiguration
}
