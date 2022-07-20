package lds

import (
	"fmt"
	"strings"
	"time"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_local_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/local_ratelimit/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/protobuf/ptypes/any"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	inboundMeshTCPProxyStatPrefix  = "inbound-mesh-tcp-proxy"
	outboundMeshTCPProxyStatPrefix = "outbound-mesh-tcp-proxy"
)

func (lb *listenerBuilder) getInboundMeshFilterChains(trafficMatches []*trafficpolicy.TrafficMatch) []*xds_listener.FilterChain {
	var filterChains []*xds_listener.FilterChain

	for _, match := range trafficMatches {
		// Create protocol specific inbound filter chains for MeshService's TargetPort
		switch strings.ToLower(match.DestinationProtocol) {
		case constants.ProtocolHTTP, constants.ProtocolGRPC:
			// Filter chain for HTTP port
			filterChainForPort, err := lb.getInboundMeshHTTPFilterChain(match)
			if err != nil {
				log.Error().Err(err).Msgf("Error building inbound HTTP filter chain for traffic match %s", match.Name)
			} else {
				filterChains = append(filterChains, filterChainForPort)
			}

		case constants.ProtocolTCP, constants.ProtocolTCPServerFirst:
			filterChainForPort, err := lb.getInboundMeshTCPFilterChain(match)
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

func (lb *listenerBuilder) getInboundHTTPFilters(trafficMatch *trafficpolicy.TrafficMatch) ([]*xds_listener.Filter, error) {
	if trafficMatch == nil {
		return nil, nil
	}

	var filters []*xds_listener.Filter

	// Apply an RBAC filter when permissive mode is disabled. The RBAC filter must be the first filter in the list of filters.
	if !lb.cfg.IsPermissiveTrafficPolicyMode() {
		// Apply RBAC policies on the inbound filters based on configured policies
		rbacFilter, err := lb.buildRBACFilter()
		if err != nil {
			log.Error().Err(err).Msgf("Error applying RBAC filter for traffic match %s", trafficMatch.Name)
			return nil, err
		}
		// RBAC filter should be the very first filter in the filter chain
		filters = append(filters, rbacFilter)
	}

	// Apply the network level local rate limit filter if configured for the TrafficMatch
	if trafficMatch.RateLimit != nil && trafficMatch.RateLimit.Local != nil && trafficMatch.RateLimit.Local.TCP != nil {
		rateLimitFilter, err := buildTCPLocalRateLimitFilter(trafficMatch.RateLimit.Local.TCP, trafficMatch.Name)
		if err != nil {
			return nil, err
		}
		filters = append(filters, rateLimitFilter)
	}

	// Build the HTTP Connection Manager filter from its options
	inboundConnManager, err := httpConnManagerOptions{
		direction:         inbound,
		rdsRoutConfigName: route.GetInboundMeshRouteConfigNameForPort(trafficMatch.DestinationPort),

		// Additional filters
		wasmStatsHeaders:         lb.getWASMStatsHeaders(),
		extAuthConfig:            lb.getExtAuthConfig(),
		enableActiveHealthChecks: lb.cfg.GetFeatureFlags().EnableEnvoyActiveHealthChecks,

		// Tracing options
		enableTracing:      lb.cfg.IsTracingEnabled(),
		tracingAPIEndpoint: lb.cfg.GetTracingEndpoint(),
	}.build()
	if err != nil {
		return nil, fmt.Errorf("Error building inbound HTTP connection manager for proxy with identity %s and traffic match %s: %w", lb.serviceIdentity, trafficMatch.Name, err)
	}

	marshalledInboundConnManager, err := anypb.New(inboundConnManager)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling inbound HTTP connection manager for proxy with identity %s and traffic match %s: %w", lb.serviceIdentity, trafficMatch.Name, err)
	}
	httpConnectionManagerFilter := &xds_listener.Filter{
		Name: envoy.HTTPConnectionManagerFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{
			TypedConfig: marshalledInboundConnManager,
		},
	}
	filters = append(filters, httpConnectionManagerFilter)

	return filters, nil
}

func (lb *listenerBuilder) getInboundMeshHTTPFilterChain(trafficMatch *trafficpolicy.TrafficMatch) (*xds_listener.FilterChain, error) {
	if trafficMatch == nil {
		return nil, nil
	}

	// Construct HTTP filters
	filters, err := lb.getInboundHTTPFilters(trafficMatch)
	if err != nil {
		log.Error().Err(err).Msgf("Error constructing inbound HTTP filters for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	// Construct downstream TLS context
	marshalledDownstreamTLSContext, err := anypb.New(envoy.GetDownstreamTLSContext(lb.serviceIdentity, true /* mTLS */, lb.cfg.GetMeshConfig().Spec.Sidecar))
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

func (lb *listenerBuilder) getInboundMeshTCPFilterChain(trafficMatch *trafficpolicy.TrafficMatch) (*xds_listener.FilterChain, error) {
	if trafficMatch == nil {
		return nil, nil
	}

	// Construct TCP filters
	filters, err := lb.getInboundTCPFilters(trafficMatch)
	if err != nil {
		log.Error().Err(err).Msgf("Error constructing inbound TCP filters for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	// Construct downstream TLS context
	marshalledDownstreamTLSContext, err := anypb.New(envoy.GetDownstreamTLSContext(lb.serviceIdentity, true /* mTLS */, lb.cfg.GetMeshConfig().Spec.Sidecar))
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

func (lb *listenerBuilder) getInboundTCPFilters(trafficMatch *trafficpolicy.TrafficMatch) ([]*xds_listener.Filter, error) {
	if trafficMatch == nil {
		return nil, nil
	}

	var filters []*xds_listener.Filter

	// Apply an RBAC filter when permissive mode is disabled. The RBAC filter must be the first filter in the list of filters.
	if !lb.cfg.IsPermissiveTrafficPolicyMode() {
		// Apply RBAC policies on the inbound filters based on configured policies
		rbacFilter, err := lb.buildRBACFilter()
		if err != nil {
			log.Error().Err(err).Msgf("Error applying RBAC filter for traffic match %s", trafficMatch.Name)
			return nil, err
		}
		// RBAC filter should be the very first filter in the filter chain
		filters = append(filters, rbacFilter)
	}

	// Apply the network level local rate limit filter if configured for the TrafficMatch
	if trafficMatch.RateLimit != nil && trafficMatch.RateLimit.Local != nil && trafficMatch.RateLimit.Local.TCP != nil {
		rateLimitFilter, err := buildTCPLocalRateLimitFilter(trafficMatch.RateLimit.Local.TCP, trafficMatch.Name)
		if err != nil {
			return nil, err
		}
		filters = append(filters, rateLimitFilter)
	}

	// Apply the TCP Proxy Filter
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix:       fmt.Sprintf("%s.%s", inboundMeshTCPProxyStatPrefix, trafficMatch.Cluster),
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: trafficMatch.Cluster},
	}
	marshalledTCPProxy, err := anypb.New(tcpProxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling TcpProxy object for egress HTTPS filter chain")
		return nil, err
	}
	tcpProxyFilter := &xds_listener.Filter{
		Name:       envoy.TCPProxyFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledTCPProxy},
	}
	filters = append(filters, tcpProxyFilter)

	return filters, nil
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

	rateLimit := &xds_local_ratelimit.LocalRateLimit{
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

// getOutboundHTTPFilter returns an HTTP connection manager network filter used to filter outbound HTTP traffic for the given route configuration
func (lb *listenerBuilder) getOutboundHTTPFilter(routeConfigName string) (*xds_listener.Filter, error) {
	var marshalledFilter *any.Any
	var err error

	// Build the HTTP connection manager filter from its options
	outboundConnManager, err := httpConnManagerOptions{
		direction:         outbound,
		rdsRoutConfigName: routeConfigName,

		// Additional filters
		wasmStatsHeaders: lb.statsHeaders,
		extAuthConfig:    nil, // Ext auth is not configured for outbound connections

		// Tracing options
		enableTracing:      lb.cfg.IsTracingEnabled(),
		tracingAPIEndpoint: lb.cfg.GetTracingEndpoint(),
	}.build()
	if err != nil {
		return nil, fmt.Errorf("Error building outbound HTTP connection manager for proxy identity %s", lb.serviceIdentity)
	}

	marshalledFilter, err = anypb.New(outboundConnManager)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling outbound HTTP connection manager for proxy identity %s", lb.serviceIdentity)
	}

	return &xds_listener.Filter{
		Name:       envoy.HTTPConnectionManagerFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledFilter},
	}, nil
}

// getOutboundFilterChainMatchForService builds a filter chain to match the HTTP or TCP based destination traffic.
// Filter Chain currently matches on the following:
// 1. Destination IP of service endpoints
// 2. Destination port of the service
func (lb *listenerBuilder) getOutboundFilterChainMatchForService(trafficMatch trafficpolicy.TrafficMatch) (*xds_listener.FilterChainMatch, error) {
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

func (lb *listenerBuilder) getOutboundHTTPFilterChainForService(trafficMatch trafficpolicy.TrafficMatch) (*xds_listener.FilterChain, error) {
	// Get HTTP filter for service
	filter, err := lb.getOutboundHTTPFilter(route.GetOutboundMeshRouteConfigNameForPort(trafficMatch.DestinationPort))
	if err != nil {
		log.Error().Err(err).Msgf("Error getting HTTP filter for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	// Get filter match criteria for destination service
	filterChainMatch, err := lb.getOutboundFilterChainMatchForService(trafficMatch)
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

func (lb *listenerBuilder) getOutboundTCPFilterChainForService(trafficMatch trafficpolicy.TrafficMatch) (*xds_listener.FilterChain, error) {
	// Get TCP filter for service
	filter, err := lb.getOutboundTCPFilter(trafficMatch)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting outbound TCP filter for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	// Get filter match criteria for destination service
	filterChainMatch, err := lb.getOutboundFilterChainMatchForService(trafficMatch)
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

func (lb *listenerBuilder) getOutboundTCPFilter(trafficMatch trafficpolicy.TrafficMatch) (*xds_listener.Filter, error) {
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix: fmt.Sprintf("%s_%s", outboundMeshTCPProxyStatPrefix, trafficMatch.Name),
	}

	if len(trafficMatch.WeightedClusters) == 0 {
		return nil, fmt.Errorf("At least 1 cluster must be configured for an upstream TCP service. None set for traffic match %s", trafficMatch.Name)
		// No weighted clusters implies a traffic split does not exist for this upstream, proxy it as is
	} else if len(trafficMatch.WeightedClusters) == 1 {
		tcpProxy.ClusterSpecifier = &xds_tcp_proxy.TcpProxy_Cluster{Cluster: trafficMatch.WeightedClusters[0].ClusterName.String()}
	} else {
		// Weighted clusters found for this upstream, proxy traffic meant for this upstream to its weighted clusters
		var clusterWeights []*xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight
		for _, cluster := range trafficMatch.WeightedClusters {
			clusterWeights = append(clusterWeights, &xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight{
				Name:   cluster.ClusterName.String(),
				Weight: uint32(cluster.Weight),
			})
		}
		tcpProxy.ClusterSpecifier = &xds_tcp_proxy.TcpProxy_WeightedClusters{
			WeightedClusters: &xds_tcp_proxy.TcpProxy_WeightedCluster{
				Clusters: clusterWeights,
			},
		}
	}

	marshalledTCPProxy, err := anypb.New(tcpProxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling TcpProxy object needed by outbound TCP filter for traffic match %s", trafficMatch.Name)
		return nil, err
	}

	return &xds_listener.Filter{
		Name:       envoy.TCPProxyFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledTCPProxy},
	}, nil
}

// getOutboundFilterChainPerUpstream returns a list of filter chains corresponding to upstream services
func (lb *listenerBuilder) getOutboundFilterChainPerUpstream() []*xds_listener.FilterChain {
	var filterChains []*xds_listener.FilterChain

	outboundMeshTrafficPolicy := lb.meshCatalog.GetOutboundMeshTrafficPolicy(lb.serviceIdentity)
	if outboundMeshTrafficPolicy == nil {
		// no outbound mesh traffic policies
		return nil
	}

	for _, trafficMatch := range outboundMeshTrafficPolicy.TrafficMatches {
		log.Trace().Msgf("Building outbound mesh filter chain %s for proxy with identity %s", trafficMatch.Name, lb.serviceIdentity)
		// Create an outbound filter chain match per TrafficMatch object
		switch strings.ToLower(trafficMatch.DestinationProtocol) {
		case constants.ProtocolHTTP, constants.ProtocolGRPC:
			// Construct HTTP filter chain
			if httpFilterChain, err := lb.getOutboundHTTPFilterChainForService(*trafficMatch); err != nil {
				log.Error().Err(err).Msgf("Error constructing outbound HTTP filter chain for traffic match %s on proxy with identity %s", trafficMatch.Name, lb.serviceIdentity)
			} else {
				filterChains = append(filterChains, httpFilterChain)
			}

		case constants.ProtocolTCP, constants.ProtocolTCPServerFirst:
			// Construct TCP filter chain
			if tcpFilterChain, err := lb.getOutboundTCPFilterChainForService(*trafficMatch); err != nil {
				log.Error().Err(err).Msgf("Error constructing outbound TCP filter chain for traffic match %s on proxy with identity %s", trafficMatch.Name, lb.serviceIdentity)
			} else {
				filterChains = append(filterChains, tcpFilterChain)
			}

		default:
			log.Error().Msgf("Cannot build outbound filter chain, unsupported protocol %s for traffic match %s", trafficMatch.DestinationProtocol, trafficMatch.Name)
		}
	}

	return filterChains
}
