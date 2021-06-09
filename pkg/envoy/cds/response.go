package cds

import (
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
func NewResponse(catalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, _ certificate.Manager) (*xds_discovery.DiscoveryResponse, error) {
	svcList, err := catalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	// Github Issue #1575
	proxyServiceName := svcList[0]

	resp := &xds_discovery.DiscoveryResponse{
		TypeUrl: string(envoy.TypeCDS),
	}
	// The clusters have to be unique, so use a map to prevent duplicates. Keys correspond to the cluster name.
	clusterFactories := make(map[string]*xds_cluster.Cluster)

	outboundServices, err := catalog.ListAllowedOutboundServices(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Error listing outbound services for proxy %q", proxyServiceName)
		return nil, err
	}
	log.Debug().Msgf("svc:%s url:%s outboundServices:%+v", proxyServiceName, resp.TypeUrl, outboundServices)

	// Build remote clusters based on allowed outbound services
	for _, dstService := range outboundServices {
		if _, found := clusterFactories[dstService.String()]; found {
			// Guard against duplicates
			continue
		}

		if catalog.GetWitesandCataloger().IsWSEdgePodService(dstService) {
			getWSEdgePodUpstreamServiceCluster(catalog, dstService, proxyServiceName.GetMeshServicePort(), cfg, clusterFactories)
			continue
		} else if catalog.GetWitesandCataloger().IsWSUnicastService(dstService.Name) {
			getWSUnicastUpstreamServiceCluster(catalog, dstService, proxyServiceName.GetMeshServicePort(), cfg, clusterFactories)
			// fall thru to generate anycast cluster
		}

		remoteCluster, err := getUpstreamServiceCluster(dstService, proxyServiceName.GetMeshServicePort(), cfg)

		if err != nil {
			log.Error().Err(err).Msgf("Failed to construct service cluster for proxy %s", proxyServiceName)
			return nil, err
		}

		if featureflags.IsBackpressureEnabled() {
			enableBackpressure(catalog, remoteCluster, dstService.GetMeshService())
		}
		//log.Debug().Msgf("remoteName:%s, remoteCluster:%+v", remoteCluster.Name, remoteCluster)

		clusterFactories[remoteCluster.Name] = remoteCluster
	}

	// Create a local cluster for the service.
	// The local cluster will be used for incoming traffic.
	localClusters, err := getLocalServiceCluster(catalog, proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get local cluster config for proxy %s", proxyServiceName)
		return nil, err
	}

	count := 0
	for _, localCluster := range localClusters {
		clusterFactories[localCluster.Name] = localCluster
		log.Debug().Msgf("local:%s localCluster:%+v", localCluster.Name, localCluster)
		count++
	}

	if cfg.IsEgressEnabled() {
		// Add a pass-through cluster for egress
		passthroughCluster := getOutboundPassthroughCluster()
		clusterFactories[passthroughCluster.Name] = passthroughCluster
	}

	for _, cluster := range clusterFactories {
		log.Debug().Msgf("Proxy service %s constructed ClusterConfiguration: %+v ", proxyServiceName, cluster)
		marshalledClusters, err := ptypes.MarshalAny(cluster)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to marshal cluster for proxy %s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledClusters)
	}

	if cfg.IsPrometheusScrapingEnabled() {
		prometheusCluster := getPrometheusCluster()
		marshalledCluster, err := ptypes.MarshalAny(&prometheusCluster)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshaling Prometheus cluster for proxy with CN=%s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledCluster)
	}

	if cfg.IsTracingEnabled() {
		tracingCluster := getTracingCluster(cfg)
		marshalledCluster, err := ptypes.MarshalAny(&tracingCluster)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshaling tracing cluster for proxy with CN=%s", proxy.GetCommonName())
			return nil, err
		}
		resp.Resources = append(resp.Resources, marshalledCluster)
	}
	//log.Debug().Msgf("Proxy service %s CDS resp: %+v ", proxyServiceName, resp.Resources)

	return resp, nil
}
