package route

import (
	"fmt"
	"sort"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v2route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-service-mesh/osm/pkg/constants"
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

	//InboundVirtualHost is the name of the virtual host on the inbound route configuration
	inboundVirtualHost = "inbound_virtualHost"

	//OutboundVirtualHost is the name of the virtual host on the outbound route configuration
	outboundVirtualHost = "outbound_virtualHost"
)

var (
	regexEngine = &matcher.RegexMatcher_GoogleRe2{GoogleRe2: &matcher.RegexMatcher_GoogleRE2{
		MaxProgramSize: &wrappers.UInt32Value{
			Value: uint32(maxRegexProgramSize),
		},
	}}
)

//UpdateRouteConfiguration consrtucts the Envoy construct necessary for TrafficTarget implementation
func UpdateRouteConfiguration(domainRoutesMap map[string][]endpoint.RoutePolicyWeightedClusters, routeConfig v2.RouteConfiguration, isSourceConfig bool, isDestinationConfig bool) v2.RouteConfiguration {
	log.Trace().Msgf("[RDS] Updating Route Configuration")
	var isLocalCluster bool
	var virtualHostName string

	if isSourceConfig {
		log.Trace().Msgf("[RDS] Updating OutboundRouteConfiguration for policy %v", domainRoutesMap)
		isLocalCluster = false
		virtualHostName = outboundVirtualHost
	} else if isDestinationConfig {
		log.Trace().Msgf("[RDS] Updating InboundRouteConfiguration for policy %v", domainRoutesMap)
		isLocalCluster = true
		virtualHostName = inboundVirtualHost
	}
	for domain, routePolicyWeightedClustersList := range domainRoutesMap {
		virtualHost := createVirtualHostStub(fmt.Sprintf("%s|%s", virtualHostName, domain), domain)
		virtualHost.Routes = createRoutes(routePolicyWeightedClustersList, isLocalCluster)
		routeConfig.VirtualHosts = append(routeConfig.VirtualHosts, &virtualHost)
	}
	return routeConfig
}

func createVirtualHostStub(name string, domain string) v2route.VirtualHost {
	virtualHost := v2route.VirtualHost{
		Name:    name,
		Domains: []string{domain},
		Routes:  []*v2route.Route{},
	}
	return virtualHost
}

func createRoutes(routePolicyWeightedClustersList []endpoint.RoutePolicyWeightedClusters, isLocalCluster bool) []*v2route.Route {
	var routes []*v2route.Route
	// for source envoy configure for wild card routes and methods with weighted distribution on all clusters based on traffic split
	if !isLocalCluster {
		weightedClusters := getDistinctWeightedClusters(routePolicyWeightedClustersList)
		totalClustersWeight := getTotalWeightForClusters(weightedClusters)
		route := fillRouteObject(constants.RegexMatchAll, constants.WildcardHTTPMethod, weightedClusters, totalClustersWeight, isLocalCluster)
		routes = append(routes, &route)
		return routes
	}
	for _, routePolicyWeightedClusters := range routePolicyWeightedClustersList {
		// For a given route path, sanitize the methods in case there
		// is wildcard or if there are duplicates
		allowedMethods := sanitizeHTTPMethods(routePolicyWeightedClusters.RoutePolicy.Methods)
		for _, method := range allowedMethods {
			route := fillRouteObject(routePolicyWeightedClusters.RoutePolicy.PathRegex, method, routePolicyWeightedClusters.WeightedClusters, 100, isLocalCluster)
			routes = append(routes, &route)
		}
	}
	return routes
}

func fillRouteObject(pathRegex string, method string, weightedCLusters []endpoint.WeightedCluster, totalClustersWeight int, isLocalCluster bool) v2route.Route {
	route := v2route.Route{
		Match: &v2route.RouteMatch{
			PathSpecifier: &v2route.RouteMatch_SafeRegex{
				SafeRegex: &matcher.RegexMatcher{
					EngineType: regexEngine,
					Regex:      pathRegex,
				},
			},
			Headers: []*v2route.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &v2route.HeaderMatcher_SafeRegexMatch{
						SafeRegexMatch: &matcher.RegexMatcher{
							EngineType: regexEngine,
							Regex:      getRegexForMethod(method),
						},
					},
				},
			},
		},
		Action: &v2route.Route_Route{
			Route: &v2route.RouteAction{
				ClusterSpecifier: &v2route.RouteAction_WeightedClusters{
					WeightedClusters: getWeightedCluster(weightedCLusters, totalClustersWeight, isLocalCluster),
				},
			},
		},
	}
	return route
}

func getWeightedCluster(weightedClusters []endpoint.WeightedCluster, totalClustersWeight int, isLocalCluster bool) *v2route.WeightedCluster {
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
	if !isLocalCluster {
		total = totalClustersWeight
	}
	wc.TotalWeight = &wrappers.UInt32Value{Value: uint32(total)}
	sort.Stable(clusterWeightByName(wc.Clusters))
	return &wc
}

func getDistinctWeightedClusters(routePolicyWeightedClustersList []endpoint.RoutePolicyWeightedClusters) []endpoint.WeightedCluster {
	var weightedClusters []endpoint.WeightedCluster
	for _, perRouteWeightedClusters := range routePolicyWeightedClustersList {
		for _, cluster := range perRouteWeightedClusters.WeightedClusters {
			var clusterExists bool
			for _, existingCluster := range weightedClusters {
				if existingCluster.ClusterName == cluster.ClusterName {
					clusterExists = true
					continue
				}
			}
			if !clusterExists {
				weightedClusters = append(weightedClusters, cluster)
			}
		}
	}
	return weightedClusters
}

func getTotalWeightForClusters(weightedClusters []endpoint.WeightedCluster) int {
	var totalWeight int
	for _, cluster := range weightedClusters {
		totalWeight += cluster.Weight
	}
	return totalWeight
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
			if method == constants.WildcardHTTPMethod {
				newAllowedMethods = []string{constants.WildcardHTTPMethod}
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

//NewRouteConfigurationStub creates the route configuration placeholder
func NewRouteConfigurationStub(routeConfigName string) v2.RouteConfiguration {
	routeConfiguration := v2.RouteConfiguration{
		Name:             routeConfigName,
		VirtualHosts:     []*v2route.VirtualHost{},
		ValidateClusters: &wrappers.BoolValue{Value: true},
	}
	return routeConfiguration
}

func getRegexForMethod(httpMethod string) string {
	methodRegex := httpMethod
	if httpMethod == constants.WildcardHTTPMethod {
		methodRegex = constants.RegexMatchAll
	}
	return methodRegex
}
