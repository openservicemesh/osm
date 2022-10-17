package catalog

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/ticker"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(computeInterface compute.Interface, certManager *certificate.Manager,
	stop <-chan struct{},
	msgBroker *messaging.Broker) *MeshCatalog {
	mc := &MeshCatalog{
		Interface:   computeInterface,
		certManager: certManager,
	}

	// Start the Resync ticker to tick based on the resync interval.
	// Starting the resync ticker only starts the ticker config watcher which
	// internally manages the lifecycle of the ticker routine.
	resyncTicker := ticker.NewResyncTicker(msgBroker, 30*time.Second /* min resync interval */)
	resyncTicker.Start(stop)

	return mc
}
