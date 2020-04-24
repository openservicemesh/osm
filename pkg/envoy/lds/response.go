package lds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

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

	meshFilterChain, err := getInMeshFilterChain(proxyServiceName, catalog, serverConnManager)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to construct in-mesh filter chain for proxy %s", proxy.GetCommonName())
	}
	if meshFilterChain != nil {
		serverListener.FilterChains = append(serverListener.FilterChains, meshFilterChain)
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

func getInMeshFilterChain(proxyServiceName endpoint.NamespacedService, mc catalog.MeshCataloger, filterConfig *any.Any) (*listener.FilterChain, error) {
	serverNames, err := getFilterChainMatchServerNames(proxyServiceName, mc)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get client server names for proxy %s", proxyServiceName)
		return nil, err
	}
	if len(serverNames) == 0 {
		log.Debug().Msgf("No mesh filter chain to apply")
		return nil, nil
	}

	filterChain := &listener.FilterChain{
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
			ServerNames:       serverNames,
			TransportProtocol: "tls",
		},
		TransportSocket: &envoy_api_v2_core.TransportSocket{
			Name: envoy.TransportSocketTLS,
			ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
				TypedConfig: envoy.GetDownstreamTLSContext(proxyServiceName),
			},
		},
	}

	return filterChain, nil
}
