package catalog

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/ticker"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(meshSpec smi.MeshSpec, certManager *certificate.Manager,
	policyController policy.Controller, stop <-chan struct{},
	computeInterface compute.Interface,
	msgBroker *messaging.Broker) *MeshCatalog {
	mc := &MeshCatalog{
		Interface:        computeInterface,
		meshSpec:         meshSpec,
		policyController: policyController,

		certManager: certManager,
	}

	// Start the Resync ticker to tick based on the resync interval.
	// Starting the resync ticker only starts the ticker config watcher which
	// internally manages the lifecycle of the ticker routine.
	resyncTicker := ticker.NewResyncTicker(msgBroker, 30*time.Second /* min resync interval */)
	resyncTicker.Start(stop)

	return mc
}

// GetTrustDomain returns the currently configured trust domain, ie: cluster.local
func (mc *MeshCatalog) GetTrustDomain() string {
	return mc.certManager.GetTrustDomain()
}
