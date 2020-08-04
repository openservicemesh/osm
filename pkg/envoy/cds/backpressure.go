package cds

import (
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
)

func enableBackpressure(catalog catalog.MeshCataloger, remoteCluster *xds_cluster.Cluster) {
	log.Info().Msgf("Enabling backpressure in service cluster")
	// Backpressure CRD only has one backpressure obj as a global config
	// TODO: Add specific backpressure settings for individual clients
	backpressures := catalog.GetSMISpec().ListBackpressures()

	// TODO: filter backpressures on labels (backpressures[i].ObjectMeta.Labels) that match that of the destination service (trafficPolicies.Destination)

	log.Trace().Msgf("Backpressures (%d found): %+v", len(backpressures), backpressures)

	if len(backpressures) > 0 {
		log.Trace().Msgf("Backpressure Spec: %+v", backpressures[0].Spec)

		remoteCluster.CircuitBreakers = &xds_cluster.CircuitBreakers{
			Thresholds: makeThresholds(&backpressures[0].Spec.MaxConnections),
		}

	}
}
