package lds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
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
func NewResponse(catalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator) (*xds_discovery.DiscoveryResponse, error) {
	svcList, err := catalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	// Github Issue #1575
	proxyServiceName := svcList[0]

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeLDS),
	}

	// --- OUTBOUND -------------------
	outboundListener, err := newOutboundListener(catalog, cfg, svcList)
	if err != nil {
		log.Error().Err(err).Msgf("Error making outbound listener config for proxy %s", proxyServiceName)
	} else {
		if outboundListener == nil {
			log.Debug().Msgf("Not programming Outbound listener for proxy %s", proxyServiceName)
		} else {
			if marshalledOutbound, err := ptypes.MarshalAny(outboundListener); err != nil {
				log.Error().Err(err).Msgf("Failed to marshal outbound listener config for proxy %s", proxyServiceName)
			} else {
				resp.Resources = append(resp.Resources, marshalledOutbound)
			}
		}
	}

	// --- INBOUND -------------------
	inboundListener := newInboundListener()
	if meshFilterChain, err := getInboundInMeshFilterChain(proxyServiceName, cfg); err != nil {
		log.Error().Err(err).Msgf("Error making in-mesh filter chain for proxy %s", proxy.GetCommonName())
	} else if meshFilterChain != nil {
		inboundListener.FilterChains = append(inboundListener.FilterChains, meshFilterChain)
	}

	// --- INGRESS -------------------
	// Apply an ingress filter chain if there are any ingress routes
	if ingressRoutesPerHost, err := catalog.GetIngressRoutesPerHost(proxyServiceName); err != nil {
		log.Error().Err(err).Msgf("Error getting ingress routes per host for service %s", proxyServiceName)
	} else {
		thereAreIngressRoutes := len(ingressRoutesPerHost) > 0

		if thereAreIngressRoutes {
			log.Info().Msgf("Found k8s Ingress for MeshService %s, applying necessary filters", proxyServiceName)
			// This proxy is fronting a service that is a backend for an ingress, add a FilterChain for it
			ingressFilterChains := getIngressFilterChains(proxyServiceName, cfg)
			inboundListener.FilterChains = append(inboundListener.FilterChains, ingressFilterChains...)
		} else {
			log.Trace().Msgf("There is no k8s Ingress for service %s", proxyServiceName)
		}
	}

	if len(inboundListener.FilterChains) > 0 {
		// Inbound filter chains can be empty if the there both ingress and in-mesh policies are not configued.
		// Configuring a listener without a filter chain is an error.
		if marshalledInbound, err := ptypes.MarshalAny(inboundListener); err != nil {
			log.Error().Err(err).Msgf("Error marshalling inbound listener config for proxy %s", proxyServiceName)
		} else {
			resp.Resources = append(resp.Resources, marshalledInbound)
		}
	}

	if cfg.IsPrometheusScrapingEnabled() {
		// Build Prometheus listener config
		prometheusConnManager := getPrometheusConnectionManager(prometheusListenerName, constants.PrometheusScrapePath, constants.EnvoyMetricsCluster)
		if prometheusListener, err := buildPrometheusListener(prometheusConnManager); err != nil {
			log.Error().Err(err).Msgf("Error building Prometheus listener config for proxy %s", proxyServiceName)
		} else {
			if marshalledPrometheus, err := ptypes.MarshalAny(prometheusListener); err != nil {
				log.Error().Err(err).Msgf("Error marshalling Prometheus listener config for proxy %s", proxyServiceName)
			} else {
				resp.Resources = append(resp.Resources, marshalledPrometheus)
			}
		}
	}

	return resp, nil
}
