package lds

import (
	"context"
	"net"
	"strconv"
	"strings"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/route"
	"github.com/open-service-mesh/osm/pkg/smi"
)

const (
	inboundListenerName    = "inbound_listener"
	outboundListenerName   = "outbound_listener"
	prometheusListenerName = "inbound_prometheus_listener"
)

// NewResponse creates a new Listener Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	log.Info().Msgf("Composing listener Discovery Response for proxy: %s", proxy.GetCommonName())
	proxyServiceName := proxy.GetService()
	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeLDS),
	}

	clientConnManager, err := ptypes.MarshalAny(getHTTPConnectionManager(route.OutboundRouteConfig))
	if err != nil {
		log.Error().Err(err).Msgf("Could not construct FilterChain")
		return nil, err
	}

	clientListener := &xds.Listener{
		Name:    outboundListenerName,
		Address: envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyOutboundListenerPort),
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: clientConnManager,
						},
					},
				},
			},
		},
	}
	log.Info().Msgf("Created listener %s for proxy %s for service %s: %+v", outboundListenerName, proxy.GetCommonName(), proxy.GetService(), clientListener)

	serverConnManager, err := ptypes.MarshalAny(getHTTPConnectionManager(route.InboundRouteConfig))
	if err != nil {
		log.Error().Err(err).Msg("Could not construct inbound listener FilterChain")
		return nil, err
	}

	serverListener := &xds.Listener{
		Name:         inboundListenerName,
		Address:      envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyInboundListenerPort),
		FilterChains: []*listener.FilterChain{},
	}

	meshFilterChains, applyMeshFilterChain, err := getInMeshFilterChains(proxyServiceName, catalog, serverConnManager)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to construct in-mesh filter chain for proxy %s", proxy.GetCommonName())
	}
	if applyMeshFilterChain {
		serverListener.FilterChains = append(serverListener.FilterChains, meshFilterChains...)
	}

	isIngress, err := catalog.IsIngressService(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking service %s for ingress", proxyServiceName)
		return nil, err
	}
	if isIngress {
		log.Info().Msgf("Found an ingress resource for service %s, applying necessary filters", proxyServiceName)
		// This proxy is fronting a service that is a backend for an ingress, add a FilterChain for it
		ingressFilterChain := &listener.FilterChain{
			FilterChainMatch: &listener.FilterChainMatch{
				// TODO(shashank): Check if the match below is needed for ingress
				SourcePrefixRanges: []*envoy_api_v2_core.CidrRange{
					convertIPAddressToCidr("0.0.0.0/0"),
				},
			},
			Filters: []*listener.Filter{
				{
					Name: wellknown.HTTPConnectionManager,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: serverConnManager,
					},
				},
			},
		}
		serverListener.FilterChains = append(serverListener.FilterChains, ingressFilterChain)
	}

	log.Info().Msgf("Created listener %s for proxy %s for service %s: %+v", inboundListenerName, proxy.GetCommonName(), proxy.GetService(), serverListener)
	prometheusConnManager, err := ptypes.MarshalAny(getPrometheusConnectionManager(prometheusListenerName, constants.PrometheusScrapePath, constants.EnvoyAdminCluster))
	if err != nil {
		log.Error().Err(err).Msgf("Could not construct prometheus listener connection manager")
		return nil, err
	}
	prometheusListener := &xds.Listener{
		Name:             prometheusListenerName,
		TrafficDirection: envoy_api_v2_core.TrafficDirection_INBOUND,
		Address:          envoy.GetAddress(constants.WildcardIPAddr, constants.EnvoyPrometheusInboundListenerPort),
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: prometheusConnManager,
						},
					},
				},
			},
		},
	}
	log.Info().Msgf("Created listener %s for proxy %s for service %s: %+v", prometheusListenerName, proxy.GetCommonName(), proxy.GetService(), prometheusListener)
	marshalledPrometheus, err := ptypes.MarshalAny(prometheusListener)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to marshal inbound listener for proxy %s", proxy.GetCommonName())
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledPrometheus)

	marshalledOutbound, err := ptypes.MarshalAny(clientListener)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to marshal outbound listener for proxy %s", proxy.GetCommonName())
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledOutbound)

	if len(serverListener.FilterChains) > 0 {
		// Inbound filter chains can be empty if the there both ingress and in-mesh policies are not configued.
		// Configuring a listener without a filter chain is an error.
		marshalledInbound, err := ptypes.MarshalAny(serverListener)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal inbound listener for proxy %s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledInbound)
	}
	return resp, nil
}

func getFilterChainMatchServerNames(proxyServiceName endpoint.NamespacedService, catalog catalog.MeshCataloger) ([]string, error) {
	serverNamesMap := make(map[string]interface{})
	var serverNames []string

	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msg("Failed listing traffic routes")
		return nil, err
	}

	for _, trafficPolicies := range allTrafficPolicies {
		isDestinationService := envoy.Contains(proxyServiceName, trafficPolicies.Destination.Services)
		if isDestinationService {
			for _, source := range trafficPolicies.Source.Services {
				if _, server := serverNamesMap[source.String()]; !server {
					serverNamesMap[source.String()] = nil
					serverNames = append(serverNames, source.String())
				}
			}

		}
	}
	return serverNames, nil
}

func getInMeshFilterChains(proxyServiceName endpoint.NamespacedService, mc catalog.MeshCataloger, filterConfig *any.Any) ([]*listener.FilterChain, bool, error) {
	allTrafficPolicies, err := mc.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to list traffic policies for proxy %s", proxyServiceName)
		return nil, false, err
	}

	serverNames, err := getFilterChainMatchServerNames(proxyServiceName, mc)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get client server names for proxy %s", proxyServiceName)
		return nil, false, err
	}

	var filterChains []*listener.FilterChain
	for _, policy := range allTrafficPolicies {
		isDestinationService := envoy.Contains(proxyServiceName, policy.Destination.Services)
		if isDestinationService {
			sourceServices := policy.Source.Services
			// For each source, build a filter chain match
			for _, srcService := range sourceServices {
				// Get the endpoint for this source
				serviceEndpoints, _ := mc.ListEndpointsForService(endpoint.ServiceName(srcService.String()))
				for _, endpoint := range serviceEndpoints {
					// Build a filter chain for this endpoint
					cidr := convertIPAddressToCidr(endpoint.IP.String())
					endpointFilterChain := &listener.FilterChain{
						Filters: []*listener.Filter{
							{
								Name: wellknown.HTTPConnectionManager,
								ConfigType: &listener.Filter_TypedConfig{
									TypedConfig: filterConfig,
								},
							},
						},
						// The FilterChainMatch uses SNI from mTLS to match against the provided list of ServerNames.
						// This ensures only clients authorized to talk to this listener are permitted to.
						FilterChainMatch: &listener.FilterChainMatch{
							ServerNames: serverNames,
							SourcePrefixRanges: []*envoy_api_v2_core.CidrRange{
								cidr,
							},
						},

						TransportSocket: &envoy_api_v2_core.TransportSocket{
							Name: envoy.TransportSocketTLS,
							ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
								TypedConfig: envoy.GetDownstreamTLSContext(proxyServiceName),
							},
						},
					}
					filterChains = append(filterChains, endpointFilterChain)
				}
			}
		}
	}

	applyFilterChain := len(filterChains) > 0
	return filterChains, applyFilterChain, nil
}

func getMaxCidrPrefixLen(addr string) uint32 {
	ip := net.ParseIP(addr)
	if ip.To4() == nil {
		// IPv6 address
		return 128
	}
	// IPv4 address
	return 32
}

func convertIPAddressToCidr(addr string) *envoy_api_v2_core.CidrRange {
	if len(addr) == 0 {
		return nil
	}
	cidr := &envoy_api_v2_core.CidrRange{
		AddressPrefix: addr,
		PrefixLen: &wrappers.UInt32Value{
			Value: getMaxCidrPrefixLen(addr),
		},
	}
	if strings.Contains(addr, "/") {
		chunks := strings.Split(addr, "/")
		cidr.AddressPrefix = chunks[0]
		prefix, _ := strconv.Atoi(chunks[1])
		cidr.PrefixLen.Value = uint32(prefix)
	}
	return cidr
}
