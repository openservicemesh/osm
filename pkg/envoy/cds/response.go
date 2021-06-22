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
	svcList, err := proxyRegistry.ListProxyServices(proxy)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}

	var clusters []*xds_cluster.Cluster

	proxyIdentity, err := envoy.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}
	log.Debug().Msgf("svc:%s url:%s outboundServices:%+v", proxyServiceName, resp.TypeUrl, outboundServices)

	// Build remote clusters based on allowed outbound services
	for _, dstService := range meshCatalog.ListAllowedOutboundServicesForIdentity(proxyIdentity.ToServiceIdentity()) {
		cluster, err := getUpstreamServiceCluster(proxyIdentity.ToServiceIdentity(), dstService, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to construct service cluster for service %s for proxy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
				dstService.Name, proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
			return nil, err
		}

<<<<<<< HEAD
		if catalog.GetWitesandCataloger().IsWSEdgePodService(dstService) {
			getWSEdgePodUpstreamServiceCluster(catalog, dstService, proxyServiceName.GetMeshServicePort(), cfg, clusterFactories)
			continue
		} else if catalog.GetWitesandCataloger().IsWSUnicastService(dstService.Name) {
			getWSUnicastUpstreamServiceCluster(catalog, dstService, proxyServiceName.GetMeshServicePort(), cfg, clusterFactories)
			// fall thru to generate anycast cluster
		}

		remoteCluster, err := getUpstreamServiceCluster(dstService, proxyServiceName.GetMeshServicePort(), cfg)

=======
		clusters = append(clusters, cluster)
	}

	// Create a local cluster for each service behind the proxy.
	// The local cluster will be used to handle incoming traffic.
	for _, proxyService := range svcList {
		localClusterName := envoy.GetLocalClusterNameForService(proxyService)
		localCluster, err := getLocalServiceCluster(meshCatalog, proxyService, localClusterName)
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get local cluster config for proxy %s", proxyService)
			return nil, err
		}
		clusters = append(clusters, localCluster)
	}

<<<<<<< HEAD
		if featureflags.IsBackpressureEnabled() {
			enableBackpressure(catalog, remoteCluster, dstService.GetMeshService())
		}
		//log.Debug().Msgf("remoteName:%s, remoteCluster:%+v", remoteCluster.Name, remoteCluster)

		clusterFactories[remoteCluster.Name] = remoteCluster
	}

	// Create a local cluster for the service.
	// The local cluster will be used for incoming traffic.
	localClusters, err := getLocalServiceCluster(catalog, proxyServiceName)
=======
	// Add egress clusters based on applied policies
	if egressTrafficPolicy, err := meshCatalog.GetEgressTrafficPolicy(proxyIdentity.ToServiceIdentity()); err != nil {
		log.Error().Err(err).Msgf("Error retrieving egress policies for proxy with identity %s, skipping egress clusters", proxyIdentity)
	} else {
		if egressTrafficPolicy != nil {
			clusters = append(clusters, getEgressClusters(egressTrafficPolicy.ClustersConfigs)...)
		}
	}

	outboundPassthroughCluser, err := getOriginalDestinationEgressCluster(envoy.OutboundPassthroughCluster)
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
	if err != nil {
		log.Error().Err(err).Msgf("Failed to passthrough cluster for egress for proxy %s", envoy.OutboundPassthroughCluster)
		return nil, err
	}
<<<<<<< HEAD

	count := 0
	for _, localCluster := range localClusters {
		clusterFactories[localCluster.Name] = localCluster
		log.Debug().Msgf("local:%s localCluster:%+v", localCluster.Name, localCluster)
		count++
	}
=======
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7

	// Add an outbound passthrough cluster for egress if global mesh-wide Egress is enabled
	if cfg.IsEgressEnabled() {
		clusters = append(clusters, outboundPassthroughCluser)
	}

	// Add an inbound prometheus cluster (from Prometheus to localhost)
	if cfg.IsPrometheusScrapingEnabled() {
		clusters = append(clusters, getPrometheusCluster())
	}

	// Add an outbound tracing cluster (from localhost to tracing sink)
	if cfg.IsTracingEnabled() {
		clusters = append(clusters, getTracingCluster(cfg))
	}

	alreadyAdded := mapset.NewSet()
	var cdsResources []types.Resource
	for _, cluster := range clusters {
		if alreadyAdded.Contains(cluster.Name) {
			log.Error().Msgf("Found duplicate clusters with name %s; Duplicate will not be sent to Envoy with XDS Certificate SerialNumber=%s on Pod with UID=%s",
				cluster.Name, proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
			continue
		}
		alreadyAdded.Add(cluster.Name)
		cdsResources = append(cdsResources, cluster)
	}
	//log.Debug().Msgf("Proxy service %s CDS resp: %+v ", proxyServiceName, resp.Resources)

	return cdsResources, nil
}
