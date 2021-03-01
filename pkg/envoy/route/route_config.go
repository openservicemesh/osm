package route

import (
	"fmt"
	"sort"

	set "github.com/deckarep/golang-set"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/featureflags"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// Direction is a type to signify the direction associated with a route
type Direction int

const (
	// OutboundRoute is the direction for an outbound route
	OutboundRoute Direction = iota

	// InboundRoute is the direction for an inbound route
	InboundRoute
)

const (
	//InboundRouteConfigName is the name of the route config that the envoy will identify
	InboundRouteConfigName = "RDS_Inbound"

	//OutboundRouteConfigName is the name of the route config that the envoy will identify
	OutboundRouteConfigName = "RDS_Outbound"

	// inboundVirtualHost is the name of the virtual host on the inbound route configuration
	inboundVirtualHost = "inbound_virtual-host"

	// outboundVirtualHost is the name of the virtual host on the outbound route configuration
	outboundVirtualHost = "outbound_virtual-host"

	// MethodHeaderKey is the key of the header for HTTP methods
	MethodHeaderKey = ":method"

	// httpHostHeader is the name of the HTTP host header
	httpHostHeader = "host"
)

// BuildRouteConfiguration constructs the Envoy constructs ([]*xds_route.RouteConfiguration) for implementing inbound and outbound routes
func BuildRouteConfiguration(inbound []*trafficpolicy.InboundTrafficPolicy, outbound []*trafficpolicy.OutboundTrafficPolicy, proxy *envoy.Proxy) []*xds_route.RouteConfiguration {
	routeConfiguration := []*xds_route.RouteConfiguration{}

	if len(inbound) > 0 {
		inboundRouteConfig := NewRouteConfigurationStub(InboundRouteConfigName)
		for _, in := range inbound {
			virtualHost := buildVirtualHostStub(inboundVirtualHost, in.Name, in.Hostnames)
			virtualHost.Routes = buildInboundRoutes(in.Rules)
			inboundRouteConfig.VirtualHosts = append(inboundRouteConfig.VirtualHosts, virtualHost)
		}

		if featureflags.IsWASMStatsEnabled() {
			for k, v := range proxy.StatsHeaders() {
				inboundRouteConfig.ResponseHeadersToAdd = append(inboundRouteConfig.ResponseHeadersToAdd, &core.HeaderValueOption{
					Header: &core.HeaderValue{
						Key:   k,
						Value: v,
					},
				})
			}
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

//NewRouteConfigurationStub creates the route configuration placeholder
func NewRouteConfigurationStub(routeConfigName string) *xds_route.RouteConfiguration {
	routeConfiguration := xds_route.RouteConfiguration{
		Name: routeConfigName,
		// ValidateClusters `true` causes RDS rejections if the CDS is not "warm" with the expected
		// clusters RDS wants to use. This can happen when CDS and RDS updates are sent closely
		// together. Setting it to false bypasses this check, and just assumes the cluster will
		// be present when it needs to be checked by traffic (or 404 otherwise).
		ValidateClusters: &wrappers.BoolValue{Value: false},
	}
	return &routeConfiguration
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
func buildInboundRoutes(rules []*trafficpolicy.Rule) []*xds_route.Route {
	var routes []*xds_route.Route
	for _, rule := range rules {
		// For a given route path, sanitize the methods in case there
		// is wildcard or if there are duplicates
		allowedMethods := sanitizeHTTPMethods(rule.Route.HTTPRouteMatch.Methods)

		// Create an RBAC policy derived from 'trafficpolicy.Rule'
		// Each route is associated with an RBAC policy
		rbacPolicyForRoute, err := buildInboundRBACFilterForRule(rule)
		if err != nil {
			log.Error().Err(err).Msgf("Error building RBAC policy for rule [%v], skipping route addition", rule)
			continue
		}

		// Each HTTP method corresponds to a separate route
		for _, method := range allowedMethods {
			route := buildRoute(rule.Route.HTTPRouteMatch.PathRegex, method, rule.Route.HTTPRouteMatch.Headers, rule.Route.WeightedClusters, 100, InboundRoute)
			route.TypedPerFilterConfig = rbacPolicyForRoute
			routes = append(routes, route)
		}
	}
	return routes
}

func buildOutboundRoutes(outRoutes []*trafficpolicy.RouteWeightedClusters) []*xds_route.Route {
	var routes []*xds_route.Route
	for _, outRoute := range outRoutes {
		emptyHeaders := map[string]string{}
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

// sanitizeHTTPMethods takes in a list of HTTP methods including a wildcard (*) and returns a wildcard if any of
// the methods is a wildcard or sanitizes the input list to avoid duplicates.
func sanitizeHTTPMethods(allowedMethods []string) []string {
	var newAllowedMethods []string
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

type clusterWeightByName []*xds_route.WeightedCluster_ClusterWeight

func (c clusterWeightByName) Len() int      { return len(c) }
func (c clusterWeightByName) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c clusterWeightByName) Less(i, j int) bool {
	if c[i].Name == c[j].Name {
		return c[i].Weight.Value < c[j].Weight.Value
	}
	return c[i].Name < c[j].Name
}

func getHeadersForRoute(method string, headersMap map[string]string) []*xds_route.HeaderMatcher {
	var headers []*xds_route.HeaderMatcher

	// add methods header
	methodsHeader := &xds_route.HeaderMatcher{
		Name: MethodHeaderKey,
		HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
			SafeRegexMatch: &xds_matcher.RegexMatcher{
				EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
				Regex:      getRegexForMethod(method),
			},
		},
	}
	headers = append(headers, methodsHeader)

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
					EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
					Regex:      headerValue,
				},
			},
		}
		headers = append(headers, &header)
	}
	return headers
}

func getRegexForMethod(httpMethod string) string {
	methodRegex := httpMethod
	if httpMethod == constants.WildcardHTTPMethod {
		methodRegex = constants.RegexMatchAll
	}
	return methodRegex
}
