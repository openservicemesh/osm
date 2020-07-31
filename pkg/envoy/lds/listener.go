package lds

import (
	"net"
	"strconv"
	"strings"

	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	xds_hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/route"
)

const (
	outboundMeshFilterChainName   = "outbound-mesh-filter-chain"
	outboundEgressFilterChainName = "outbound-egress-filter-chain"
)

func newOutboundListener(cfg configurator.Configurator) (*xds_listener.Listener, error) {
	connManager := getHTTPConnectionManager(route.OutboundRouteConfigName, cfg)

	marshalledConnManager, err := ptypes.MarshalAny(connManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HttpConnectionManager object")
		return nil, err
	}

	outboundListener := &xds_listener.Listener{
		Name:             outboundListenerName,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort),
		TrafficDirection: xds_core.TrafficDirection_OUTBOUND,
		FilterChains: []*xds_listener.FilterChain{
			{
				Name: outboundMeshFilterChainName,
				Filters: []*xds_listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &xds_listener.Filter_TypedConfig{
							TypedConfig: marshalledConnManager,
						},
					},
				},
			},
		},
	}

	if cfg.IsEgressEnabled() {
		err := updateOutboundListenerForEgress(outboundListener, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Error building egress config for outbound listener")
			// An error in egress config should not disrupt in-mesh traffic, so only log an error
		}
	}

	return outboundListener, nil
}

func updateOutboundListenerForEgress(outboundListener *xds_listener.Listener, cfg configurator.Configurator) error {
	// When egress, the in-mesh CIDR is used to distinguish in-mesh traffic
	meshCIDRRanges := cfg.GetMeshCIDRRanges()
	if len(meshCIDRRanges) == 0 {
		log.Error().Err(errInvalidCIDRRange).Msg("Mesh CIDR ranges unspecified, required when egress is enabled")
		return errInvalidCIDRRange
	}

	var prefixRanges []*xds_core.CidrRange

	for _, cidr := range meshCIDRRanges {
		cidrRange, err := getCIDRRange(cidr)
		if err != nil {
			log.Error().Err(err).Msgf("Error parsing CIDR: %s", cidr)
			return err
		}
		prefixRanges = append(prefixRanges, cidrRange)
	}
	outboundListener.FilterChains[0].FilterChainMatch = &xds_listener.FilterChainMatch{
		PrefixRanges: prefixRanges,
	}

	// With egress, a filter chain to match TLS traffic is added to the outbound listener.
	// In-mesh traffic will always be HTTP so this filter chain will not match for in-mesh.
	// HTTPS egress traffic will match this filter chain and will be proxied to its original
	// destination.
	egressFilterChain, err := buildEgressFilterChain()
	if err != nil {
		return err
	}
	outboundListener.FilterChains = append(outboundListener.FilterChains, egressFilterChain)
	listenerFilters := buildEgressListenerFilters()
	outboundListener.ListenerFilters = append(outboundListener.ListenerFilters, listenerFilters...)

	return nil
}

func newInboundListener() *xds_listener.Listener {
	return &xds_listener.Listener{
		Name:             inboundListenerName,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort),
		TrafficDirection: xds_core.TrafficDirection_INBOUND,
		FilterChains:     []*xds_listener.FilterChain{},
		ListenerFilters: []*xds_listener.ListenerFilter{
			{
				Name: wellknown.TlsInspector,
			},
		},
	}
}

func buildPrometheusListener(connManager *xds_hcm.HttpConnectionManager) (*xds_listener.Listener, error) {
	marshalledConnManager, err := ptypes.MarshalAny(connManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HttpConnectionManager object")
		return nil, err
	}

	return &xds_listener.Listener{
		Name:             prometheusListenerName,
		TrafficDirection: xds_core.TrafficDirection_INBOUND,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyPrometheusInboundListenerPort),
		FilterChains: []*xds_listener.FilterChain{
			{
				Filters: []*xds_listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &xds_listener.Filter_TypedConfig{
							TypedConfig: marshalledConnManager,
						},
					},
				},
			},
		},
	}, nil
}

func buildEgressFilterChain() (*xds_listener.FilterChain, error) {
	tcpProxy := &xds_tcp_proxy.TcpProxy{
		StatPrefix:       envoy.OutboundPassthroughCluster,
		ClusterSpecifier: &xds_tcp_proxy.TcpProxy_Cluster{Cluster: envoy.OutboundPassthroughCluster},
	}
	marshalledTCPProxy, err := envoy.MessageToAny(tcpProxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling TcpProxy object for egress HTTPS filter chain")
		return nil, err
	}

	return &xds_listener.FilterChain{
		Name: outboundEgressFilterChainName,
		Filters: []*xds_listener.Filter{
			{
				Name:       wellknown.TCPProxy,
				ConfigType: &xds_listener.Filter_TypedConfig{TypedConfig: marshalledTCPProxy},
			},
		},
	}, nil
}

func buildEgressListenerFilters() []*xds_listener.ListenerFilter {
	return []*xds_listener.ListenerFilter{
		{
			// The OriginalDestination ListenerFilter is used to redirect traffic
			// to its original destination.
			Name: wellknown.OriginalDestination,
		},
		{
			// The TlsInspector ListenerFilter is used to examine the transport protocol
			Name: wellknown.TlsInspector,
		},
	}
}

func parseCIDR(cidr string) (string, uint32, error) {
	var addr string

	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return addr, 0, err
	}
	chunks := strings.Split(cidr, "/")
	addr = chunks[0]
	prefix, err := strconv.Atoi(chunks[1])
	if err != nil {
		return addr, 0, err
	}

	return addr, uint32(prefix), nil
}

func getCIDRRange(cidr string) (*xds_core.CidrRange, error) {
	addr, prefix, err := parseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	cidrRange := &xds_core.CidrRange{
		AddressPrefix: addr,
		PrefixLen: &wrappers.UInt32Value{
			Value: prefix,
		},
	}

	return cidrRange, nil
}
