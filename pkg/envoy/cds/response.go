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
	"github.com/openservicemesh/osm/pkg/envoy/registry"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, _ certificate.Manager, proxyRegistry *registry.ProxyRegistry) ([]types.Resource, error) {
	var clusters []*xds_cluster.Cluster

	proxyIdentity, err := envoy.GetServiceIdentityFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up identity for proxy %s", proxy.String())
		return nil, err
	}

	// Build remote clusters based on allowed outbound services
	for _, dstService := range meshCatalog.ListAllowedOutboundServicesForIdentity(proxyIdentity) {
		// The global service is comprised of other weighted clusters, so doesn't need it's own specific cluster.
		if dstService.Global() {
			continue
		}
		opts := []clusterOption{withTLS}
		if cfg.IsPermissiveTrafficPolicyMode() {
			opts = append(opts, permissive)
		}
		cluster, err := getUpstreamServiceCluster(proxyIdentity, dstService, cfg, opts...)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to construct service cluster for service %s for proxy %s", dstService.Name, proxy.String())
			return nil, err
		}

		clusters = append(clusters, cluster)
	}

	if proxy.Kind() == envoy.KindGateway {
		return removeDups(clusters), nil
	}

	svcList, err := proxyRegistry.ListProxyServices(proxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for proxy %s", proxy.String())
		return nil, err
	}

	// Create a local cluster for each service behind the proxy.
	// The local cluster will be used to handle incoming traffic.
	for _, proxyService := range svcList {
		localClusterName := envoy.GetLocalClusterNameForService(proxyService)
		localCluster, err := getLocalServiceCluster(meshCatalog, proxyService, localClusterName)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get local cluster config for proxy %s", proxyService)
			return nil, err
		}
		clusters = append(clusters, localCluster)
	}

	// Add egress clusters based on applied policies
	if egressTrafficPolicy, err := meshCatalog.GetEgressTrafficPolicy(proxyIdentity); err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress policies for proxy with identity %s, skipping egress clusters", proxyIdentity)
	} else {
		if egressTrafficPolicy != nil {
			clusters = append(clusters, getEgressClusters(egressTrafficPolicy.ClustersConfigs)...)
		}
	}

	outboundPassthroughCluser, err := getOriginalDestinationEgressCluster(envoy.OutboundPassthroughCluster)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to passthrough cluster for egress for proxy %s", envoy.OutboundPassthroughCluster)
		return nil, err
	}

	// Add an outbound passthrough cluster for egress if global mesh-wide Egress is enabled
	if cfg.IsEgressEnabled() {
		clusters = append(clusters, outboundPassthroughCluser)
	}

	// Add an inbound prometheus cluster (from Prometheus to localhost)
	if pod, err := envoy.GetPodFromCertificate(proxy.GetCertificateCommonName(), meshCatalog.GetKubeController()); err != nil {
		log.Warn().Msgf("Could not find pod for connecting proxy %s. No metadata was recorded.", proxy.GetCertificateSerialNumber())
	} else if meshCatalog.GetKubeController().IsMetricsEnabled(pod) {
		clusters = append(clusters, getPrometheusCluster())
	}

	// Add an outbound tracing cluster (from localhost to tracing sink)
	if cfg.IsTracingEnabled() {
		clusters = append(clusters, getTracingCluster(cfg))
	}

	return removeDups(clusters), nil
}

func removeDups(clusters []*xds_cluster.Cluster) []types.Resource {
	alreadyAdded := mapset.NewSet()
	var cdsResources []types.Resource
	for _, cluster := range clusters {
		if alreadyAdded.Contains(cluster.Name) {
			log.Error().Msgf("Found duplicate clusters with name %s; duplicate will not be sent to proxy.", cluster.Name)
			continue
		}
		alreadyAdded.Add(cluster.Name)
		cdsResources = append(cdsResources, cluster)
	}

	return cdsResources
}
