package rds

import (
	"fmt"
	"sort"
	"time"

	mapset "github.com/deckarep/golang-set"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_http_local_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	xds_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

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

	// methodHeaderKey is the key of the header for HTTP methods
	methodHeaderKey = ":method"

	// httpHostHeaderKey is the name of the HTTP host header in HTTPRouteMatch.Headers
	httpHostHeaderKey = "host"

	// authorityHeaderKey is the key corresponding to the HTTP Host/Authority header programmed as a header matcher in an Envoy route
	authorityHeaderKey = ":authority"

	httpLocalRateLimiterStatsPrefix = "http_local_rate_limiter"
)

// applyInboundVirtualHostConfig updates the VirtualHost configuration based on the given policy
func applyInboundVirtualHostConfig(vhost *xds_route.VirtualHost, policy *trafficpolicy.InboundTrafficPolicy) {
	if vhost == nil || policy == nil {
		return
	}

	config := make(map[string]*any.Any)

	// Apply VirtualHost level rate limiting config
	if policy.RateLimit != nil && policy.RateLimit.Local != nil && policy.RateLimit.Local.HTTP != nil {
		if filter, err := getLocalRateLimitFilterConfig(policy.RateLimit.Local.HTTP); err != nil {
			log.Error().Err(err).Msgf("Error applying local rate limiting config for vhost %s, ignoring it", vhost.Name)
		} else {
			config[envoy.HTTPLocalRateLimitFilterName] = filter
		}
	}

	if policy.RateLimit != nil && policy.RateLimit.Global != nil && policy.RateLimit.Global.HTTP != nil {
		vhost.RateLimits = getGlobalRateLimitConfig(policy.RateLimit.Global.HTTP.Descriptors)
	}

	vhost.TypedPerFilterConfig = config
}

// getLocalRateLimitFilterConfig returns the marshalled HTTP local rate limiting config for the given policy
func getLocalRateLimitFilterConfig(config *policyv1alpha1.HTTPLocalRateLimitSpec) (*any.Any, error) {
	if config == nil {
		return nil, nil
	}

	var fillInterval time.Duration
	switch config.Unit {
	case "second":
		fillInterval = time.Second
	case "minute":
		fillInterval = time.Minute
	case "hour":
		fillInterval = time.Hour
	default:
		return nil, fmt.Errorf("invalid unit %q for HTTP request rate limiting", config.Unit)
	}

	rl := &xds_http_local_ratelimit.LocalRateLimit{
		StatPrefix: httpLocalRateLimiterStatsPrefix,
		TokenBucket: &xds_type.TokenBucket{
			MaxTokens:     config.Requests + config.Burst,
			TokensPerFill: wrapperspb.UInt32(config.Requests),
			FillInterval:  durationpb.New(fillInterval),
		},
		ResponseHeadersToAdd: getRateLimitHeaderValueOptions(config.ResponseHeadersToAdd),
		FilterEnabled: &xds_core.RuntimeFractionalPercent{
			DefaultValue: &xds_type.FractionalPercent{
				Numerator:   100,
				Denominator: xds_type.FractionalPercent_HUNDRED,
			},
		},
		FilterEnforced: &xds_core.RuntimeFractionalPercent{
			DefaultValue: &xds_type.FractionalPercent{
				Numerator:   100,
				Denominator: xds_type.FractionalPercent_HUNDRED,
			},
		},
	}

	// Set the response status code if not specified. Envoy defaults to 429 (Too Many Requests).
	if config.ResponseStatusCode > 0 {
		rl.Status = &xds_type.HttpStatus{Code: xds_type.StatusCode(config.ResponseStatusCode)}
	}

	marshalled, err := anypb.New(rl)
	if err != nil {
		return nil, err
	}

	return marshalled, nil
}

func getGlobalRateLimitConfig(descriptors []policyv1alpha1.HTTPGlobalRateLimitDescriptor) []*xds_route.RateLimit {
	var rateLimits []*xds_route.RateLimit
	for _, descriptor := range descriptors {
		rl := &xds_route.RateLimit{}

		for _, entry := range descriptor.Entries {
			switch {
			case entry.GenericKey != nil:
				rl.Actions = append(rl.Actions, &xds_route.RateLimit_Action{
					ActionSpecifier: &xds_route.RateLimit_Action_GenericKey_{
						GenericKey: &xds_route.RateLimit_Action_GenericKey{
							DescriptorKey:   entry.GenericKey.Key,
							DescriptorValue: entry.GenericKey.Value,
						},
					},
				})

			case entry.RemoteAddress != nil:
				rl.Actions = append(rl.Actions, &xds_route.RateLimit_Action{
					ActionSpecifier: &xds_route.RateLimit_Action_RemoteAddress_{
						RemoteAddress: &xds_route.RateLimit_Action_RemoteAddress{},
					},
				})

			case entry.RequestHeader != nil:
				rl.Actions = append(rl.Actions, &xds_route.RateLimit_Action{
					ActionSpecifier: &xds_route.RateLimit_Action_RequestHeaders_{
						RequestHeaders: &xds_route.RateLimit_Action_RequestHeaders{
							HeaderName:    entry.RequestHeader.Name,
							DescriptorKey: entry.RequestHeader.Key,
						},
					},
				})

			case entry.HeaderValueMatch != nil:
				rl.Actions = append(rl.Actions, &xds_route.RateLimit_Action{
					ActionSpecifier: &xds_route.RateLimit_Action_HeaderValueMatch_{
						HeaderValueMatch: &xds_route.RateLimit_Action_HeaderValueMatch{
							DescriptorKey:   entry.HeaderValueMatch.Key,
							DescriptorValue: entry.HeaderValueMatch.Value,
							Headers:         getHeaderMatchers(entry.HeaderValueMatch.Headers),
							ExpectMatch: func() *wrappers.BoolValue {
								if entry.HeaderValueMatch.ExpectMatch != nil {
									return wrapperspb.Bool(*entry.HeaderValueMatch.ExpectMatch)
								}
								return nil
							}(),
						},
					},
				})
			}
		}

		rateLimits = append(rateLimits, rl)
	}

	return rateLimits
}

func getHeaderMatchers(headers []policyv1alpha1.HTTPHeaderMatcher) []*xds_route.HeaderMatcher {
	var headerMatchers []*xds_route.HeaderMatcher

	for _, h := range headers {
		hm := &xds_route.HeaderMatcher{
			Name: h.Name,
		}

		switch {
		case h.Exact != "":
			hm.HeaderMatchSpecifier = &xds_route.HeaderMatcher_StringMatch{
				StringMatch: &xds_matcher.StringMatcher{
					MatchPattern: &xds_matcher.StringMatcher_Exact{Exact: h.Exact},
				},
			}

		case h.Prefix != "":
			hm.HeaderMatchSpecifier = &xds_route.HeaderMatcher_StringMatch{
				StringMatch: &xds_matcher.StringMatcher{
					MatchPattern: &xds_matcher.StringMatcher_Prefix{Prefix: h.Prefix},
				},
			}

		case h.Suffix != "":
			hm.HeaderMatchSpecifier = &xds_route.HeaderMatcher_StringMatch{
				StringMatch: &xds_matcher.StringMatcher{
					MatchPattern: &xds_matcher.StringMatcher_Suffix{Suffix: h.Suffix},
				},
			}

		case h.Regex != "":
			hm.HeaderMatchSpecifier = &xds_route.HeaderMatcher_StringMatch{
				StringMatch: &xds_matcher.StringMatcher{
					MatchPattern: &xds_matcher.StringMatcher_SafeRegex{
						SafeRegex: &xds_matcher.RegexMatcher{
							EngineType: &xds_matcher.RegexMatcher_GoogleRe2{GoogleRe2: &xds_matcher.RegexMatcher_GoogleRE2{}},
							Regex:      h.Regex,
						}},
				},
			}

		case h.Contains != "":
			hm.HeaderMatchSpecifier = &xds_route.HeaderMatcher_StringMatch{
				StringMatch: &xds_matcher.StringMatcher{
					MatchPattern: &xds_matcher.StringMatcher_Contains{Contains: h.Contains},
				},
			}

		case h.Present != nil:
			hm.HeaderMatchSpecifier = &xds_route.HeaderMatcher_PresentMatch{
				PresentMatch: *h.Present,
			}
		}

		headerMatchers = append(headerMatchers, hm)
	}

	return headerMatchers
}

// getRateLimitHeaderValueOptions returns a list of HeaderValueOption objects corresponding
// to the given list of rate limiting HTTPHeaderValue objects
func getRateLimitHeaderValueOptions(headerValues []policyv1alpha1.HTTPHeaderValue) []*xds_core.HeaderValueOption {
	var hvOptions []*xds_core.HeaderValueOption

	for _, hv := range headerValues {
		hvOptions = append(hvOptions, &xds_core.HeaderValueOption{
			Header: &xds_core.HeaderValue{
				Key:   hv.Name,
				Value: hv.Value,
			},
			Append: &wrappers.BoolValue{
				Value: false,
			},
		})
	}

	return hvOptions
}

// newRouteConfigurationStub creates the route configuration placeholder
func newRouteConfigurationStub(routeConfigName string) *xds_route.RouteConfiguration {
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

// buildVirtualHostStub creates the virtual host placeholder
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
		rbacConfig, err := buildInboundRBACFilterForRule(rule)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrBuildingRBACPolicyForRoute)).
				Msgf("Error building RBAC policy for rule [%v], skipping route addition", rule)
			continue
		}

		// Each HTTP method corresponds to a separate route
		for _, method := range allowedMethods {
			route := buildRoute(rule.Route, method)
			applyInboundRouteConfig(route, rbacConfig, rule.Route.RateLimit)
			routes = append(routes, route)
		}
	}
	return routes
}

func applyInboundRouteConfig(route *xds_route.Route, rbacConfig *any.Any, rateLimit *policyv1alpha1.HTTPPerRouteRateLimitSpec) {
	if route == nil {
		return
	}

	perFilterConfig := make(map[string]*any.Any)

	// Apply RBACPerRoute policy
	perFilterConfig[envoy.HTTPRBACFilterName] = rbacConfig

	// Apply local rate limit policy
	if rateLimit != nil && rateLimit.Local != nil {
		if filter, err := getLocalRateLimitFilterConfig(rateLimit.Local); err != nil {
			log.Error().Err(err).Msgf("Error applying local rate limiting config for route path %s, ignoring it", route.GetMatch().GetPath())
		} else {
			perFilterConfig[envoy.HTTPLocalRateLimitFilterName] = filter
		}
	}

	route.TypedPerFilterConfig = perFilterConfig
}

// buildOutboundRoutes takes route information from the given outbound traffic policy and returns a list of xds routes
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

// buildEgressRoutes takes route information from the given egress traffic policy and returns a list of xds routes
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
	getPerRouteRateLimitDescriptors := func(rl *policyv1alpha1.HTTPPerRouteRateLimitSpec) []policyv1alpha1.HTTPGlobalRateLimitDescriptor {
		if rl != nil && rl.Global != nil {
			return rl.Global.Descriptors
		}
		return nil
	}

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
				RateLimits:  getGlobalRateLimitConfig(getPerRouteRateLimitDescriptors(weightedClusters.RateLimit)),
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
// Remove checks when validation webhook is implemented
func buildRetryPolicy(retry *policyv1alpha1.RetryPolicySpec) *xds_route.RetryPolicy {
	if retry == nil {
		return nil
	}

	rp := &xds_route.RetryPolicy{}

	rp.RetryOn = retry.RetryOn
	// NumRetries default is set to 1
	if retry.NumRetries != nil {
		rp.NumRetries = wrapperspb.UInt32(*retry.NumRetries)
	}

	// PerTryTimeout default uses the global route timeout
	// Disabling route config timeout does not affect perTryTimeout
	if retry.PerTryTimeout != nil {
		rp.PerTryTimeout = durationpb.New(retry.PerTryTimeout.Duration)
	}

	// RetryBackOff default base interval is 25 ms
	if retry.RetryBackoffBaseInterval != nil {
		rp.RetryBackOff = &xds_route.RetryPolicy_RetryBackOff{
			BaseInterval: durationpb.New(retry.RetryBackoffBaseInterval.Duration),
		}
	}

	return rp
}

// sanitizeHTTPMethods takes in a list of HTTP methods including a wildcard (*) and returns a wildcard if any of
// the methods is a wildcard or sanitizes the input list to avoid duplicates.
func sanitizeHTTPMethods(allowedMethods []string) []string {
	var newAllowedMethods []string
	keys := make(map[string]struct{})
	for _, method := range allowedMethods {
		if method != "" {
			if method == constants.WildcardHTTPMethod {
				return []string{constants.WildcardHTTPMethod}
			}
			if _, exists := keys[method]; !exists {
				keys[method] = struct{}{}
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
