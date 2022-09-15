package lds

import (
	"errors"
	"fmt"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_http_local_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/anypb"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/auth"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/protobuf"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	websocketUpgradeType = "websocket"
)

func ListenerBuilder() *listenerBuilder { //nolint: revive // unexported-return
	return &listenerBuilder{}
}

func (lb *listenerBuilder) Name(name string) *listenerBuilder {
	lb.name = name
	return lb
}

func (lb *listenerBuilder) ProxyIdentity(si identity.ServiceIdentity) *listenerBuilder {
	lb.proxyIdentity = si
	return lb
}

func (lb *listenerBuilder) Address(address string, port uint32) *listenerBuilder {
	lb.address = envoy.GetAddress(address, port)
	return lb
}

func (lb *listenerBuilder) TrafficDirection(dir xds_core.TrafficDirection) *listenerBuilder {
	lb.trafficDirection = dir
	return lb
}

func (lb *listenerBuilder) OutboundMeshTrafficPolicy(t *trafficpolicy.OutboundMeshTrafficPolicy) *listenerBuilder {
	lb.outboundMeshTrafficPolicy = t
	return lb
}

func (lb *listenerBuilder) InboundMeshTrafficPolicy(t *trafficpolicy.InboundMeshTrafficPolicy) *listenerBuilder {
	lb.inboundMeshTrafficPolicy = t
	return lb
}

func (lb *listenerBuilder) EgressTrafficPolicy(t *trafficpolicy.EgressTrafficPolicy) *listenerBuilder {
	lb.egressTrafficPolicy = t
	return lb
}

func (lb *listenerBuilder) IngressTrafficPolicies(t []*trafficpolicy.IngressTrafficPolicy) *listenerBuilder {
	lb.ingressTrafficPolicies = t
	return lb
}

func (lb *listenerBuilder) PermissiveMesh(enable bool) *listenerBuilder {
	lb.permissiveMesh = enable
	return lb
}

func (lb *listenerBuilder) PermissiveEgress(enable bool) *listenerBuilder {
	lb.permissiveEgress = enable
	return lb
}

func (lb *listenerBuilder) getFilterBuilder() *filterBuilder {
	if lb.filBuilder == nil {
		lb.filBuilder = getFilterBuilder()
	}
	return lb.filBuilder
}

func (lb *listenerBuilder) defaultOutboundListenerFilters() *listenerBuilder {
	lb.listenerFilters = append(lb.listenerFilters,
		&xds_listener.ListenerFilter{
			// The OriginalDestination ListenerFilter is used to restore the original destination address
			// as opposed to the listener's address (due to iptables redirection).
			// This enables  filter chain matching on the original destination address (ip, port).
			Name: envoy.OriginalDstFilterName,
			ConfigType: &xds_listener.ListenerFilter_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: envoy.OriginalDstFilterTypeURL,
				},
			},
		},
	)
	return lb
}

func (lb *listenerBuilder) DefaultInboundListenerFilters() *listenerBuilder {
	lb.listenerFilters = append(lb.listenerFilters,
		&xds_listener.ListenerFilter{
			// To inspect TLS metadata, such as the transport protocol and SNI
			Name: envoy.TLSInspectorFilterName,
			ConfigType: &xds_listener.ListenerFilter_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: envoy.TLSInspectorFilterTypeURL,
				},
			},
		},
		&xds_listener.ListenerFilter{
			// The OriginalDestination ListenerFilter is used to restore the original destination address
			// as opposed to the listener's address (due to iptables redirection).
			// This enables  filter chain matching on the original destination address (ip, port).
			Name: envoy.OriginalDstFilterName,
			ConfigType: &xds_listener.ListenerFilter_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: envoy.OriginalDstFilterTypeURL,
				},
			},
		},
	)
	return lb
}

func (lb *listenerBuilder) TracingEndpoint(endpoint string) *listenerBuilder {
	lb.httpTracingEndpoint = endpoint
	return lb
}

func (lb *listenerBuilder) ExtAuthzConfig(config *auth.ExtAuthConfig) *listenerBuilder {
	lb.extAuthzConfig = config
	return lb
}

func (lb *listenerBuilder) WASMStatsHeaders(headers map[string]string) *listenerBuilder {
	lb.wasmStatsHeaders = headers
	return lb
}

func (lb *listenerBuilder) ActiveHealthCheck(enable bool) *listenerBuilder {
	lb.activeHealthCheck = enable
	return lb
}

func (lb *listenerBuilder) TrafficTargets(t []trafficpolicy.TrafficTargetWithRoutes) *listenerBuilder {
	lb.trafficTargets = t
	return lb
}

func (lb *listenerBuilder) TrustDomain(trustDomain string) *listenerBuilder {
	lb.trustDomain = trustDomain
	return lb
}

func (lb *listenerBuilder) SidecarSpec(sidecarSpec configv1alpha2.SidecarSpec) *listenerBuilder {
	lb.sidecarSpec = sidecarSpec
	return lb
}

func (lb *listenerBuilder) Build() (*xds_listener.Listener, error) {
	var l *xds_listener.Listener
	switch lb.trafficDirection {
	case xds_core.TrafficDirection_OUTBOUND:
		l = lb.buildOutboundListener()

	case xds_core.TrafficDirection_INBOUND:
		l = lb.buildInboundListener()

	default:
		return nil, fmt.Errorf("listener %s: unsupported traffic direction %s", l.Name, l.TrafficDirection)
	}

	if len(l.FilterChains) == 0 && l.DefaultFilterChain == nil {
		// Programming a listener with no filter chains is an error.
		// It is possible for the listener to have no filter chains if
		// there are no configurations that permit traffic through this proxy.
		// In this case, return a nil filter chain so that it doesn't get programmed.
		return nil, nil
	}

	return l, l.Validate()
}

func (lb *listenerBuilder) buildOutboundListener() *xds_listener.Listener {
	lb.defaultOutboundListenerFilters()

	l := &xds_listener.Listener{
		Name:             lb.name,
		Address:          lb.address,
		TrafficDirection: lb.trafficDirection,
		ListenerFilters:  lb.listenerFilters,
		AccessLog:        envoy.GetAccessLog(),
	}

	var outboundTrafficMatches []*trafficpolicy.TrafficMatch // used to configure FilterDisabled match predicate

	l.FilterChains = lb.buildOutboundFilterChains()
	if lb.outboundMeshTrafficPolicy != nil {
		outboundTrafficMatches = append(outboundTrafficMatches, lb.outboundMeshTrafficPolicy.TrafficMatches...)
	}

	if lb.permissiveEgress {
		// Enable permissive (global) egress to unknown destinations
		l.DefaultFilterChain = getDefaultPassthroughFilterChain()
	} else if lb.egressTrafficPolicy != nil {
		// Build Egress policy filter chains
		l.FilterChains = append(l.FilterChains, lb.getEgressFilterChainsForMatches(lb.egressTrafficPolicy.TrafficMatches)...)
		outboundTrafficMatches = append(outboundTrafficMatches, lb.egressTrafficPolicy.TrafficMatches...)
	}

	var filterDisableMatchPredicate *xds_listener.ListenerFilterChainMatchPredicate
	if len(outboundTrafficMatches) > 0 {
		filterDisableMatchPredicate = getFilterMatchPredicateForTrafficMatches(outboundTrafficMatches)
	}

	l.ListenerFilters = append(l.ListenerFilters,
		// Configure match predicate for ports serving server-first protocols (ex. mySQL, postgreSQL etc.).
		// Ports corresponding to server-first protocols, where the server initiates the first byte of a connection, will
		// cause the HttpInspector ListenerFilter to timeout because it waits for data from the client to inspect the protocol.
		// Such ports will set the protocol to 'tcp-server-first' in an Egress policy.
		// The 'FilterDisabled' field configures the match predicate.
		&xds_listener.ListenerFilter{
			// To inspect TLS metadata, such as the transport protocol and SNI
			Name: envoy.TLSInspectorFilterName,
			ConfigType: &xds_listener.ListenerFilter_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: envoy.TLSInspectorFilterTypeURL,
				},
			},
			FilterDisabled: filterDisableMatchPredicate,
		},
		&xds_listener.ListenerFilter{
			// To inspect if the application protocol is HTTP based
			Name: envoy.HTTPInspectorFilterName,
			ConfigType: &xds_listener.ListenerFilter_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: envoy.HTTPInspectorFilterTypeURL,
				},
			},
			FilterDisabled: filterDisableMatchPredicate,
		},
	)

	// ListenerFilter can timeout for server-first protocols. In such cases, continue the processing of the connection
	// and fallback to the default filter chain.
	l.ContinueOnListenerFiltersTimeout = true

	return l
}

func (lb *listenerBuilder) buildInboundListener() *xds_listener.Listener {
	l := &xds_listener.Listener{
		Name:             lb.name,
		Address:          lb.address,
		TrafficDirection: lb.trafficDirection,
		ListenerFilters:  lb.listenerFilters,
		FilterChains:     lb.buildInboundFilterChains(),
		AccessLog:        envoy.GetAccessLog(),
	}

	return l
}

// buildOutboundHTTPFilter returns an HTTP connection manager network filter used to filter outbound HTTP traffic for the given route configuration
func (lb *listenerBuilder) buildOutboundHTTPFilter(routeConfigName string) (*xds_listener.Filter, error) {
	hb := HTTPConnManagerBuilder()
	hb.StatsPrefix(routeConfigName).
		RouteConfigName(routeConfigName)

	if lb.httpTracingEndpoint != "" {
		tracing, err := getHTTPTracingConfig(lb.httpTracingEndpoint)
		if err != nil {
			return nil, fmt.Errorf("error building outbound http filter: %w", err)
		}
		hb.Tracing(tracing)
	}
	if lb.wasmStatsHeaders != nil {
		wasmFilters, wasmLocalReplyConfig, err := getWASMStatsConfig(lb.wasmStatsHeaders)
		if err != nil {
			return nil, fmt.Errorf("error building outbound http filter: %w", err)
		}
		hb.LocalReplyConfig(wasmLocalReplyConfig)
		for _, f := range wasmFilters {
			hb.AddFilter(f)
		}
	}

	return hb.Build()
}

func (lb *listenerBuilder) buildInboundFilterChains() []*xds_listener.FilterChain {
	var filterChains []*xds_listener.FilterChain

	filterChains = append(filterChains, lb.buildInboundMeshFilterChains()...)
	filterChains = append(filterChains, lb.buildIngressFilterChains()...)

	return filterChains
}

// FilterBuilder returns an instance used to build filters
func getFilterBuilder() *filterBuilder { //nolint: revive // unexported-return
	return &filterBuilder{}
}

// StatsPrefix sets the stats prefix for the filter
func (fb *filterBuilder) StatsPrefix(statsPrefix string) *filterBuilder {
	fb.statsPrefix = statsPrefix
	return fb
}

// WithRBAC sets the RBAC properties used to build the filter
func (fb *filterBuilder) WithRBAC(t []trafficpolicy.TrafficTargetWithRoutes, trustDomain string) *filterBuilder {
	fb.withRBAC = true
	fb.trafficTargets = t
	fb.trustDomain = trustDomain
	return fb
}

func (fb *filterBuilder) TCPLocalRateLimit(rl *policyv1alpha1.TCPLocalRateLimitSpec) *filterBuilder {
	fb.tcpLocalRateLimit = rl
	return fb
}

func (fb *filterBuilder) TCPGlobalRateLimit(rl *policyv1alpha1.TCPGlobalRateLimitSpec) *filterBuilder {
	fb.tcpGlobalRateLimit = rl
	return fb
}

func (fb *filterBuilder) httpConnManager() *httpConnManagerBuilder {
	if fb.hcmBuilder == nil {
		fb.hcmBuilder = HTTPConnManagerBuilder()
	}

	return fb.hcmBuilder
}

func (fb *filterBuilder) TCPProxy() *tcpProxyBuilder {
	if fb.tcpProxyBuilder == nil {
		fb.tcpProxyBuilder = TCPProxyBuilder()
	}
	return fb.tcpProxyBuilder
}

func (fb *filterBuilder) Build() ([]*xds_listener.Filter, error) {
	var filters []*xds_listener.Filter

	// RBAC filter should be the very first filter in the filter chain
	if fb.withRBAC {
		rbacFilter, err := buildRBACFilter(fb.trafficTargets, fb.trustDomain)
		if err != nil {
			return nil, err
		}
		filters = append(filters, rbacFilter)
	}

	// Rate limit filters
	if fb.tcpLocalRateLimit != nil {
		rateLimitFilter, err := buildTCPLocalRateLimitFilter(fb.tcpLocalRateLimit, fb.statsPrefix)
		if err != nil {
			return nil, err
		}
		filters = append(filters, rateLimitFilter)
	}
	if fb.tcpGlobalRateLimit != nil {
		rateLimitFilter, err := buildTCPGlobalRateLimitFilter(fb.tcpGlobalRateLimit, fb.statsPrefix)
		if err != nil {
			return nil, err
		}
		filters = append(filters, rateLimitFilter)
	}

	// HTTP connection manager filter
	if fb.hcmBuilder != nil {
		hcmFilter, err := fb.hcmBuilder.Build()
		if err != nil {
			return nil, err
		}
		filters = append(filters, hcmFilter)
	}

	if fb.tcpProxyBuilder != nil {
		tcpProxyFilter, err := fb.tcpProxyBuilder.Build()
		if err != nil {
			return nil, err
		}
		filters = append(filters, tcpProxyFilter)
	}

	return filters, nil
}

// HTTPConnManagerBuilder returns the HTTP Connection Manager builder instance
func HTTPConnManagerBuilder() *httpConnManagerBuilder { //nolint: revive // unexported-return
	return &httpConnManagerBuilder{}
}

// StatsPrefix sets the stats prefix on the builder
func (hb *httpConnManagerBuilder) StatsPrefix(statsPrefix string) *httpConnManagerBuilder {
	hb.statsPrefix = statsPrefix
	return hb
}

// RouteConfigName sets the route config name on the builder
func (hb *httpConnManagerBuilder) RouteConfigName(name string) *httpConnManagerBuilder {
	hb.routeConfigName = name
	return hb
}

// defaultFilters sets the default HTTP filters on the builder
func (hb *httpConnManagerBuilder) defaultFilters() []*xds_hcm.HttpFilter {
	return []*xds_hcm.HttpFilter{
		{
			// HTTP RBAC filter - required to perform HTTP based RBAC per route
			Name: envoy.HTTPRBACFilterName,
			ConfigType: &xds_hcm.HttpFilter_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: envoy.HTTPRBACFilterTypeURL,
				},
			},
		},
		{
			// HTTP local rate limit filter - required to perform local rate limiting
			Name: envoy.HTTPLocalRateLimitFilterName,
			ConfigType: &xds_hcm.HttpFilter_TypedConfig{
				TypedConfig: protobuf.MustMarshalAny(
					&xds_http_local_ratelimit.LocalRateLimit{
						StatPrefix: hb.statsPrefix,
						// Since no token bucket is defined here, the filter is disabled
						// at the listener level. For HTTP traffic, the rate limiting
						// config is applied at the VirtualHost/Route level.
						// Ref: https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/local_rate_limit_filter#using-rate-limit-descriptors-for-local-rate-limiting
					},
				),
			},
		},
	}
}

// AddFilter adds the given HttpFilter to the builder's filter list.
// It ensures the HTTP router filter is always added only once, which
// is a requirement in Envoy.
func (hb *httpConnManagerBuilder) AddFilter(filter *xds_hcm.HttpFilter) *httpConnManagerBuilder {
	if filter == nil {
		return hb
	}

	if filter.Name == envoy.HTTPRouterFilterName {
		hb.routerFilter = filter
		return hb
	}

	hb.filters = append(hb.filters, filter)

	return hb
}

// LocalReplyConfig sets the given LocalReplyConfig on the builder
func (hb *httpConnManagerBuilder) LocalReplyConfig(config *xds_hcm.LocalReplyConfig) *httpConnManagerBuilder {
	hb.localReplyConfig = config
	return hb
}

// Tracing sets the Tracing config on the builder
func (hb *httpConnManagerBuilder) Tracing(config *xds_hcm.HttpConnectionManager_Tracing) *httpConnManagerBuilder {
	hb.tracing = config
	return hb
}

func (hb *httpConnManagerBuilder) HTTPGlobalRateLimit(rl *policyv1alpha1.HTTPGlobalRateLimitSpec) *httpConnManagerBuilder {
	hb.httpGlobalRateLimit = rl
	return hb
}

// Build builds the HttpConnectionManager filter from the builder config
func (hb *httpConnManagerBuilder) Build() (*xds_listener.Filter, error) {
	httpFilters := hb.defaultFilters()
	httpFilters = append(httpFilters, hb.filters...)

	// NOTE: router filter must always be the last filter in the list
	if hb.routerFilter == nil {
		// Router filter - required to perform HTTP connection management
		hb.routerFilter = &xds_hcm.HttpFilter{
			Name: envoy.HTTPRouterFilterName,
			ConfigType: &xds_hcm.HttpFilter_TypedConfig{
				TypedConfig: &any.Any{
					TypeUrl: envoy.HTTPRouterFilterTypeURL,
				},
			},
		}
	}
	httpFilters = append(httpFilters, hb.routerFilter)

	connManager := &xds_hcm.HttpConnectionManager{
		StatPrefix:  hb.statsPrefix,
		CodecType:   xds_hcm.HttpConnectionManager_AUTO,
		HttpFilters: httpFilters,
		RouteSpecifier: &xds_hcm.HttpConnectionManager_Rds{
			Rds: &xds_hcm.Rds{
				ConfigSource:    envoy.GetADSConfigSource(),
				RouteConfigName: hb.routeConfigName,
			},
		},
		AccessLog: envoy.GetAccessLog(),
		UpgradeConfigs: []*xds_hcm.HttpConnectionManager_UpgradeConfig{
			{
				UpgradeType: websocketUpgradeType,
			},
		},
	}

	if hb.tracing != nil {
		connManager.GenerateRequestId = &wrappers.BoolValue{
			Value: true,
		}
		connManager.Tracing = hb.tracing
	}

	marshalled, err := anypb.New(connManager)
	if err != nil {
		return nil, err
	}

	return &xds_listener.Filter{
		Name:       envoy.HTTPConnectionManagerFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalled},
	}, nil
}

// TCPProxyBuilder returns a TCP proxy builder instance
func TCPProxyBuilder() *tcpProxyBuilder { //nolint: revive // unexported-return
	return &tcpProxyBuilder{}
}

// StatsPrefix sets the stats prefix to use for the TCP proxy filter
func (tb *tcpProxyBuilder) StatsPrefix(statsPrefix string) *tcpProxyBuilder {
	tb.statsPrefix = statsPrefix
	return tb
}

// Cluster sets the cluster to use for the TCP proxy filter
func (tb *tcpProxyBuilder) Cluster(cluster string) *tcpProxyBuilder {
	tb.cluster = cluster
	return tb
}

func (tb *tcpProxyBuilder) WeightedClusters(wc []service.WeightedCluster) *tcpProxyBuilder {
	tb.weightedClusters = wc
	return tb
}

// Build builds the TCP proxy filter
func (tb *tcpProxyBuilder) Build() (*xds_listener.Filter, error) {
	if tb.cluster != "" && len(tb.weightedClusters) > 0 {
		return nil, errors.New("TcpProxy: only one of cluster or weightedClusters can be specified")
	} else if tb.cluster == "" && len(tb.weightedClusters) == 0 {
		return nil, errors.New("TcpProxy: at least one of cluster or weightedCluster must be specified")
	}

	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix: tb.statsPrefix,
	}

	if tb.cluster != "" {
		tcpProxy.ClusterSpecifier = &xds_tcp_proxy.TcpProxy_Cluster{Cluster: tb.cluster}
	} else if len(tb.weightedClusters) == 1 {
		tcpProxy.ClusterSpecifier = &xds_tcp_proxy.TcpProxy_Cluster{Cluster: tb.weightedClusters[0].ClusterName.String()}
	} else {
		var clusterWeights []*xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight
		for _, cluster := range tb.weightedClusters {
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
		return nil, err
	}

	marshalled := &xds_listener.Filter{
		Name:       envoy.TCPProxyFilterName,
		ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledTCPProxy},
	}

	return marshalled, nil
}
