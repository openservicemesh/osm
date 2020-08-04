package route

import (
	"fmt"
	"sort"
	"strings"

	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"

	set "github.com/deckarep/golang-set"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	//InboundRouteConfigName is the name of the route config that the envoy will identify
	InboundRouteConfigName = "RDS_Inbound"

	//OutboundRouteConfigName is the name of the route config that the envoy will identify
	OutboundRouteConfigName = "RDS_Outbound"

	// maxRegexProgramSize is the max supported regex complexity
	maxRegexProgramSize = 1024

	//InboundVirtualHost is the name of the virtual host on the inbound route configuration
	inboundVirtualHost = "inbound_virtualHost"

	//OutboundVirtualHost is the name of the virtual host on the outbound route configuration
	outboundVirtualHost = "outbound_virtualHost"

	// MethodHeaderKey is the key of the header for HTTP methods
	MethodHeaderKey = ":method"

	// wildcardPathPrefix is the wildcard path prefix for HTTP paths
	wildcardPathPrefix = "/"

	// httpHostHeader is the name of the HTTP host header
	httpHostHeader = "host"
)

var (
	regexEngine = &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{
		MaxProgramSize: &wrappers.UInt32Value{
			Value: uint32(maxRegexProgramSize),
		},
	}}
)

//UpdateRouteConfiguration consrtucts the Envoy construct necessary for TrafficTarget implementation
func UpdateRouteConfiguration(domainRoutesMap map[string]map[string]trafficpolicy.RouteWeightedClusters, routeConfig *xds_route.RouteConfiguration, isSourceConfig bool, isDestinationConfig bool) {
	log.Trace().Msgf("[RDS] Updating Route Configuration")
	var isLocalCluster bool
	var virtualHostPrefix string

	if isSourceConfig {
		log.Trace().Msgf("[RDS] Updating OutboundRouteConfiguration for policy %v", domainRoutesMap)
		isLocalCluster = false
		virtualHostPrefix = outboundVirtualHost
	} else if isDestinationConfig {
		log.Trace().Msgf("[RDS] Updating InboundRouteConfiguration for policy %v", domainRoutesMap)
		isLocalCluster = true
		virtualHostPrefix = inboundVirtualHost
	}
	for domain, routePolicyWeightedClustersMap := range domainRoutesMap {
		virtualHost := createVirtualHostStub(virtualHostPrefix, domain)
		virtualHost.Routes = createRoutes(routePolicyWeightedClustersMap, isLocalCluster)
		routeConfig.VirtualHosts = append(routeConfig.VirtualHosts, virtualHost)
	}
}

func createVirtualHostStub(namePrefix string, domain string) *xds_route.VirtualHost {
	// If domain consists a comma separated list of domains, it means multiple
	// domains match against the same route config.
	domains := strings.Split(domain, ",")
	for i := range domains {
		domains[i] = strings.TrimSpace(domains[i])
	}

	name := fmt.Sprintf("%s|%s", namePrefix, kubernetes.GetServiceFromHostname(domains[0]))
	virtualHost := xds_route.VirtualHost{
		Name:    name,
		Domains: domains,
		Routes:  []*xds_route.Route{},
	}
	return &virtualHost
}

func createRoutes(routePolicyWeightedClustersMap map[string]trafficpolicy.RouteWeightedClusters, isLocalCluster bool) []*xds_route.Route {
	var routes []*xds_route.Route
	if !isLocalCluster {
		// For a source service, configure a wildcard route match (without any headers) with weighted routes to upstream clusters based on traffic split policies
		weightedClusters := getDistinctWeightedClusters(routePolicyWeightedClustersMap)
		totalClustersWeight := getTotalWeightForClusters(weightedClusters)
		emptyHeaders := make(map[string]string)
		route := getRoute(constants.RegexMatchAll, constants.WildcardHTTPMethod, emptyHeaders, weightedClusters, totalClustersWeight, isLocalCluster)
		routes = append(routes, route)
		return routes
	}
	for _, routePolicyWeightedClusters := range routePolicyWeightedClustersMap {
		// For a given route path, sanitize the methods in case there
		// is wildcard or if there are duplicates
		allowedMethods := sanitizeHTTPMethods(routePolicyWeightedClusters.Route.Methods)
		for _, method := range allowedMethods {
			route := getRoute(routePolicyWeightedClusters.Route.PathRegex, method, routePolicyWeightedClusters.Route.Headers, routePolicyWeightedClusters.WeightedClusters, 100, isLocalCluster)
			routes = append(routes, route)
		}
	}
	return routes
}

func getRoute(pathRegex string, method string, headersMap map[string]string, weightedClusters set.Set, totalClustersWeight int, isLocalCluster bool) *xds_route.Route {
	route := xds_route.Route{
		Match: &xds_route.RouteMatch{
			PathSpecifier: &xds_route.RouteMatch_SafeRegex{
				SafeRegex: &xds_matcher.RegexMatcher{
					EngineType: regexEngine,
					Regex:      pathRegex,
				},
			},
			Headers: getHeadersForRoute(method, headersMap),
		},
		Action: &xds_route.Route_Route{
			Route: &xds_route.RouteAction{
				ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
					WeightedClusters: getWeightedCluster(weightedClusters, totalClustersWeight, isLocalCluster),
				},
			},
		},
	}
	return &route
}

func getHeadersForRoute(method string, headersMap map[string]string) []*xds_route.HeaderMatcher {
	var headers []*xds_route.HeaderMatcher

	// add methods header
	methodsHeader := xds_route.HeaderMatcher{
		Name: MethodHeaderKey,
		HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
			SafeRegexMatch: &xds_matcher.RegexMatcher{
				EngineType: regexEngine,
				Regex:      getRegexForMethod(method),
			},
		},
	}
	headers = append(headers, &methodsHeader)

	// add all other custom headers
	for headerKey, headerValue := range headersMap {
		// omit the host header as we have already configured this
		if headerKey == httpHostHeader {
			continue
		}
		header := xds_route.HeaderMatcher{
			Name: headerKey,
			HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
				SafeRegexMatch: &xds_matcher.RegexMatcher{
					EngineType: regexEngine,
					Regex:      headerValue,
				},
			},
		}
		headers = append(headers, &header)

	}
	return headers
}

func getWeightedCluster(weightedClusters set.Set, totalClustersWeight int, isLocalCluster bool) *xds_route.WeightedCluster {
	var wc xds_route.WeightedCluster
	var total int
	for clusterInterface := range weightedClusters.Iter() {
		cluster := clusterInterface.(service.WeightedCluster)
		clusterName := string(cluster.ClusterName)
		total += cluster.Weight
		if isLocalCluster {
			clusterName += envoy.LocalClusterSuffix
		}
		wc.Clusters = append(wc.Clusters, &xds_route.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: &wrappers.UInt32Value{Value: uint32(cluster.Weight)},
		})
	}
	if !isLocalCluster {
		// for source service, the pre-computed total weight based on traffic splits is used
		total = totalClustersWeight
	}
	wc.TotalWeight = &wrappers.UInt32Value{Value: uint32(total)}
	sort.Stable(clusterWeightByName(wc.Clusters))
	return &wc
}

// This method gets a list of all the distinct upstream clusters for a domain
// needed to configure source service's weighted routes
func getDistinctWeightedClusters(routePolicyWeightedClustersMap map[string]trafficpolicy.RouteWeightedClusters) set.Set {
	weightedClusters := set.NewSet()
	for _, perRouteWeightedClusters := range routePolicyWeightedClustersMap {
		if weightedClusters.Cardinality() == 0 {
			weightedClusters = perRouteWeightedClusters.WeightedClusters
		}
		weightedClusters.Union(perRouteWeightedClusters.WeightedClusters)
	}
	return weightedClusters
}

func getTotalWeightForClusters(weightedClusters set.Set) int {
	var totalWeight int
	for clusterInterface := range weightedClusters.Iter() {
		cluster := clusterInterface.(service.WeightedCluster)
		totalWeight += cluster.Weight
	}
	return totalWeight
}

type clusterWeightByName []*xds_route.WeightedCluster_ClusterWeight

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
func NewRouteConfigurationStub(routeConfigName string) *xds_route.RouteConfiguration {
	routeConfiguration := xds_route.RouteConfiguration{
		Name:             routeConfigName,
		VirtualHosts:     []*xds_route.VirtualHost{},
		ValidateClusters: &wrappers.BoolValue{Value: true},
	}
	return &routeConfiguration
}

func getRegexForMethod(httpMethod string) string {
	methodRegex := httpMethod
	if httpMethod == constants.WildcardHTTPMethod {
		methodRegex = constants.RegexMatchAll
	}
	return methodRegex
}
