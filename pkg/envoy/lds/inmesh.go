package lds

import (
	"fmt"
	"strings"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	inboundMeshHTTPFilterChainPrefix  = "inbound-mesh-http-filter-chain"
	outboundMeshHTTPFilterChainPrefix = "outbound-mesh-http-filter-chain"
	inboundMeshTCPFilterChainPrefix   = "inbound-mesh-tcp-filter-chain"
	outboundMeshTCPFilterChainPrefix  = "outbound-mesh-tcp-filter-chain"
	inboundMeshTCPProxyStatPrefix     = "inbound-mesh-tcp-proxy"
	outboundMeshTCPProxyStatPrefix    = "outbound-mesh-tcp-proxy"
)

func (lb *listenerBuilder) getInboundMeshFilterChains(proxyService service.MeshService) []*xds_listener.FilterChain {
	var filterChains []*xds_listener.FilterChain

	// Create protocol specific inbound filter chains for MeshService's TargetPort
	switch strings.ToLower(proxyService.Protocol) {
	case constants.ProtocolHTTP, constants.ProtocolGRPC:
		// Filter chain for HTTP port
		filterChainForPort, err := lb.getInboundMeshHTTPFilterChain(proxyService, uint32(proxyService.TargetPort))
		if err != nil {
			log.Error().Err(err).Msgf("Error building inbound HTTP filter chain for proxy:port %s:%d", proxyService, proxyService.TargetPort)
		}
		filterChains = append(filterChains, filterChainForPort)

	case constants.ProtocolTCP, constants.ProtocolTCPServerFirst:
		filterChainForPort, err := lb.getInboundMeshTCPFilterChain(proxyService, uint32(proxyService.TargetPort))
		if err != nil {
			log.Error().Err(err).Msgf("Error building inbound TCP filter chain for proxy:port %s:%d", proxyService, proxyService.TargetPort)
		}
		filterChains = append(filterChains, filterChainForPort)

	default:
		log.Error().Msgf("Cannot build inbound filter chain, unsupported protocol %s for proxy-service:port %s:%d", proxyService.Protocol, proxyService, proxyService.TargetPort)
	}

	return filterChains
}

func (lb *listenerBuilder) getInboundHTTPFilters(proxyService service.MeshService, servicePort uint32) ([]*xds_listener.Filter, error) {
	var filters []*xds_listener.Filter

	// Apply an RBAC filter when permissive mode is disabled. The RBAC filter must be the first filter in the list of filters.
	if !lb.cfg.IsPermissiveTrafficPolicyMode() {
		// Apply RBAC policies on the inbound filters based on configured policies
		rbacFilter, err := lb.buildRBACFilter()
		if err != nil {
			log.Error().Err(err).Msgf("Error applying RBAC filter for proxy service %s", proxyService)
			return nil, err
		}
		// RBAC filter should be the very first filter in the filter chain
		filters = append(filters, rbacFilter)
	}

	// Build the HTTP Connection Manager filter from its options
	inboundConnManager, err := httpConnManagerOptions{
		direction:         inbound,
		rdsRoutConfigName: route.GetInboundMeshRouteConfigNameForPort(int(servicePort)),

		// Additional filters
		wasmStatsHeaders:         lb.getWASMStatsHeaders(),
		extAuthConfig:            lb.getExtAuthConfig(),
		enableActiveHealthChecks: lb.cfg.GetFeatureFlags().EnableEnvoyActiveHealthChecks,

		// Tracing options
		enableTracing:      lb.cfg.IsTracingEnabled(),
		tracingAPIEndpoint: lb.cfg.GetTracingEndpoint(),
	}.build()
	if err != nil {
		return nil, errors.Wrapf(err, "Error building inbound HTTP connection manager for proxy with identity %s and service %s", lb.serviceIdentity, proxyService)
	}

	marshalledInboundConnManager, err := anypb.New(inboundConnManager)
	if err != nil {
		return nil, errors.Wrapf(err, "Error marshalling inbound HTTP connection manager for proxy with identity %s and service %s", lb.serviceIdentity, proxyService)
	}
	httpConnectionManagerFilter := &xds_listener.Filter{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &xds_listener.Filter_TypedConfig{
			TypedConfig: marshalledInboundConnManager,
		},
	}
	filters = append(filters, httpConnectionManagerFilter)

	return filters, nil
}

func (lb *listenerBuilder) getInboundMeshHTTPFilterChain(proxyService service.MeshService, servicePort uint32) (*xds_listener.FilterChain, error) {
	// Construct HTTP filters
	filters, err := lb.getInboundHTTPFilters(proxyService, servicePort)
	if err != nil {
		log.Error().Err(err).Msgf("Error constructing inbound HTTP filters for proxy service %s", proxyService)
		return nil, err
	}

	// Construct downstream TLS context
	marshalledDownstreamTLSContext, err := anypb.New(envoy.GetDownstreamTLSContext(lb.serviceIdentity, true /* mTLS */, lb.cfg.GetMeshConfig().Spec.Sidecar))
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling DownstreamTLSContext for proxy service %s", proxyService)
		return nil, err
	}

	filterchainName := fmt.Sprintf("%s:%s:%d", inboundMeshHTTPFilterChainPrefix, proxyService, servicePort)

	serverNames := []string{proxyService.ServerName()}

	filterChain := &xds_listener.FilterChain{
		Name:    filterchainName,
		Filters: filters,

		// The 'FilterChainMatch' field defines the criteria for matching traffic against filters in this filter chain
		FilterChainMatch: &xds_listener.FilterChainMatch{
			// The DestinationPort is the service port the downstream directs traffic to
			DestinationPort: &wrapperspb.UInt32Value{
				Value: servicePort,
			},

			// The ServerName is the SNI set by the downstream in the UptreamTlsContext by GetUpstreamTLSContext()
			// This is not a field obtained from the mTLS Certificate.
			ServerNames: serverNames,

			// Only match when transport protocol is TLS
			TransportProtocol: envoy.TransportProtocolTLS,

			// In-mesh proxies will advertise this, set in the UpstreamTlsContext by GetUpstreamTLSContext()
			ApplicationProtocols: envoy.ALPNInMesh,
		},

		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		},
	}

	return filterChain, nil
}

func (lb *listenerBuilder) getInboundMeshTCPFilterChain(proxyService service.MeshService, servicePort uint32) (*xds_listener.FilterChain, error) {
	// Construct TCP filters
	filters, err := lb.getInboundTCPFilters(proxyService)
	if err != nil {
		log.Error().Err(err).Msgf("Error constructing inbound TCP filters for proxy service %s", proxyService)
		return nil, err
	}

	// Construct downstream TLS context
	marshalledDownstreamTLSContext, err := anypb.New(envoy.GetDownstreamTLSContext(lb.serviceIdentity, true /* mTLS */, lb.cfg.GetMeshConfig().Spec.Sidecar))
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling DownstreamTLSContext for proxy service %s", proxyService)
		return nil, err
	}

	serverNames := []string{proxyService.ServerName()}

	filterchainName := fmt.Sprintf("%s:%s:%d", inboundMeshTCPFilterChainPrefix, proxyService, servicePort)
	return &xds_listener.FilterChain{
		Name: filterchainName,
		FilterChainMatch: &xds_listener.FilterChainMatch{
			// The DestinationPort is the service port the downstream directs traffic to
			DestinationPort: &wrapperspb.UInt32Value{
				Value: servicePort,
			},

			// The ServerName is the SNI set by the downstream in the UptreamTlsContext by GetUpstreamTLSContext()
			// This is not a field obtained from the mTLS Certificate.
			ServerNames: serverNames,

			// Only match when transport protocol is TLS
			TransportProtocol: envoy.TransportProtocolTLS,

			// In-mesh proxies will advertise this, set in the UpstreamTlsContext by GetUpstreamTLSContext()
			ApplicationProtocols: envoy.ALPNInMesh,
		},
		Filters: filters,
		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		},
	}, nil
}

func (lb *listenerBuilder) getInboundTCPFilters(proxyService service.MeshService) ([]*xds_listener.Filter, error) {
	var filters []*xds_listener.Filter

	// Apply an RBAC filter when permissive mode is disabled. The RBAC filter must be the first filter in the list of filters.
	if !lb.cfg.IsPermissiveTrafficPolicyMode() {
		// Apply RBAC policies on the inbound filters based on configured policies
		rbacFilter, err := lb.buildRBACFilter()
		if err != nil {
			log.Error().Err(err).Msgf("Error applying RBAC filter for proxy service %s", proxyService)
			return nil, err
		}
		// RBAC filter should be the very first filter in the filter chain
		filters = append(filters, rbacFilter)
	}

	// Apply the TCP Proxy Filter
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix:       fmt.Sprintf("%s.%s", inboundMeshTCPProxyStatPrefix, proxyService.EnvoyLocalClusterName()),
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: proxyService.EnvoyLocalClusterName()},
	}
	marshalledTCPProxy, err := anypb.New(tcpProxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingXDSResource)).
			Msgf("Error marshalling TcpProxy object for egress HTTPS filter chain")
		return nil, err
	}
	tcpProxyFilter := &xds_listener.Filter{
		Name:       wellknown.TCPProxy,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledTCPProxy},
	}
	filters = append(filters, tcpProxyFilter)

	return filters, nil
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
		return nil, errors.Wrapf(err, "Error building outbound HTTP connection manager for proxy identity %s", lb.serviceIdentity)
	}

	marshalledFilter, err = anypb.New(outboundConnManager)
	if err != nil {
		return nil, errors.Wrapf(err, "Error marshalling outbound HTTP connection manager for proxy identity %s", lb.serviceIdentity)
	}

	return &xds_listener.Filter{
		Name:       wellknown.HTTPConnectionManager,
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
		return nil, errors.Errorf("Destination IP ranges not specified for mesh upstream traffic match %s", trafficMatch.Name)
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

	filterChainName := fmt.Sprintf("%s:%s", outboundMeshHTTPFilterChainPrefix, trafficMatch.Name)
	return &xds_listener.FilterChain{
		Name:             filterChainName,
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

	filterChainName := fmt.Sprintf("%s:%s", outboundMeshTCPFilterChainPrefix, trafficMatch.Name)
	return &xds_listener.FilterChain{
		Name:             filterChainName,
		Filters:          []*xds_listener.Filter{filter},
		FilterChainMatch: filterChainMatch,
	}, nil
}

func (lb *listenerBuilder) getOutboundTCPFilter(trafficMatch trafficpolicy.TrafficMatch) (*xds_listener.Filter, error) {
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix: fmt.Sprintf("%s_%s", outboundMeshTCPProxyStatPrefix, trafficMatch.Name),
	}

	if len(trafficMatch.WeightedClusters) == 0 {
		return nil, errors.Errorf("At least 1 cluster must be configured for an upstream TCP service. None set for traffic match %s", trafficMatch.Name)
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
		Name:       wellknown.TCPProxy,
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
