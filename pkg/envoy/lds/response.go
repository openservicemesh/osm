package lds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
)

// NewResponse creates a new Listener Discovery Response.
// The response build 3 Listeners:
// 1. Inbound listener to handle incoming traffic
// 2. Outbound listener to handle outgoing traffic
// 3. Prometheus listener for metrics
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, cm *certificate.Manager, proxyRegistry *registry.ProxyRegistry) ([]types.Resource, error) {
	var ldsResources []types.Resource

	var statsHeaders map[string]string
	if featureflags := cfg.GetFeatureFlags(); featureflags.EnableWASMStats {
		statsHeaders = proxy.StatsHeaders()
	}

	lb := newListenerBuilder(meshCatalog, proxy.Identity, cfg, statsHeaders, cm.GetTrustDomain())

	// --- OUTBOUND -------------------
	outboundListener, err := lb.newOutboundListener()
	if err != nil {
		log.Error().Err(err).Str("proxy", proxy.String()).Msg("Error building outbound listener")
	} else {
		if outboundListener == nil {
			// This check is important to prevent attempting to configure a listener without a filter chain which
			// otherwise results in an error.
			log.Debug().Str("proxy", proxy.String()).Msg("Not programming nil outbound listener")
		} else {
			ldsResources = append(ldsResources, outboundListener)
		}
	}

	// --- INBOUND -------------------
	inboundListener := newInboundListener()

	svcList, err := proxyRegistry.ListProxyServices(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Str("proxy", proxy.String()).Msgf("Error looking up MeshServices associated with proxy")
		return nil, err
	}

	// Create inbound mesh filter chains based on mesh traffic policies
	inboundMeshTrafficPolicy := meshCatalog.GetInboundMeshTrafficPolicy(lb.serviceIdentity, svcList)
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

	if pod, err := meshCatalog.GetKubeController().GetPodForProxy(proxy); err != nil {
		log.Warn().Str("proxy", proxy.String()).Msgf("Could not find pod for connecting proxy, no metadata was recorded")
	} else if k8s.IsMetricsEnabled(pod) {
		// Build Prometheus listener config
		prometheusConnManager := getPrometheusConnectionManager()
		if prometheusListener, err := buildPrometheusListener(prometheusConnManager); err != nil {
			log.Error().Err(err).Str("proxy", proxy.String()).Msgf("Error building Prometheus listener")
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
