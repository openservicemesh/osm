package lds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/smi"
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
	outboundListener, err := newOutboundListener(cfg)
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

	inboundListener := newInboundListener()
	meshFilterChain, err := getInboundInMeshFilterChain(proxyServiceName, cfg)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to construct in-mesh filter chain for proxy %s", proxy.GetCommonName())
	}
	if meshFilterChain != nil {
		inboundListener.FilterChains = append(inboundListener.FilterChains, meshFilterChain)
	}

	// Apply an ingress filter chain if there are any ingress routes
	ingressRoutesPerHost, err := catalog.GetIngressRoutesPerHost(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting ingress routes per host for service %s", proxyServiceName)
		return nil, err
	}
	if len(ingressRoutesPerHost) > 0 {
		log.Info().Msgf("Found an ingress resource for service %s, applying necessary filters", proxyServiceName)
		// This proxy is fronting a service that is a backend for an ingress, add a FilterChain for it
		ingressFilterChains := getIngressFilterChains(proxyServiceName, cfg)
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
