package cds

import (
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/service"
)

func enableBackpressure(catalog catalog.MeshCataloger, remoteCluster *xds_cluster.Cluster, svc service.MeshService) {
	log.Info().Msgf("Enabling backpressure in service cluster")
	// Backpressure CRD only has one backpressure obj as a global config
	// TODO: Add specific backpressure settings for individual clients
	backpressure := catalog.GetSMISpec().GetBackpressurePolicy(svc)
	if backpressure == nil {
		log.Trace().Msgf("Backpressure policy not found for service %s", svc)
		return
	}

	log.Trace().Msgf("Backpressure Spec: %+v", backpressure.Spec)
	remoteCluster.CircuitBreakers = &xds_cluster.CircuitBreakers{
		Thresholds: makeThresholds(&backpressure.Spec.MaxConnections),
	}
}
