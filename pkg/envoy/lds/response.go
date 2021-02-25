package lds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/featureflags"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Listener Discovery Response.
// The response build 3 Listeners:
// 1. Inbound listener to handle incoming traffic
// 2. Outbound listener to handle outgoing traffic
// 3. Prometheus listener for metrics
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	svcList, err := meshCatalog.GetServicesFromEnvoyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy certificate SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}

	svcAccount, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving ServiceAccount for Envoy with certificate with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeLDS),
	}

	var statsHeaders map[string]string
	if featureflags.IsWASMStatsEnabled() {
		statsHeaders = proxy.StatsHeaders()
	}

	lb := newListenerBuilder(meshCatalog, svcAccount, cfg, statsHeaders)

	// --- OUTBOUND -------------------
	outboundListener, err := lb.newOutboundListener()
	if err != nil {
		log.Error().Err(err).Msgf("Error making outbound listener config for proxy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
	} else {
		if outboundListener == nil {
			// This check is important to prevent attempting to configure a listener without a filter chain which
			// otherwise results in an error.
			log.Debug().Msgf("Not programming Outbound listener for proxy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
				proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		} else {
			if marshalledOutbound, err := ptypes.MarshalAny(outboundListener); err != nil {
				log.Error().Err(err).Msgf("Failed to marshal outbound listener config for proxy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
					proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
			} else {
				resp.Resources = append(resp.Resources, marshalledOutbound)
			}
		}
	}

	// --- INBOUND -------------------
	inboundListener := newInboundListener()
	// Create inbound filter chains per service behind proxy
	for _, proxyService := range svcList {
		// Create in-mesh filter chains
		inboundSvcFilterChains := lb.getInboundMeshFilterChains(proxyService)
		inboundListener.FilterChains = append(inboundListener.FilterChains, inboundSvcFilterChains...)

		// Create ingress filter chains if there are any ingress routes
		if ingressInboundPolicies, err := meshCatalog.GetIngressPoliciesForService(proxyService); err != nil {
			log.Error().Err(err).Msgf("Error getting ingress inbound traffic policies for service %s", proxyService)
		} else {
			thereAreIngressRoutes := len(ingressInboundPolicies) > 0
			if thereAreIngressRoutes {
				log.Info().Msgf("Found k8s Ingress for MeshService %s, applying necessary filters", proxyService)
				// This proxy is fronting a service that is a backend for an ingress, add a FilterChain for it
				ingressFilterChains := lb.getIngressFilterChains(proxyService)
				inboundListener.FilterChains = append(inboundListener.FilterChains, ingressFilterChains...)
			} else {
				log.Trace().Msgf("There is no k8s Ingress for service %s", proxyService)
			}
		}
	}

	if len(inboundListener.FilterChains) > 0 {
		// Inbound filter chains can be empty if the there both ingress and in-mesh policies are not configured.
		// Configuring a listener without a filter chain is an error.
		if marshalledInbound, err := ptypes.MarshalAny(inboundListener); err != nil {
			log.Error().Err(err).Msgf("Error marshalling inbound listener config for proxy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
				proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		} else {
			resp.Resources = append(resp.Resources, marshalledInbound)
		}
	}

	if cfg.IsPrometheusScrapingEnabled() {
		// Build Prometheus listener config
		prometheusConnManager := getPrometheusConnectionManager(prometheusListenerName, constants.PrometheusScrapePath, constants.EnvoyMetricsCluster)
		if prometheusListener, err := buildPrometheusListener(prometheusConnManager); err != nil {
			log.Error().Err(err).Msgf("Error building Prometheus listener config for proxy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
				proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		} else {
			if marshalledPrometheus, err := ptypes.MarshalAny(prometheusListener); err != nil {
				log.Error().Err(err).Msgf("Error marshalling Prometheus listener config for proxy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
					proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
			} else {
				resp.Resources = append(resp.Resources, marshalledPrometheus)
			}
		}
	}

	return resp, nil
}

func newListenerBuilder(meshCatalog catalog.MeshCataloger, svcAccount service.K8sServiceAccount, cfg configurator.Configurator, statsHeaders map[string]string) *listenerBuilder {
	return &listenerBuilder{
		meshCatalog:  meshCatalog,
		svcAccount:   svcAccount,
		cfg:          cfg,
		statsHeaders: statsHeaders,
	}
}
