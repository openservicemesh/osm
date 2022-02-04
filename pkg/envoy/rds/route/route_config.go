package route

import (
	"fmt"
	"sort"
	"time"

	mapset "github.com/deckarep/golang-set"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// InboundRouteConfigName is the name of the inbound mesh RDS route configuration
	InboundRouteConfigName = "rds-inbound"

	// OutboundRouteConfigName is the name of the outbound mesh RDS route configuration
	OutboundRouteConfigName = "rds-outbound"

	// IngressRouteConfigName is the name of the ingress RDS route configuration
	IngressRouteConfigName = "rds-ingress"

	// egressRouteConfigNamePrefix is the prefix for the name of the egress RDS route configuration
	egressRouteConfigNamePrefix = "rds-egress"

	// inboundVirtualHost is prefix for the virtual host's name in the inbound route configuration
	inboundVirtualHost = "inbound_virtual-host"

	// outboundVirtualHost is the prefix for the virtual host's name in the outbound route configuration
	outboundVirtualHost = "outbound_virtual-host"

	// egressVirtualHost is the prefix for the virtual host's name in the egress route configuration
	egressVirtualHost = "egress_virtual-host"

	// ingressVirtualHost is the prefix for the virtual host's name in the ingress route configuration
	ingressVirtualHost = "ingress_virtual-host"

	// methodHeaderKey is the key of the header for HTTP methods
	methodHeaderKey = ":method"

	// httpHostHeaderKey is the name of the HTTP host header in HTTPRouteMatch.Headers
	httpHostHeaderKey = "host"

	// authorityHeaderKey is the key corresponding to the HTTP Host/Authority header programmed as a header matcher in an Envoy route
	authorityHeaderKey = ":authority"
)

// BuildInboundMeshRouteConfiguration constructs the Envoy constructs ([]*xds_route.RouteConfiguration) for implementing inbound and outbound routes
func BuildInboundMeshRouteConfiguration(portSpecificRouteConfigs map[int][]*trafficpolicy.InboundTrafficPolicy, proxy *envoy.Proxy, cfg configurator.Configurator) []*xds_route.RouteConfiguration {
	var routeConfigs []*xds_route.RouteConfiguration

	// An Envoy RouteConfiguration will exist for each HTTP upstream port.
	// This is required to avoid route conflicts that can arise when the same host header
	// has different routes on different destination ports for that host.
	for port, configs := range portSpecificRouteConfigs {
		routeConfig := NewRouteConfigurationStub(GetInboundMeshRouteConfigNameForPort(port))
		for _, config := range configs {
			virtualHost := buildVirtualHostStub(inboundVirtualHost, config.Name, config.Hostnames)
			virtualHost.Routes = buildInboundRoutes(config.Rules)
			routeConfig.VirtualHosts = append(routeConfig.VirtualHosts, virtualHost)
		}
		if featureFlags := cfg.GetFeatureFlags(); featureFlags.EnableWASMStats {
			for k, v := range proxy.StatsHeaders() {
				routeConfig.ResponseHeadersToAdd = append(routeConfig.ResponseHeadersToAdd, &core.HeaderValueOption{
					Header: &core.HeaderValue{
						Key:   k,
						Value: v,
					},
				})
			}
		}
		routeConfigs = append(routeConfigs, routeConfig)
	}

	return routeConfigs
}

// BuildIngressConfiguration constructs the Envoy constructs ([]*xds_route.RouteConfiguration) for implementing ingress routes
func BuildIngressConfiguration(ingress []*trafficpolicy.InboundTrafficPolicy) *xds_route.RouteConfiguration {
	if len(ingress) == 0 {
		return nil
	}

	ingressRouteConfig := NewRouteConfigurationStub(IngressRouteConfigName)
	for _, in := range ingress {
		virtualHost := buildVirtualHostStub(ingressVirtualHost, in.Name, in.Hostnames)
		virtualHost.Routes = buildInboundRoutes(in.Rules)
		ingressRouteConfig.VirtualHosts = append(ingressRouteConfig.VirtualHosts, virtualHost)
	}

	return ingressRouteConfig
}

// BuildOutboundMeshRouteConfiguration constructs the Envoy construct (*xds_route.RouteConfiguration) for the given outbound mesh route configs
func BuildOutboundMeshRouteConfiguration(portSpecificRouteConfigs map[int][]*trafficpolicy.OutboundTrafficPolicy) []*xds_route.RouteConfiguration {
	var routeConfigs []*xds_route.RouteConfiguration

	// An Envoy RouteConfiguration will exist for each HTTP upstream port.
	// This is required to avoid route conflicts that can arise when the same host header
	// has different routes on different destination ports for that host.
	for port, configs := range portSpecificRouteConfigs {
		routeConfig := NewRouteConfigurationStub(GetOutboundMeshRouteConfigNameForPort(port))
		for _, config := range configs {
			virtualHost := buildVirtualHostStub(outboundVirtualHost, config.Name, config.Hostnames)
			virtualHost.Routes = buildOutboundRoutes(config.Routes)
			routeConfig.VirtualHosts = append(routeConfig.VirtualHosts, virtualHost)
		}
		routeConfigs = append(routeConfigs, routeConfig)
	}

	return routeConfigs
}

// BuildEgressRouteConfiguration constructs the Envoy construct (*xds_route.RouteConfiguration) for the given egress route configs
func BuildEgressRouteConfiguration(portSpecificRouteConfigs map[int][]*trafficpolicy.EgressHTTPRouteConfig) []*xds_route.RouteConfiguration {
	var routeConfigs []*xds_route.RouteConfiguration

	// An Envoy RouteConfiguration will exist for each HTTP egress port.
	// This is required to avoid route conflicts that can arise when the same host header
	// has different routes on different destination ports for that host.
	for port, configs := range portSpecificRouteConfigs {
		routeConfig := NewRouteConfigurationStub(GetEgressRouteConfigNameForPort(port))
		for _, config := range configs {
			virtualHost := buildVirtualHostStub(egressVirtualHost, config.Name, config.Hostnames)
			virtualHost.Routes = buildEgressRoutes(config.RoutingRules)
			routeConfig.VirtualHosts = append(routeConfig.VirtualHosts, virtualHost)
		}
		routeConfigs = append(routeConfigs, routeConfig)
	}

	return routeConfigs
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
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrBuildingRBACPolicyForRoute)).
				Msgf("Error building RBAC policy for rule [%v], skipping route addition", rule)
			continue
		}

		// Each HTTP method corresponds to a separate route
		for _, method := range allowedMethods {
			route := buildRoute(rule.Route, method)
			route.TypedPerFilterConfig = rbacPolicyForRoute
			routes = append(routes, route)
		}
	}
	return routes
}

func buildOutboundRoutes(outRoutes []*trafficpolicy.RouteWeightedClusters) []*xds_route.Route {
	var routes []*xds_route.Route
	for _, outRoute := range outRoutes {
		// Create temp variable to avoid potentially overwriting the loop variable
		tempOutbound := *outRoute
		tempOutbound.HTTPRouteMatch.PathMatchType = trafficpolicy.PathMatchRegex
		tempOutbound.HTTPRouteMatch.Path = constants.RegexMatchAll
		tempOutbound.HTTPRouteMatch.Headers = map[string]string{}
		routes = append(routes, buildRoute(tempOutbound, constants.WildcardHTTPMethod))
	}

	return routes
}

func buildEgressRoutes(routingRules []*trafficpolicy.EgressHTTPRoutingRule) []*xds_route.Route {
	var routes []*xds_route.Route
	for _, rule := range routingRules {
		// For a given route path, sanitize the methods in case there
		// is wildcard or if there are duplicates
		allowedHTTPMethods := sanitizeHTTPMethods(rule.Route.HTTPRouteMatch.Methods)

		// Build the route for the given egress routing rule and method
		// Each HTTP method corresponds to a separate route
		for _, httpMethod := range allowedHTTPMethods {
			route := buildRoute(rule.Route, httpMethod)
			routes = append(routes, route)
		}
	}
	return routes
}

func buildRoute(weightedClusters trafficpolicy.RouteWeightedClusters, method string) *xds_route.Route {
	route := xds_route.Route{
		Match: &xds_route.RouteMatch{
			Headers: getHeadersForRoute(method, weightedClusters.HTTPRouteMatch.Headers),
		},
		Action: &xds_route.Route_Route{
			Route: &xds_route.RouteAction{
				ClusterSpecifier: &xds_route.RouteAction_WeightedClusters{
					WeightedClusters: buildWeightedCluster(weightedClusters.WeightedClusters),
				},
				// Disable default 15s timeout. This otherwise results in requests that take
				// longer than 15s to timeout, e.g. large file transfers.
				Timeout:     &duration.Duration{Seconds: 0},
				RetryPolicy: buildRetryPolicy(weightedClusters.RetryPolicy),
			},
		},
	}

	switch weightedClusters.HTTPRouteMatch.PathMatchType {
	case trafficpolicy.PathMatchRegex:
		route.Match.PathSpecifier = &xds_route.RouteMatch_SafeRegex{
			SafeRegex: &xds_matcher.RegexMatcher{
				EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
				Regex:      weightedClusters.HTTPRouteMatch.Path,
			},
		}

	case trafficpolicy.PathMatchExact:
		route.Match.PathSpecifier = &xds_route.RouteMatch_Path{
			Path: weightedClusters.HTTPRouteMatch.Path,
		}

	case trafficpolicy.PathMatchPrefix:
		route.Match.PathSpecifier = &xds_route.RouteMatch_Prefix{
			Prefix: weightedClusters.HTTPRouteMatch.Path,
		}
	}

	return &route
}

func buildWeightedCluster(weightedClusters mapset.Set) *xds_route.WeightedCluster {
	var wc xds_route.WeightedCluster
	var total int
	for clusterInterface := range weightedClusters.Iter() {
		cluster := clusterInterface.(service.WeightedCluster)
		total += cluster.Weight
		wc.Clusters = append(wc.Clusters, &xds_route.WeightedCluster_ClusterWeight{
			Name:   cluster.ClusterName.String(),
			Weight: &wrappers.UInt32Value{Value: uint32(cluster.Weight)},
		})
	}

	if total < 1 {
		// ref: https://github.com/envoyproxy/go-control-plane/blob/31f9241a16e627ba7696bed59a6353c95412ddb5/envoy/config/route/v3/route_components.pb.validate.go#L772
		log.Error().Msgf("Total weight of weighted cluster must be >= 1, got %d", total)
		return nil
	}
	wc.TotalWeight = &wrappers.UInt32Value{Value: uint32(total)}
	sort.Stable(clusterWeightByName(wc.Clusters))
	return &wc
}

// TODO: Add validation webhook for retry policy
func buildRetryPolicy(retry *v1alpha1.RetryPolicySpec) *xds_route.RetryPolicy {
	if retry == nil {
		return nil
	}

	rp := &xds_route.RetryPolicy{}

	rp.RetryOn = retry.RetryOn
	rp.NumRetries = &wrapperspb.UInt32Value{
		Value: uint32(retry.NumRetries),
	}
	rp.PerTryTimeout = timeToDuration(retry.PerTryTimeout)
	rp.RetryBackOff = &xds_route.RetryPolicy_RetryBackOff{
		BaseInterval: timeToDuration(retry.RetryBackoffBaseInterval),
	}

	return rp
}

func timeToDuration(timeStr string) *durationpb.Duration {
	if timeStr == "" {
		return nil
	}

	duration, err := time.ParseDuration(timeStr)
	if err != nil {
		log.Error().Msgf("Error parsing time: %s", timeStr)
		return nil
	}
	return durationpb.New(duration)
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
		Name: methodHeaderKey,
		HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
			SafeRegexMatch: &xds_matcher.RegexMatcher{
				EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
				Regex:      getRegexForMethod(method),
			},
		},
	}
	headers = append(headers, methodsHeader)

	// add host headers
	if hostHeaderValue, ok := headersMap[httpHostHeaderKey]; ok {
		hostHeader := &xds_route.HeaderMatcher{
			Name: authorityHeaderKey,
			HeaderMatchSpecifier: &xds_route.HeaderMatcher_SafeRegexMatch{
				SafeRegexMatch: &xds_matcher.RegexMatcher{
					EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
					Regex:      hostHeaderValue,
				},
			},
		}
		headers = append(headers, hostHeader)
	}

	// add all other custom headers
	for headerKey, headerValue := range headersMap {
		// omit the host header as this is configured above
		if headerKey == httpHostHeaderKey {
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

// GetEgressRouteConfigNameForPort returns the Egress route configuration object's name given the port it is targeted to
func GetEgressRouteConfigNameForPort(port int) string {
	return fmt.Sprintf("%s.%d", egressRouteConfigNamePrefix, port)
}

// GetOutboundMeshRouteConfigNameForPort returns the outbound mesh route configuration object's name given the port it is targeted to
func GetOutboundMeshRouteConfigNameForPort(port int) string {
	return fmt.Sprintf("%s.%d", OutboundRouteConfigName, port)
}

// GetInboundMeshRouteConfigNameForPort returns the inbound mesh route configuration object's name given the port it is targeted to
func GetInboundMeshRouteConfigNameForPort(port int) string {
	return fmt.Sprintf("%s.%d", InboundRouteConfigName, port)
}
