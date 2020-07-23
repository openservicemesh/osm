package lds

import (
	"net"
	"strconv"
	"strings"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoy_hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

const (
	outboundMeshFilterChainName   = "outbound-mesh-filter-chain"
	outboundEgressFilterChainName = "outbound-egress-filter-chain"
)

func buildOutboundListener(connManager *envoy_hcm.HttpConnectionManager, cfg configurator.Configurator) (*xds.Listener, error) {
	marshalledConnManager, err := ptypes.MarshalAny(connManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HttpConnectionManager object")
		return nil, err
	}

	outboundListener := &xds.Listener{
		Name:             outboundListenerName,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort),
		TrafficDirection: envoy_api_v2_core.TrafficDirection_OUTBOUND,
		FilterChains: []*listener.FilterChain{
			{
				Name: outboundMeshFilterChainName,
				Filters: []*listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
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

func updateOutboundListenerForEgress(outboundListener *xds.Listener, cfg configurator.Configurator) error {
	// When egress, the in-mesh CIDR is used to distinguish in-mesh traffic
	meshCIDRRanges := cfg.GetMeshCIDRRanges()
	if len(meshCIDRRanges) == 0 {
		log.Error().Err(errInvalidCIDRRange).Msg("Mesh CIDR ranges unspecified, required when egress is enabled")
		return errInvalidCIDRRange
	}

	var prefixRanges []*envoy_api_v2_core.CidrRange

	for _, cidr := range meshCIDRRanges {
		cidrRange, err := getCIDRRange(cidr)
		if err != nil {
			log.Error().Err(err).Msgf("Error parsing CIDR: %s", cidr)
			return err
		}
		prefixRanges = append(prefixRanges, cidrRange)
	}
	outboundListener.FilterChains[0].FilterChainMatch = &listener.FilterChainMatch{
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

func buildInboundListener() *xds.Listener {
	return &xds.Listener{
		Name:             inboundListenerName,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort),
		TrafficDirection: envoy_api_v2_core.TrafficDirection_INBOUND,
		FilterChains:     []*listener.FilterChain{},
		ListenerFilters: []*listener.ListenerFilter{
			{
				Name: wellknown.TlsInspector,
			},
		},
	}
}

func buildPrometheusListener(connManager *envoy_hcm.HttpConnectionManager) (*xds.Listener, error) {
	marshalledConnManager, err := ptypes.MarshalAny(connManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling HttpConnectionManager object")
		return nil, err
	}

	listener := &xds.Listener{
		Name:             prometheusListenerName,
		TrafficDirection: envoy_api_v2_core.TrafficDirection_INBOUND,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyPrometheusInboundListenerPort),
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: marshalledConnManager,
						},
					},
				},
			},
		},
	}
	return listener, nil
}

func buildEgressFilterChain() (*listener.FilterChain, error) {
	tcpProxy := &tcp_proxy.TcpProxy{
		StatPrefix:       envoy.OutboundPassthroughCluster,
		ClusterSpecifier: &tcp_proxy.TcpProxy_Cluster{Cluster: envoy.OutboundPassthroughCluster},
	}
	marshalledTCPProxy, err := envoy.MessageToAny(tcpProxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling TcpProxy object for egress HTTPS filter chain")
		return nil, err
	}

	return &listener.FilterChain{
		Name: outboundEgressFilterChainName,
		Filters: []*listener.Filter{
			{
				Name:       wellknown.TCPProxy,
				ConfigType: &listener.Filter_TypedConfig{TypedConfig: marshalledTCPProxy},
			},
		},
	}, nil
}

func buildEgressListenerFilters() []*listener.ListenerFilter {
	return []*listener.ListenerFilter{
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

func getCIDRRange(cidr string) (*envoy_api_v2_core.CidrRange, error) {
	addr, prefix, err := parseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	cidrRange := &envoy_api_v2_core.CidrRange{
		AddressPrefix: addr,
		PrefixLen: &wrappers.UInt32Value{
			Value: prefix,
		},
	}

	return cidrRange, nil
}
