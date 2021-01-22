package cds

import (
	mapset "github.com/deckarep/golang-set"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/featureflags"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	svcList, err := meshCatalog.GetServicesFromEnvoyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}
	// Github Issue #1575
	proxyServiceName := svcList[0]

	var clusters []*xds_cluster.Cluster

	proxyIdentity, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}

	// Build remote clusters based on allowed outbound services
	for _, dstService := range meshCatalog.ListAllowedOutboundServicesForIdentity(proxyIdentity) {
		cluster, err := getUpstreamServiceCluster(dstService, proxyServiceName, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to construct service cluster for service %s for proxy %s", dstService.Name, proxyServiceName)
			return nil, err
		}

		if featureflags.IsBackpressureEnabled() {
			enableBackpressure(meshCatalog, cluster, dstService)
		}

		clusters = append(clusters, cluster)
	}

	// Create a local cluster for the service.
	// The local cluster will be used for incoming traffic.
	localClusterName := envoy.GetLocalClusterNameForService(proxyServiceName)
	localCluster, err := getLocalServiceCluster(meshCatalog, proxyServiceName, localClusterName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get local cluster config for proxy %s", proxyServiceName)
		return nil, err
	}
	clusters = append(clusters, localCluster)

	// Add an outbound passthrough cluster for egress
	if cfg.IsEgressEnabled() {
		clusters = append(clusters, getOutboundPassthroughCluster())
	}

	// Add an inbound prometheus cluster (from Prometheus to localhost)
	if cfg.IsPrometheusScrapingEnabled() {
		clusters = append(clusters, getPrometheusCluster())
	}

	// Add an outbound tracing cluster (from localhost to tracing sink)
	if cfg.IsTracingEnabled() {
		clusters = append(clusters, getTracingCluster(cfg))
	}

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeCDS),
	}

	alreadyAdded := mapset.NewSet()
	for _, cluster := range clusters {
		if alreadyAdded.Contains(cluster.Name) {
			log.Error().Msgf("Found duplicate clusters with name %s; Duplicate will not be sent to Envoy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
				cluster.Name, proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
			continue
		}
		alreadyAdded.Add(cluster.Name)
		marshalledClusters, err := ptypes.MarshalAny(cluster)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal cluster %s for Envoy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
				cluster.Name, proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}

	return resp, nil
}
