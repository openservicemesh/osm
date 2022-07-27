package lds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/handler"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
)

type Handler struct {
	handler.XDSHandler

	MeshCatalog      catalog.MeshCataloger
	Proxy            *envoy.Proxy
	DiscoveryRequest *xds_discovery.DiscoveryRequest
	Cfg              configurator.Configurator
	CertManager      *certificate.Manager
	ProxyRegistry    *registry.ProxyRegistry
}

// NewResponse creates a new Listener Discovery Response.
// The response build 3 Listeners:
// 1. Inbound listener to handle incoming traffic
// 2. Outbound listener to handle outgoing traffic
// 3. Prometheus listener for metrics
func (h *Handler) Respond() ([]types.Resource, error) {
	var ldsResources []types.Resource

	var statsHeaders map[string]string
	if featureflags := h.Cfg.GetFeatureFlags(); featureflags.EnableWASMStats {
		statsHeaders = h.Proxy.StatsHeaders()
	}

	lb := newListenerBuilder(h.MeshCatalog, h.Proxy.Identity, h.Cfg, statsHeaders, h.CertManager.GetTrustDomain())

	// --- OUTBOUND -------------------
	outboundListener, err := lb.newOutboundListener()
	if err != nil {
		log.Error().Err(err).Str("proxy", h.Proxy.String()).Msg("Error building outbound listener")
	} else {
		if outboundListener == nil {
			// This check is important to prevent attempting to configure a listener without a filter chain which
			// otherwise results in an error.
			log.Debug().Str("proxy", h.Proxy.String()).Msg("Not programming nil outbound listener")
		} else {
			ldsResources = append(ldsResources, outboundListener)
		}
	}

	// --- INBOUND -------------------
	inboundListener := newInboundListener()

	svcList, err := h.ProxyRegistry.ListProxyServices(h.Proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Str("proxy", h.Proxy.String()).Msgf("Error looking up MeshServices associated with proxy")
		return nil, err
	}

	// Create inbound mesh filter chains based on mesh traffic policies
	inboundMeshTrafficPolicy := h.MeshCatalog.GetInboundMeshTrafficPolicy(lb.serviceIdentity, svcList)
	if inboundMeshTrafficPolicy != nil {
		inboundListener.FilterChains = append(inboundListener.FilterChains, lb.getInboundMeshFilterChains(inboundMeshTrafficPolicy.TrafficMatches)...)
	}
	// Create ingress filter chains per service behind proxy
	for _, proxyService := range svcList {
		// Add ingress filter chains
		ingressFilterChains := lb.getIngressFilterChains(proxyService)
		inboundListener.FilterChains = append(inboundListener.FilterChains, ingressFilterChains...)
	}

	if len(inboundListener.FilterChains) > 0 {
		// Inbound filter chains can be empty if the there both ingress and in-mesh policies are not configured.
		// Configuring a listener without a filter chain is an error.
		ldsResources = append(ldsResources, inboundListener)
	}

	if pod, err := h.MeshCatalog.GetKubeController().GetPodForProxy(h.Proxy); err != nil {
		log.Warn().Str("proxy", h.Proxy.String()).Msgf("Could not find pod for connecting proxy, no metadata was recorded")
	} else if k8s.IsMetricsEnabled(pod) {
		// Build Prometheus listener config
		prometheusConnManager := getPrometheusConnectionManager()
		if prometheusListener, err := buildPrometheusListener(prometheusConnManager); err != nil {
			log.Error().Err(err).Str("proxy", h.Proxy.String()).Msgf("Error building Prometheus listener")
		} else {
			ldsResources = append(ldsResources, prometheusListener)
		}
	}

	return ldsResources, nil
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func newListenerBuilder(meshCatalog catalog.MeshCataloger, svcIdentity identity.ServiceIdentity, cfg configurator.Configurator, statsHeaders map[string]string, trustDomain string) *listenerBuilder {
	return &listenerBuilder{
		meshCatalog:     meshCatalog,
		serviceIdentity: svcIdentity,
		cfg:             cfg,
		statsHeaders:    statsHeaders,
		trustDomain:     trustDomain,
	}
}

func (h *Handler) SetProxy(proxy *envoy.Proxy) {
	h.Proxy = proxy
}

func (h *Handler) SetDiscoveryRequest(request *xds_discovery.DiscoveryRequest) {
	h.DiscoveryRequest = request
}
