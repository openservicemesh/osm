package lds

import (
	"fmt"
	"strings"
	"time"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_config_ratelimit "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	xds_common_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	xds_network_local_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/local_ratelimit/v3"
	xds_global_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/ratelimit/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func (lb *listenerBuilder) buildInboundMeshFilterChains() []*xds_listener.FilterChain {
	if lb.inboundMeshTrafficPolicy == nil {
		return nil
	}

	var filterChains []*xds_listener.FilterChain

	for _, match := range lb.inboundMeshTrafficPolicy.TrafficMatches {
		// Create protocol specific inbound filter chains for MeshService's TargetPort
		switch strings.ToLower(match.DestinationProtocol) {
		case constants.ProtocolHTTP, constants.ProtocolGRPC:
			// Filter chain for HTTP port
			filterChainForPort, err := lb.buildInboundHTTPFilterChain(match)
			if err != nil {
				log.Error().Err(err).Msgf("Error building inbound HTTP filter chain for traffic match %s", match.Name)
			} else {
				filterChains = append(filterChains, filterChainForPort)
			}

		case constants.ProtocolTCP, constants.ProtocolTCPServerFirst:
			filterChainForPort, err := lb.buildInboundTCPFilterChain(match)
			if err != nil {
				log.Error().Err(err).Msgf("Error building inbound TCP filter chain for traffic match %s", match.Name)
			} else {
				filterChains = append(filterChains, filterChainForPort)
			}

		default:
			log.Error().Msgf("Cannot build inbound filter chain, unsupported protocol %s for traffic match %s", match.DestinationProtocol, match.Name)
		}
	}

	return filterChains
}

func (lb *listenerBuilder) buildInboundHTTPFilterChain(trafficMatch *trafficpolicy.TrafficMatch) (*xds_listener.FilterChain, error) {
	if trafficMatch == nil {
		return nil, nil
	}

	// Build filters
	fb := lb.getFilterBuilder().
		StatsPrefix(trafficMatch.Name)

	// Network RBAC
	if !lb.permissiveMesh {
		fb.WithRBAC(lb.trafficTargets, lb.trustDomain)
	}

	// TCP local rate limit
	if trafficMatch.RateLimit != nil && trafficMatch.RateLimit.Local != nil && trafficMatch.RateLimit.Local.TCP != nil {
		fb.TCPLocalRateLimit(trafficMatch.RateLimit.Local.TCP)
	}

	// TCP global rate limit
	if trafficMatch.RateLimit != nil && trafficMatch.RateLimit.Global != nil && trafficMatch.RateLimit.Global.TCP != nil {
		fb.TCPGlobalRateLimit(trafficMatch.RateLimit.Global.TCP)
	}

	fb.httpConnManager().StatsPrefix(route.GetInboundMeshRouteConfigNameForPort(trafficMatch.DestinationPort)).
		RouteConfigName(route.GetInboundMeshRouteConfigNameForPort(trafficMatch.DestinationPort))

	if lb.httpTracingEndpoint != "" {
		tracing, err := getHTTPTracingConfig(lb.httpTracingEndpoint)
		if err != nil {
			return nil, fmt.Errorf("error building inbound http filter chain: %w", err)
		}
		fb.httpConnManager().Tracing(tracing)
	}
	if lb.extAuthzConfig != nil && lb.extAuthzConfig.Enable {
		fb.httpConnManager().AddFilter(getExtAuthzHTTPFilter(lb.extAuthzConfig))
	}
	if lb.wasmStatsHeaders != nil {
		wasmFilters, wasmLocalReplyConfig, err := getWASMStatsConfig(lb.wasmStatsHeaders)
		if err != nil {
			return nil, fmt.Errorf("error building inbound http filter chain: %w", err)
		}
		fb.httpConnManager().LocalReplyConfig(wasmLocalReplyConfig)
		for _, f := range wasmFilters {
			fb.httpConnManager().AddFilter(f)
		}
	}
	if lb.activeHealthCheck {
		healthCheckFilter, err := getHealthCheckFilter()
		if err != nil {
			return nil, fmt.Errorf("error building inbound http filter chain: %w", err)
		}
		fb.httpConnManager().AddFilter(healthCheckFilter)
	}

	// Build the inbound filters
	filters, err := fb.Build()
	if err != nil {
		return nil, fmt.Errorf("error building inbound HTTP filter chain: %w", err)
	}

	// Construct downstream TLS context
	marshalledDownstreamTLSContext, err := anypb.New(envoy.GetDownstreamTLSContext(lb.proxyIdentity, true /* mTLS */, lb.sidecarSpec))
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling DownstreamTLSContext for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	filterChain := &xds_listener.FilterChain{
		Name:    trafficMatch.Name,
		Filters: filters,

		// The 'FilterChainMatch' field defines the criteria for matching traffic against filters in this filter chain
		FilterChainMatch: &xds_listener.FilterChainMatch{
			// The DestinationPort is the service port the downstream directs traffic to
			DestinationPort: &wrapperspb.UInt32Value{
				Value: uint32(trafficMatch.DestinationPort),
			},

			// The ServerName is the SNI set by the downstream in the UptreamTlsContext by GetUpstreamTLSContext()
			// This is not a field obtained from the mTLS Certificate.
			ServerNames: trafficMatch.ServerNames,

			// Only match when transport protocol is TLS
			TransportProtocol: envoy.TransportProtocolTLS,

			// In-mesh proxies will advertise this, set in the UpstreamTlsContext by GetUpstreamTLSContext()
			ApplicationProtocols: envoy.ALPNInMesh,
		},

		TransportSocket: &xds_core.TransportSocket{
			Name: trafficMatch.Name,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		},
	}

	return filterChain, nil
}

func (lb *listenerBuilder) buildInboundTCPFilterChain(trafficMatch *trafficpolicy.TrafficMatch) (*xds_listener.FilterChain, error) {
	if trafficMatch == nil {
		return nil, nil
	}

	// Build filters
	fb := getFilterBuilder().
		StatsPrefix(trafficMatch.Name)

	fb.TCPProxy().
		StatsPrefix(trafficMatch.Name).
		Cluster(trafficMatch.Cluster)

	// Network RBAC
	if !lb.permissiveMesh && len(lb.trafficTargets) > 0 {
		fb.WithRBAC(lb.trafficTargets, lb.trustDomain)
	}

	// TCP local rate limit
	if trafficMatch.RateLimit != nil && trafficMatch.RateLimit.Local != nil && trafficMatch.RateLimit.Local.TCP != nil {
		fb.TCPLocalRateLimit(trafficMatch.RateLimit.Local.TCP)
	}

	// TCP global rate limit
	if trafficMatch.RateLimit != nil && trafficMatch.RateLimit.Global != nil && trafficMatch.RateLimit.Global.TCP != nil {
		fb.TCPGlobalRateLimit(trafficMatch.RateLimit.Global.TCP)
	}

	// Build the inbound filters
	filters, err := fb.Build()
	if err != nil {
		return nil, fmt.Errorf("error building inbound TCP filters: %w", err)
	}

	// Construct downstream TLS context
	marshalledDownstreamTLSContext, err := anypb.New(envoy.GetDownstreamTLSContext(lb.proxyIdentity, true /* mTLS */, lb.sidecarSpec))
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling DownstreamTLSContext for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	return &xds_listener.FilterChain{
		Name: trafficMatch.Name,
		FilterChainMatch: &xds_listener.FilterChainMatch{
			// The DestinationPort is the service port the downstream directs traffic to
			DestinationPort: &wrapperspb.UInt32Value{
				Value: uint32(trafficMatch.DestinationPort),
			},

			// The ServerName is the SNI set by the downstream in the UptreamTlsContext by GetUpstreamTLSContext()
			// This is not a field obtained from the mTLS Certificate.
			ServerNames: trafficMatch.ServerNames,

			// Only match when transport protocol is TLS
			TransportProtocol: envoy.TransportProtocolTLS,

			// In-mesh proxies will advertise this, set in the UpstreamTlsContext by GetUpstreamTLSContext()
			ApplicationProtocols: envoy.ALPNInMesh,
		},
		Filters: filters,
		TransportSocket: &xds_core.TransportSocket{
			Name: trafficMatch.Name,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		},
	}, nil
}

func buildTCPLocalRateLimitFilter(config *policyv1alpha1.TCPLocalRateLimitSpec, statPrefix string) (*xds_listener.Filter, error) {
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
		return nil, fmt.Errorf("invalid unit %q for TCP connection rate limiting", config.Unit)
	}

	rateLimit := &xds_network_local_ratelimit.LocalRateLimit{
		StatPrefix: statPrefix,
		TokenBucket: &xds_type.TokenBucket{
			MaxTokens:     config.Connections + config.Burst,
			TokensPerFill: wrapperspb.UInt32(config.Connections),
			FillInterval:  durationpb.New(fillInterval),
		},
	}

	marshalledConfig, err := anypb.New(rateLimit)
	if err != nil {
		return nil, err
	}

	filter := &xds_listener.Filter{
		Name:       envoy.L4LocalRateLimitFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledConfig},
	}

	return filter, nil
}

func buildTCPGlobalRateLimitFilter(config *policyv1alpha1.TCPGlobalRateLimitSpec, statPrefix string) (*xds_listener.Filter, error) {
	if config == nil {
		return nil, nil
	}

	rateLimit := &xds_global_ratelimit.RateLimit{
		StatPrefix: statPrefix,
		Domain:     config.Domain,
		RateLimitService: &xds_config_ratelimit.RateLimitServiceConfig{
			GrpcService: &xds_core.GrpcService{
				TargetSpecifier: &xds_core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &xds_core.GrpcService_EnvoyGrpc{
						ClusterName: service.RateLimitServiceClusterName(config.RateLimitService),
					},
				},
			},
			TransportApiVersion: xds_core.ApiVersion_V3,
		},
	}

	var descriptors []*xds_common_ratelimit.RateLimitDescriptor
	for _, desc := range config.Descriptors {
		var entries []*xds_common_ratelimit.RateLimitDescriptor_Entry
		for _, entry := range desc.Entries {
			entries = append(entries, &xds_common_ratelimit.RateLimitDescriptor_Entry{Key: entry.Key, Value: entry.Value})
		}

		descriptors = append(descriptors, &xds_common_ratelimit.RateLimitDescriptor{Entries: entries})
	}
	rateLimit.Descriptors = descriptors

	if config.Timeout != nil {
		rateLimit.Timeout = durationpb.New(config.Timeout.Duration)
		rateLimit.RateLimitService.GrpcService.Timeout = durationpb.New(config.Timeout.Duration)
	}

	if config.FailOpen != nil {
		rateLimit.FailureModeDeny = !*config.FailOpen
	}

	marshalledConfig, err := anypb.New(rateLimit)
	if err != nil {
		return nil, err
	}

	filter := &xds_listener.Filter{
		Name:       envoy.L4GlobalRateLimitFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledConfig},
	}

	return filter, nil
}

// buildOutboundFilterChainMatch builds a filter chain to match the HTTP or TCP based destination traffic.
// Filter Chain currently matches on the following:
// 1. Destination IP of service endpoints
// 2. Destination port of the service
func buildOutboundFilterChainMatch(trafficMatch trafficpolicy.TrafficMatch) (*xds_listener.FilterChainMatch, error) {
	filterMatch := &xds_listener.FilterChainMatch{
		DestinationPort: &wrapperspb.UInt32Value{
			Value: uint32(trafficMatch.DestinationPort),
		},
	}

	if len(trafficMatch.DestinationIPRanges) == 0 {
		return nil, fmt.Errorf("Destination IP ranges not specified for mesh upstream traffic match %s", trafficMatch.Name)
	}
	for _, ipRange := range trafficMatch.DestinationIPRanges {
		cidr, err := envoy.GetCIDRRangeFromStr(ipRange)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.ErrInvalidEgressIPRange.String()).
				Msgf("Error parsing IP range %s while building outbound mesh filter chain match %s, skipping", ipRange, trafficMatch.Name)
			continue
		}
		filterMatch.PrefixRanges = append(filterMatch.PrefixRanges, cidr)
	}

	return filterMatch, nil
}

func (lb *listenerBuilder) buildOutboundHTTPFilterChain(trafficMatch trafficpolicy.TrafficMatch) (*xds_listener.FilterChain, error) {
	// Get HTTP filter for service
	filter, err := lb.buildOutboundHTTPFilter(route.GetOutboundMeshRouteConfigNameForPort(trafficMatch.DestinationPort))
	if err != nil {
		log.Error().Err(err).Msgf("Error getting HTTP filter for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	// Get filter match criteria for destination service
	filterChainMatch, err := buildOutboundFilterChainMatch(trafficMatch)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting HTTP filter chain match for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	return &xds_listener.FilterChain{
		Name:             trafficMatch.Name,
		Filters:          []*xds_listener.Filter{filter},
		FilterChainMatch: filterChainMatch,
	}, nil
}

func (lb *listenerBuilder) buildOutboundTCPFilterChain(trafficMatch trafficpolicy.TrafficMatch) (*xds_listener.FilterChain, error) {
	// Build filters
	fb := getFilterBuilder().
		StatsPrefix(trafficMatch.Name)
	fb.TCPProxy().
		StatsPrefix(trafficMatch.Name).
		WeightedClusters(trafficMatch.WeightedClusters)

	filters, err := fb.Build()
	if err != nil {
		return nil, fmt.Errorf("error building inbound TCP filters: %w", err)
	}

	// Get filter match criteria for destination service
	filterChainMatch, err := buildOutboundFilterChainMatch(trafficMatch)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting HTTP filter chain match for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	return &xds_listener.FilterChain{
		Name:             trafficMatch.Name,
		Filters:          filters,
		FilterChainMatch: filterChainMatch,
	}, nil
}

// NEWCODE
// getOutboundFilterChainPerUpstream returns a list of filter chains corresponding to upstream services
func (lb *listenerBuilder) buildOutboundFilterChains() []*xds_listener.FilterChain {
	if lb.outboundMeshTrafficPolicy == nil {
		return nil
	}

	var filterChains []*xds_listener.FilterChain

	for _, trafficMatch := range lb.outboundMeshTrafficPolicy.TrafficMatches {
		log.Trace().Msgf("Building outbound mesh filter chain %s for proxy with identity %s", trafficMatch.Name, lb.proxyIdentity)
		// Create an outbound filter chain match per TrafficMatch object
		switch strings.ToLower(trafficMatch.DestinationProtocol) {
		case constants.ProtocolHTTP, constants.ProtocolGRPC:
			// Construct HTTP filter chain
			if httpFilterChain, err := lb.buildOutboundHTTPFilterChain(*trafficMatch); err != nil {
				log.Error().Err(err).Msgf("Error constructing outbound HTTP filter chain for traffic match %s on proxy with identity %s", trafficMatch.Name, lb.proxyIdentity)
			} else {
				filterChains = append(filterChains, httpFilterChain)
			}

		case constants.ProtocolTCP, constants.ProtocolTCPServerFirst:
			// Construct TCP filter chain
			if tcpFilterChain, err := lb.buildOutboundTCPFilterChain(*trafficMatch); err != nil {
				log.Error().Err(err).Msgf("Error constructing outbound TCP filter chain for traffic match %s on proxy with identity %s", trafficMatch.Name, lb.proxyIdentity)
			} else {
				filterChains = append(filterChains, tcpFilterChain)
			}

		default:
			log.Error().Msgf("Cannot build outbound filter chain, unsupported protocol %s for traffic match %s", trafficMatch.DestinationProtocol, trafficMatch.Name)
		}
	}

	return filterChains
}
