package cds

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/witesand"
	"strconv"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

// getWSEdgePodUpstreamServiceCluster returns an Envoy Cluster corresponding to the given upstream service
func getWSEdgePodUpstreamServiceCluster(catalog catalog.MeshCataloger, upstreamSvc, downstreamSvc service.MeshServicePort, cfg configurator.Configurator, clusterFactories map[string]*xds_cluster.Cluster) error {
	wscatalog := catalog.GetWitesandCataloger()
	apigroupClusterNames, err := wscatalog.ListApigroupClusterNames()
	if err != nil {
		return err
	}
	edgePodNames, err := wscatalog.ListAllEdgePods()
	if err != nil {
		return err
	}

	// create clusters with apigroup-names with ROUND_ROBIN
	for _, apigroupName := range apigroupClusterNames {
		clusterName := apigroupName + ":" + strconv.Itoa(upstreamSvc.Port)

		remoteCluster := &xds_cluster.Cluster{
			Name:                 clusterName,
			ConnectTimeout:       ptypes.DurationProto(clusterConnectTimeout),
			ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
			Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
			CircuitBreakers: &xds_cluster.CircuitBreakers{
				Thresholds:   makeWSThresholds(),
			},
		}

		remoteCluster.ClusterDiscoveryType = &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS}
		remoteCluster.EdsClusterConfig = &xds_cluster.Cluster_EdsClusterConfig{EdsConfig: envoy.GetADSConfigSource()}
		remoteCluster.LbPolicy = xds_cluster.Cluster_ROUND_ROBIN
		clusterFactories[remoteCluster.Name] = remoteCluster
	}

	// create clusters with apigroup-names + "device-hash" with ROUND_ROBIN
	for _, apigroupName := range apigroupClusterNames {
		clusterName := apigroupName + witesand.DeviceHashSuffix + ":" + strconv.Itoa(upstreamSvc.Port)

		remoteCluster := &xds_cluster.Cluster{
			Name:                 clusterName,
			ConnectTimeout:       ptypes.DurationProto(clusterConnectTimeout),
			ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
			Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
			CircuitBreakers: &xds_cluster.CircuitBreakers{
				Thresholds:   makeWSThresholds(),
			},
		}

		remoteCluster.ClusterDiscoveryType = &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS}
		remoteCluster.EdsClusterConfig = &xds_cluster.Cluster_EdsClusterConfig{EdsConfig: envoy.GetADSConfigSource()}
		remoteCluster.LbPolicy = xds_cluster.Cluster_RING_HASH
		clusterFactories[remoteCluster.Name] = remoteCluster
	}

	// create clusters with pod-names with ROUND_ROBIN
	for _, edgePodName := range edgePodNames {
		clusterName := edgePodName + ":" + strconv.Itoa(upstreamSvc.Port)

		remoteCluster := &xds_cluster.Cluster{
			Name:                 clusterName,
			ConnectTimeout:       ptypes.DurationProto(clusterConnectTimeout),
			ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
			Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
			CircuitBreakers: &xds_cluster.CircuitBreakers{
				Thresholds:   makeWSThresholds(),
			},
		}

		remoteCluster.ClusterDiscoveryType = &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS}
		remoteCluster.EdsClusterConfig = &xds_cluster.Cluster_EdsClusterConfig{EdsConfig: envoy.GetADSConfigSource()}
		remoteCluster.LbPolicy = xds_cluster.Cluster_ROUND_ROBIN
		clusterFactories[remoteCluster.Name] = remoteCluster
	}

	return nil
}

// create one cluster for each pod in the service.
// cluster name of the form "<pod-name>:<port-num>"
func getWSUnicastUpstreamServiceCluster(catalog catalog.MeshCataloger, upstreamSvc, downstreamSvc service.MeshServicePort, cfg configurator.Configurator, clusterFactories map[string]*xds_cluster.Cluster) error {
	serviceEndpoints, err := catalog.ListEndpointsForService(upstreamSvc.GetMeshService())
	if err != nil {
		return err
	}

	// create clusters with pod-names
	for _, endpoint := range serviceEndpoints {
		clusterName := endpoint.PodName + ":" + strconv.Itoa(upstreamSvc.Port)

		remoteCluster := &xds_cluster.Cluster{
			Name:                 clusterName,
			ConnectTimeout:       ptypes.DurationProto(clusterConnectTimeout),
			ProtocolSelection:    xds_cluster.Cluster_USE_DOWNSTREAM_PROTOCOL,
			Http2ProtocolOptions: &xds_core.Http2ProtocolOptions{},
			CircuitBreakers: &xds_cluster.CircuitBreakers{
				Thresholds:   makeWSThresholds(),
			},
		}

		remoteCluster.ClusterDiscoveryType = &xds_cluster.Cluster_Type{Type: xds_cluster.Cluster_EDS}
		remoteCluster.EdsClusterConfig = &xds_cluster.Cluster_EdsClusterConfig{EdsConfig: envoy.GetADSConfigSource()}
		clusterFactories[remoteCluster.Name] = remoteCluster
	}

	return nil
}

