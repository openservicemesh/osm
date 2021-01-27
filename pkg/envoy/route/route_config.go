package route

import (
	"fmt"
	"sort"

	set "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// BuildRouteConfiguration constructs the Envoy constructs ([]*xds_route.RouteConfiguration) for implementing inbound and outbound routes
func BuildRouteConfiguration(inbound []*trafficpolicy.InboundTrafficPolicy, outbound []*trafficpolicy.OutboundTrafficPolicy) []*xds_route.RouteConfiguration {
	routeConfiguration := []*xds_route.RouteConfiguration{}

	if len(inbound) > 0 {
		inboundRouteConfig := NewRouteConfigurationStub(InboundRouteConfigName)
		for _, in := range inbound {
			virtualHost := buildVirtualHostStub(inboundVirtualHost, in.Name, in.Hostnames)
			virtualHost.Routes = buildInboundRoutes(in.Rules)
			inboundRouteConfig.VirtualHosts = append(inboundRouteConfig.VirtualHosts, virtualHost)
		}

		routeConfiguration = append(routeConfiguration, inboundRouteConfig)
	}
	if len(outbound) > 0 {
		outboundRouteConfig := NewRouteConfigurationStub(OutboundRouteConfigName)

		for _, out := range outbound {
			virtualHost := buildVirtualHostStub(outboundVirtualHost, out.Name, out.Hostnames)
			virtualHost.Routes = buildOutboundRoutes(out.Routes)
			outboundRouteConfig.VirtualHosts = append(outboundRouteConfig.VirtualHosts, virtualHost)
		}
		routeConfiguration = append(routeConfiguration, outboundRouteConfig)
	}

	return routeConfiguration
}

func buildVirtualHostStub(namePrefix string, host string, domains []string) *xds_route.VirtualHost {
	name := fmt.Sprintf("%s|%s", namePrefix, host)
	virtualHost := xds_route.VirtualHost{
		Name:    name,
		Domains: domains,
	}
	return &virtualHost
}

// buildInboundRoutes takes a route information from the given inbound traffic policy and returns a list of xds routes
//	TODO: Currently, the information about which identity can access a particular route is not used but is passed in
func buildInboundRoutes(rules []*trafficpolicy.Rule) []*xds_route.Route {
	var routes []*xds_route.Route
	for _, rule := range rules {
		// For a given route path, sanitize the methods in case there
		// is wildcard or if there are duplicates
		allowedMethods := sanitizeHTTPMethods(rule.Route.HTTPRouteMatch.Methods)
		for _, method := range allowedMethods {
			route := buildRoute(rule.Route.HTTPRouteMatch.PathRegex, method, rule.Route.HTTPRouteMatch.Headers, rule.Route.WeightedClusters, 100, InboundRoute)
			routes = append(routes, route)
		}
	}
	return routes
}

func buildOutboundRoutes(outRoutes []*trafficpolicy.RouteWeightedClusters) []*xds_route.Route {
	var routes []*xds_route.Route
	for _, outRoute := range outRoutes {
		emptyHeaders := map[string]string{}
		// TODO: When implementing trafficsplit v1alpha4, buildRoute here should take in path, method, headers from trafficpolicy.HTTPRouteMatch
		routes = append(routes, buildRoute(constants.RegexMatchAll, constants.WildcardHTTPMethod, emptyHeaders, outRoute.WeightedClusters, outRoute.TotalClustersWeight(), OutboundRoute))
	}
	return routes
}

func buildRoute(pathRegex, method string, headersMap map[string]string, weightedClusters set.Set, totalWeight int, direction Direction) *xds_route.Route {
	route := xds_route.Route{
		Match: &xds_route.RouteMatch{
			PathSpecifier: &xds_route.RouteMatch_SafeRegex{
				SafeRegex: &xds_matcher.RegexMatcher{
					EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
					Regex:      pathRegex,
				},
			},
			Headers: getHeadersForRoute(method, headersMap),
		},
		Action: &xds_route.Route_Route{
			Route: &xds_route.RouteAction{
				ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
					WeightedClusters: buildWeightedCluster(weightedClusters, totalWeight, direction),
				},
			},
		},
	}
	return &route
}

func buildWeightedCluster(weightedClusters set.Set, totalWeight int, direction Direction) *xds_route.WeightedCluster {
	var wc xds_route.WeightedCluster
	var total int
	for clusterInterface := range weightedClusters.Iter() {
		cluster := clusterInterface.(service.WeightedCluster)
		clusterName := string(cluster.ClusterName)
		total += cluster.Weight
		if direction == InboundRoute {
			// An inbound route is associated with a local cluster. The inbound route is applied
			// on the destination cluster, and the destination clusters that accept inbound
			// traffic have the name of the form 'someClusterName-local`.
			clusterName = envoy.GetLocalClusterNameForServiceCluster(clusterName)
		}
		wc.Clusters = append(wc.Clusters, &xds_route.WeightedCluster_ClusterWeight{
			Name:   clusterName,
			Weight: &wrappers.UInt32Value{Value: uint32(cluster.Weight)},
		})
	}
	if direction == OutboundRoute {
		total = totalWeight
	}
	wc.TotalWeight = &wrappers.UInt32Value{Value: uint32(total)}
	sort.Stable(clusterWeightByName(wc.Clusters))
	return &wc
}
