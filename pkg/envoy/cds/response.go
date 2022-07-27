package cds

import (
	mapset "github.com/deckarep/golang-set"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/handler"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
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

// NewResponse creates a new Cluster Discovery Response.
// func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, _ *certificate.Manager, proxyRegistry *registry.ProxyRegistry) ([]types.Resource, error) {

func (h *Handler) Respond() ([]types.Resource, error) {

	var clusters []*xds_cluster.Cluster

	// Build upstream clusters based on allowed outbound traffic policies
	outboundMeshTrafficPolicy := h.MeshCatalog.GetOutboundMeshTrafficPolicy(h.Proxy.Identity)
	if outboundMeshTrafficPolicy != nil {
		clusters = append(clusters, upstreamClustersFromClusterConfigs(h.Proxy.Identity, outboundMeshTrafficPolicy.ClustersConfigs, h.Cfg.GetMeshConfig().Spec.Sidecar)...)
	}

	// Build local clusters based on allowed inbound traffic policies
	proxyServices, err := h.ProxyRegistry.ListProxyServices(h.Proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingServiceList)).
			Str("proxy", h.Proxy.String()).Msg("Error looking up MeshServices associated with proxy")
		return nil, err
	}
	inboundMeshTrafficPolicy := h.MeshCatalog.GetInboundMeshTrafficPolicy(h.Proxy.Identity, proxyServices)
	if inboundMeshTrafficPolicy != nil {
		clusters = append(clusters, localClustersFromClusterConfigs(inboundMeshTrafficPolicy.ClustersConfigs)...)
	}

	// Add egress clusters based on applied policies
	if egressTrafficPolicy, err := h.MeshCatalog.GetEgressTrafficPolicy(h.Proxy.Identity); err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress policies for proxy with identity %s, skipping egress clusters", h.Proxy.Identity)
	} else {
		if egressTrafficPolicy != nil {
			clusters = append(clusters, getEgressClusters(egressTrafficPolicy.ClustersConfigs)...)
		}
	}

	outboundPassthroughCluser, err := getOriginalDestinationEgressCluster(envoy.OutboundPassthroughCluster, nil)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrGettingOrgDstEgressCluster.String()).
			Msgf("Failed to passthrough cluster for egress for proxy %s", envoy.OutboundPassthroughCluster)
		return nil, err
	}

	// Add an outbound passthrough cluster for egress if global mesh-wide Egress is enabled
	if h.Cfg.IsEgressEnabled() {
		clusters = append(clusters, outboundPassthroughCluser)
	}

	// Add an inbound prometheus cluster (from Prometheus to localhost)
	if pod, err := h.MeshCatalog.GetKubeController().GetPodForProxy(h.Proxy); err != nil {
		log.Warn().Str("proxy", h.Proxy.String()).Msg("Could not find pod for connecting proxy, no metadata was recorded")
	} else if k8s.IsMetricsEnabled(pod) {
		clusters = append(clusters, getPrometheusCluster())
	}

	// Add an outbound tracing cluster (from localhost to tracing sink)
	if h.Cfg.IsTracingEnabled() {
		clusters = append(clusters, getTracingCluster(h.Cfg))
	}

	return removeDups(clusters), nil
}

func removeDups(clusters []*xds_cluster.Cluster) []types.Resource {
	alreadyAdded := mapset.NewSet()
	var cdsResources []types.Resource
	for _, cluster := range clusters {
		if alreadyAdded.Contains(cluster.Name) {
			log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDuplicateClusters)).
				Msgf("Found duplicate clusters with name %s; duplicate will not be sent to h.Proxy.", cluster.Name)
			continue
		}
		alreadyAdded.Add(cluster.Name)
		cdsResources = append(cdsResources, cluster)
	}

	return cdsResources
}

func (h *Handler) SetMeshCataloger(cataloger catalog.MeshCataloger) {
	h.MeshCatalog = cataloger
}

func (h *Handler) SetProxy(proxy *envoy.Proxy) {
	h.Proxy = proxy
}

func (h *Handler) SetDiscoveryRequest(request *xds_discovery.DiscoveryRequest) {
	h.DiscoveryRequest = request
}

func (h *Handler) SetConfigurator(cfg configurator.Configurator) {
	h.Cfg = cfg
}

func (h *Handler) SetCertManager(certManager *certificate.Manager) {
	h.CertManager = certManager
}

func (h *Handler) SetProxyRegistry(proxyRegistry *registry.ProxyRegistry) {
	h.ProxyRegistry = proxyRegistry
}
