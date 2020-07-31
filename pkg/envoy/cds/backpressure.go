package cds

import (
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/openservicemesh/osm/pkg/smi"
)

func enableBackpressure(meshSpec smi.MeshSpec, remoteCluster *xds_cluster.Cluster) {
	log.Info().Msgf("Enabling backpressure in service cluster")
	// Backpressure CRD only has one backpressure obj as a global config
	// TODO: Add specific backpressure settings for individual clients
	backpressures := meshSpec.ListBackpressures()

	// TODO: filter backpressures on labels (backpressures[i].ObjectMeta.Labels) that match that of the destination service (trafficPolicies.Destination.MeshService)

	log.Trace().Msgf("Backpressures (%d found): %+v", len(backpressures), backpressures)

	if len(backpressures) > 0 {
		log.Trace().Msgf("Backpressure Spec: %+v", backpressures[0].Spec)

		remoteCluster.CircuitBreakers = &xds_cluster.CircuitBreakers{
			Thresholds: makeThresholds(&backpressures[0].Spec.MaxConnections),
		}

	}
}
