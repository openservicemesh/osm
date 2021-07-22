package cds

import (
	mapset "github.com/deckarep/golang-set"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Cluster Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, _ certificate.Manager, proxyRegistry *registry.ProxyRegistry) ([]types.Resource, error) {
	if proxy.Kind() == envoy.KindGateway {
		return getClustersForMulticlusterGateway(meshCatalog)
	}

	// TODO(draychev): Why is GetServiceIdentityFromProxyCertificate not on proxy.GetServiceIdentity()?
	proxyIdentity, err := envoy.GetServiceIdentityFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrGettingServiceIdentity.String()).
			Msgf("Error looking up identity for proxy %s", proxy.String())
		return nil, err
	}

	var clusters []*xds_cluster.Cluster

	remoteClusters, err := getRemoteClusters(proxy, proxyIdentity, cfg, meshCatalog)
	if err != nil {
		return nil, err
	}
	clusters = append(clusters, remoteClusters...)

	localClusters, err := getLocalClusters(proxy, proxyRegistry, meshCatalog)
	if err != nil {
		return nil, err
	}
	clusters = append(clusters, localClusters...)

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
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrGettingOrgDstEgressCluster.String()).
			Msgf("Failed to passthrough cluster for egress for proxy %s", envoy.OutboundPassthroughCluster)
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

func getRemoteClusters(proxy *envoy.Proxy, proxyIdentity identity.ServiceIdentity, cfg configurator.Configurator, meshCatalog catalog.MeshCataloger) ([]*xds_cluster.Cluster, error) {
	var clusters []*xds_cluster.Cluster

	// Build remote clusters based on allowed outbound services
	for _, dstService := range meshCatalog.ListOutboundServicesForIdentity(proxyIdentity) {
		opts := []clusterOption{withTLS}
		if cfg.IsPermissiveTrafficPolicyMode() {
			opts = append(opts, permissive)
		}
		cluster, err := getUpstreamServiceCluster(proxyIdentity, dstService, cfg, opts...)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.ErrObtainingUpstreamServiceCluster.String()).
				Msgf("Failed to construct service cluster for service %s for proxy %s", dstService.Name, proxy.String())
			return nil, err
		}

		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

func getLocalClusters(proxy *envoy.Proxy, proxyRegistry registry.ProxyServiceMapper, meshCatalog catalog.MeshCataloger) ([]*xds_cluster.Cluster, error) {
	var clusters []*xds_cluster.Cluster

	svcList, err := proxyRegistry.ListProxyServices(proxy)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrFetchingServiceList.String()).
			Msgf("Error looking up MeshService for proxy %s", proxy.String())
		return nil, err
	}

	// Create a local cluster for each service behind the proxy.
	// The local cluster will be used to handle incoming traffic.
	for _, proxyService := range svcList {
		localClusterName := envoy.GetLocalClusterNameForService(proxyService)
		localCluster, err := getLocalServiceCluster(meshCatalog, proxyService, localClusterName)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.ErrGettingLocalServiceCluster.String()).
				Msgf("Failed to get local cluster config for proxy %s", proxyService)
			return nil, err
		}
		clusters = append(clusters, localCluster)
	}
	return clusters, nil
}

func removeDups(clusters []*xds_cluster.Cluster) []types.Resource {
	alreadyAdded := mapset.NewSet()
	var cdsResources []types.Resource
	for _, cluster := range clusters {
		if alreadyAdded.Contains(cluster.Name) {
			log.Error().Str(errcode.Kind, errcode.ErrDuplicateClusters.String()).
				Msgf("Found duplicate clusters with name %s; duplicate will not be sent to proxy.", cluster.Name)
			continue
		}
		alreadyAdded.Add(cluster.Name)
		cdsResources = append(cdsResources, cluster)
	}

	return cdsResources
}

func getGatewayRemoteCluster(remoteService service.MeshService) (*xds_cluster.Cluster, error) {
	clusterName := remoteService.NameWithoutCluster()
	marshalledUpstreamTLSContext, err := ptypes.MarshalAny(envoy.GetUpstreamTLSContext(identity.ServiceIdentity(remoteService.FQDN()), remoteService))
	if err != nil {
		log.Err(err).Msg("Error creating TLS Context for OSM Gateway")
		return nil, err
	}

	return &xds_cluster.Cluster{
		Name:           clusterName,
		AltStatName:    clusterName,
		ConnectTimeout: ptypes.DurationProto(clusterConnectTimeout),
		ClusterDiscoveryType: &xds_cluster.Cluster_Type{
			Type: xds_cluster.Cluster_EDS,
		},
		EdsClusterConfig: &xds_cluster.Cluster_EdsClusterConfig{
			EdsConfig: envoy.GetADSConfigSource(),
		},
		LbPolicy: xds_cluster.Cluster_ROUND_ROBIN,
		TransportSocket: &xds_core.TransportSocket{
			Name: wellknown.TransportSocketTls,
			ConfigType: &xds_core.TransportSocket_TypedConfig{
				TypedConfig: marshalledUpstreamTLSContext,
			},
		},
	}, nil
}

func getClustersForMulticlusterGateway(meshCatalog catalog.MeshCataloger) ([]types.Resource, error) {
	var clusters []*xds_cluster.Cluster
	for _, svc := range meshCatalog.ListAllMeshServices() {
		remoteCluster, err := getGatewayRemoteCluster(svc)
		if err != nil {
			log.Error().Err(err).Msg("Error constructing remote service cluster for Multicluster gateway proxy")
			return nil, err
		}
		clusters = append(clusters, remoteCluster)
	}
	return removeDups(clusters), nil
}
