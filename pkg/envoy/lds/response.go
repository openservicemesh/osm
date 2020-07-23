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
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/route"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
)

const (
	inboundListenerName    = "inbound_listener"
	outboundListenerName   = "outbound_listener"
	prometheusListenerName = "inbound_prometheus_listener"
)

// NewResponse creates a new Listener Discovery Response.
// The response build 3 Listeners:
// 1. Inbound listener to handle incoming traffic
// 2. Outbound listener to handle outgoing traffic
// 3. Prometheus listener for metrics
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest, cfg configurator.Configurator) (*xds.DiscoveryResponse, error) {
	svc, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	proxyServiceName := *svc

	resp := &xds.DiscoveryResponse{
		TypeUrl: string(envoy.TypeLDS),
	}

	// Build the outbound listener config
	outboundConnManager := getHTTPConnectionManager(route.OutboundRouteConfig)
	outboundListener, err := buildOutboundListener(outboundConnManager, cfg.IsEgressEnabled())
	if err != nil {
		log.Error().Err(err).Msgf("Error building outbound listener config for proxy %s", proxyServiceName)
		return nil, err
	}
	marshalledOutbound, err := ptypes.MarshalAny(outboundListener)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to marshal outbound listener config for proxy %s", proxyServiceName)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledOutbound)

	// Build the inbound listener config
	inboundConnManager := getHTTPConnectionManager(route.InboundRouteConfig)
	marshalledInboundConnManager, err := ptypes.MarshalAny(inboundConnManager)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling inbound HttpConnectionManager object for proxy %s", proxyServiceName)
		return nil, err
	}

	inboundListener := buildInboundListener()
	meshFilterChain, err := getInboundInMeshFilterChain(proxyServiceName, catalog, marshalledInboundConnManager)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to construct in-mesh filter chain for proxy %s", proxy.GetCommonName())
	}
	if meshFilterChain != nil {
		inboundListener.FilterChains = append(inboundListener.FilterChains, meshFilterChain)
	}

	// Apply a filter chain for ingress if applicable
	isIngress, err := catalog.IsIngressService(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Error checking service %s for ingress", proxyServiceName)
		return nil, err
	}
	if isIngress {
		log.Info().Msgf("Found an ingress resource for service %s, applying necessary filters", proxyServiceName)
		// This proxy is fronting a service that is a backend for an ingress, add a FilterChain for it
		ingressFilterChains, err := getInboundIngressFilterChains(proxyServiceName, marshalledInboundConnManager)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to construct ingress filter chain for proxy %s", proxyServiceName)
		}
		inboundListener.FilterChains = append(inboundListener.FilterChains, ingressFilterChains...)
	}

	if len(inboundListener.FilterChains) > 0 {
		// Inbound filter chains can be empty if the there both ingress and in-mesh policies are not configued.
		// Configuring a listener without a filter chain is an error.
		marshalledInbound, err := ptypes.MarshalAny(inboundListener)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling inbound listener config for proxy %s", proxyServiceName)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledInbound)
	}

	if cfg.IsPrometheusScrapingEnabled() {
		// Build Prometheus listener config
		prometheusConnManager := getPrometheusConnectionManager(prometheusListenerName, constants.PrometheusScrapePath, constants.EnvoyMetricsCluster)
		prometheusListener, err := buildPrometheusListener(prometheusConnManager)
		if err != nil {
			log.Error().Err(err).Msgf("Error building Prometheus listener config for proxy %s", proxyServiceName)
			return nil, err
		}
		marshalledPrometheus, err := ptypes.MarshalAny(prometheusListener)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling Prometheus listener config for proxy %s", proxyServiceName)
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledPrometheus)
	}

	return resp, nil
}

func getInboundInMeshFilterChain(proxyServiceName service.NamespacedService, mc catalog.MeshCataloger, filterConfig *any.Any) (*listener.FilterChain, error) {
	marshalledDownstreamTLSContext, err := envoy.MessageToAny(envoy.GetDownstreamTLSContext(proxyServiceName, true /* mTLS */))
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling DownstreamTLSContext object for proxy %s", proxyServiceName)
		return nil, err
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

		// Apply this filter chain only to requests where the auth.UpstreamTlsContext.Sni matches
		// one from the list of ServerNames provided below.
		// This field is configured by the GetDownstreamTLSContext() function.
		// This is not a field obtained from the mTLS Certificate.
		FilterChainMatch: &listener.FilterChainMatch{
			ServerNames:          []string{proxyServiceName.GetCommonName().String()},
			TransportProtocol:    envoy.TransportProtocolTLS,
			ApplicationProtocols: envoy.ALPNInMesh, // in-mesh proxies will advertise this, set in UpstreamTlsContext
		},

		TransportSocket: &envoy_api_v2_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledDownstreamTLSContext,
			},
		},
	}

	return filterChain, nil
}

func getInboundIngressFilterChains(proxyServiceName service.NamespacedService, filterConfig *any.Any) ([]*listener.FilterChain, error) {
	marshalledDownstreamTLSContext, err := envoy.MessageToAny(envoy.GetDownstreamTLSContext(proxyServiceName, false /* TLS */))
	if err != nil {
		log.Error().Err(err).Msgf("Error marshalling DownstreamTLSContext object for proxy %s", proxyServiceName)
		return nil, err
	}
	return []*listener.FilterChain{
		{
			// Filter chain with SNI matching enabled for clients that set the SNI
			FilterChainMatch: &listener.FilterChainMatch{
				TransportProtocol: envoy.TransportProtocolTLS,
				ServerNames:       []string{proxyServiceName.GetCommonName().String()},
			},
			TransportSocket: &envoy_api_v2_core.TransportSocket{
				Name: wellknown.TransportSocketTls,
				ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
					TypedConfig: marshalledDownstreamTLSContext,
				},
			},
			Filters: []*listener.Filter{
				{
					Name: wellknown.HTTPConnectionManager,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: filterConfig,
					},
				},
			},
		},
		{
			// Filter chain without SNI matching enabled for clients that don't set the SNI
			FilterChainMatch: &listener.FilterChainMatch{
				TransportProtocol: envoy.TransportProtocolTLS,
			},
			TransportSocket: &envoy_api_v2_core.TransportSocket{
				Name: wellknown.TransportSocketTls,
				ConfigType: &envoy_api_v2_core.TransportSocket_TypedConfig{
					TypedConfig: marshalledDownstreamTLSContext,
				},
			},
			Filters: []*listener.Filter{
				{
					Name: wellknown.HTTPConnectionManager,
					ConfigType: &listener.Filter_TypedConfig{
						TypedConfig: filterConfig,
					},
				},
			},
		},
	}, nil
}
