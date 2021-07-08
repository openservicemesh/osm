package lds

import (
	"fmt"
	"sort"
	"strings"

	mapset "github.com/deckarep/golang-set"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds/route"
	"github.com/openservicemesh/osm/pkg/service"
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

	protocolToPortMap, err := lb.meshCatalog.GetTargetPortToProtocolMappingForService(proxyService)
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving port to protocol mapping for service %s", proxyService)
		return filterChains
	}

	// Create protocol specific inbound filter chains per port to handle different ports serving different protocols
	for port, appProtocol := range protocolToPortMap {
		switch strings.ToLower(appProtocol) {
		case constants.ProtocolHTTP, constants.ProtocolGRPC:
			// Filter chain for HTTP port
			filterChainForPort, err := lb.getInboundMeshHTTPFilterChain(proxyService, port)
			if err != nil {
				log.Error().Err(err).Msgf("Error building inbound HTTP filter chain for proxy:port %s:%d", proxyService, port)
				continue // continue building filter chains for other ports on the service
			}
			filterChains = append(filterChains, filterChainForPort)

		case constants.ProtocolTCP:
			filterChainForPort, err := lb.getInboundMeshTCPFilterChain(proxyService, port)
			if err != nil {
				log.Error().Err(err).Msgf("Error building inbound TCP filter chain for proxy:port %s:%d", proxyService, port)
				continue // continue building filter chains for other ports on the service
			}
			filterChains = append(filterChains, filterChainForPort)

		default:
			log.Error().Msgf("Cannot build inbound filter chain, unsupported protocol %s for proxy:port %s:%d", appProtocol, proxyService, port)
		}
	}

	return filterChains
}

func (lb *listenerBuilder) getInboundHTTPFilters(proxyService service.MeshService) ([]*xds_listener.Filter, error) {
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

	// Apply the HTTP Connection Manager Filter
	inboundConnManager := getHTTPConnectionManager(route.InboundRouteConfigName, lb.cfg, lb.statsHeaders, inbound)
	marshalledInboundConnManager, err := ptypes.MarshalAny(inboundConnManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling inbound HttpConnectionManager for proxy  service %s", proxyService)
		return nil, err
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
	filters, err := lb.getInboundHTTPFilters(proxyService)
	if err != nil {
		log.Error().Err(err).Msgf("Error constructing inbound HTTP filters for proxy service %s", proxyService)
		return nil, err
	}

	// Construct downstream TLS context
	marshalledDownstreamTLSContext, err := ptypes.MarshalAny(envoy.GetDownstreamTLSContext(lb.serviceIdentity, true /* mTLS */))
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling DownstreamTLSContext for proxy service %s", proxyService)
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
	marshalledDownstreamTLSContext, err := ptypes.MarshalAny(envoy.GetDownstreamTLSContext(lb.serviceIdentity, true /* mTLS */))
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling DownstreamTLSContext for proxy service %s", proxyService)
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
	localServiceCluster := envoy.GetLocalClusterNameForService(proxyService)
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix:       fmt.Sprintf("%s.%s", inboundMeshTCPProxyStatPrefix, localServiceCluster),
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: localServiceCluster},
	}
	marshalledTCPProxy, err := ptypes.MarshalAny(tcpProxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling TcpProxy object for egress HTTPS filter chain")
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

	marshalledFilter, err = ptypes.MarshalAny(
		getHTTPConnectionManager(routeConfigName, lb.cfg, lb.statsHeaders, outbound))
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HTTP connection manager object")
		return nil, err
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
func (lb *listenerBuilder) getOutboundFilterChainMatchForService(dstSvc service.MeshService, port uint32) (*xds_listener.FilterChainMatch, error) {
	filterMatch := &xds_listener.FilterChainMatch{
		DestinationPort: &wrapperspb.UInt32Value{
			Value: port,
		},
	}

	endpoints, err := lb.meshCatalog.GetResolvableServiceEndpoints(dstSvc)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting GetResolvableServiceEndpoints for %q", dstSvc)
		return nil, err
	}

	if len(endpoints) == 0 {
		err := errors.Errorf("Endpoints not found for service %q", dstSvc)
		log.Error().Err(err).Msgf("Error constructing HTTP filter chain match for service %q", dstSvc)
		return nil, err
	}

	endpointSet := mapset.NewSet()
	for _, endp := range endpoints {
		endpointSet.Add(endp.IP.String())
	}

	// For deterministic ordering
	var sortedEndpoints []string
	endpointSet.Each(func(elem interface{}) bool {
		sortedEndpoints = append(sortedEndpoints, elem.(string))
		return false
	})
	sort.Strings(sortedEndpoints)

	for _, ip := range sortedEndpoints {
		filterMatch.PrefixRanges = append(filterMatch.PrefixRanges, &xds_core.CidrRange{
			AddressPrefix: ip,
			PrefixLen: &wrapperspb.UInt32Value{
				Value: singleIpv4Mask,
			},
		})
	}

	return filterMatch, nil
}

func (lb *listenerBuilder) getOutboundHTTPFilterChainForService(upstream service.MeshService, port uint32) (*xds_listener.FilterChain, error) {
	// Get HTTP filter for service
	filter, err := lb.getOutboundHTTPFilter(route.OutboundRouteConfigName)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting HTTP filter for upstream service %s", upstream)
		return nil, err
	}

	// Get filter match criteria for destination service
	filterChainMatch, err := lb.getOutboundFilterChainMatchForService(upstream, port)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting HTTP filter chain match for upstream service %s", upstream)
		return nil, err
	}

	filterChainName := fmt.Sprintf("%s:%s", outboundMeshHTTPFilterChainPrefix, upstream)
	return &xds_listener.FilterChain{
		Name:             filterChainName,
		Filters:          []*xds_listener.Filter{filter},
		FilterChainMatch: filterChainMatch,
	}, nil
}

func (lb *listenerBuilder) getOutboundTCPFilterChainForService(upstream service.MeshService, port uint32) (*xds_listener.FilterChain, error) {
	// Get TCP filter for service
	filter, err := lb.getOutboundTCPFilter(upstream)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting outbound TCP filter for upstream service %s", upstream)
		return nil, err
	}

	// Get filter match criteria for destination service
	filterChainMatch, err := lb.getOutboundFilterChainMatchForService(upstream, port)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting HTTP filter chain match for upstream service %s", upstream)
		return nil, err
	}

	filterChainName := fmt.Sprintf("%s:%s", outboundMeshTCPFilterChainPrefix, upstream)
	return &xds_listener.FilterChain{
		Name:             filterChainName,
		Filters:          []*xds_listener.Filter{filter},
		FilterChainMatch: filterChainMatch,
	}, nil
}

func (lb *listenerBuilder) getOutboundTCPFilter(upstream service.MeshService) (*xds_listener.Filter, error) {
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix: fmt.Sprintf("%s.%s", outboundMeshTCPProxyStatPrefix, upstream),
	}

	weightedClusters := lb.meshCatalog.GetWeightedClustersForUpstream(upstream)

	if len(weightedClusters) == 0 {
		// No weighted clusters implies a traffic split does not exist for this upstream, proxy it as is
		tcpProxy.ClusterSpecifier = &xds_tcp_proxy.TcpProxy_Cluster{Cluster: upstream.String()}
	} else {
		// Weighted clusters found for this upstream, proxy traffic meant for this upstream to its weighted clusters
		var clusterWeights []*xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight
		for _, cluster := range weightedClusters {
			clusterWeights = append(clusterWeights, &xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight{
				Name:   string(cluster.ClusterName),
				Weight: uint32(cluster.Weight),
			})
		}
		tcpProxy.ClusterSpecifier = &xds_tcp_proxy.TcpProxy_WeightedClusters{
			WeightedClusters: &xds_tcp_proxy.TcpProxy_WeightedCluster{
				Clusters: clusterWeights,
			},
		}
	}

	marshalledTCPProxy, err := ptypes.MarshalAny(tcpProxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling TcpProxy object needed by outbound TCP filter for upstream service %s", upstream)
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

	upstreamServices := lb.meshCatalog.ListMeshServicesForIdentity(lb.serviceIdentity)
	if len(upstreamServices) == 0 {
		log.Debug().Msgf("Proxy with identity %s does not have any allowed upstream services", lb.serviceIdentity)
		return filterChains
	}

	// Iterate all destination services
	for _, upstreamSvc := range upstreamServices {
		log.Trace().Msgf("Building outbound filter chain for upstream service %s for proxy with identity %s", upstreamSvc, lb.serviceIdentity)
		protocolToPortMap, err := lb.meshCatalog.GetPortToProtocolMappingForService(upstreamSvc)
		if err != nil {
			log.Error().Err(err).Msgf("Error retrieving port to protocol mapping for upstream service %s", upstreamSvc)
			continue
		}

		// Create protocol specific inbound filter chains per port to handle different ports serving different protocols
		for port, appProtocol := range protocolToPortMap {
			switch strings.ToLower(appProtocol) {
			case constants.ProtocolHTTP, constants.ProtocolGRPC:
				// Construct HTTP filter chain
				if httpFilterChain, err := lb.getOutboundHTTPFilterChainForService(upstreamSvc, port); err != nil {
					log.Error().Err(err).Msgf("Error constructing outbound HTTP filter chain for upstream service %s on proxy with identity %s", upstreamSvc, lb.serviceIdentity)
				} else {
					filterChains = append(filterChains, httpFilterChain)
				}

			case constants.ProtocolTCP:
				// Construct TCP filter chain
				if tcpFilterChain, err := lb.getOutboundTCPFilterChainForService(upstreamSvc, port); err != nil {
					log.Error().Err(err).Msgf("Error constructing outbound TCP filter chain for upstream service %s on proxy with identity %s", upstreamSvc, lb.serviceIdentity)
				} else {
					filterChains = append(filterChains, tcpFilterChain)
				}

			default:
				log.Error().Msgf("Cannot build outbound filter chain, unsupported protocol %s for upstream:port %s:%d", appProtocol, upstreamSvc, port)
			}
		}
	}

	return filterChains
}
