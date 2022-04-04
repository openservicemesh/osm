package catalog

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/ticker"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(kubeController k8s.Controller, meshSpec smi.MeshSpec, certManager certificate.Manager,
	policyController policy.Controller, stop <-chan struct{},
	cfg configurator.Configurator, serviceProviders []service.Provider, endpointsProviders []endpoint.Provider,
	msgBroker *messaging.Broker) *MeshCatalog {
	mc := &MeshCatalog{
		serviceProviders:   serviceProviders,
		endpointsProviders: endpointsProviders,
		meshSpec:           meshSpec,
		policyController:   policyController,
		configurator:       cfg,

		kubeController: kubeController,
	}

	// Start the Resync ticker to tick based on the resync interval.
	// Starting the resync ticker only starts the ticker config watcher which
	// internally manages the lifecycle of the ticker routine.
	resyncTicker := ticker.NewResyncTicker(msgBroker, 30*time.Second /* min resync interval */)
	resyncTicker.Start(stop)

	return mc
}

// GetKubeController returns the kube controller instance handling the current cluster
func (mc *MeshCatalog) GetKubeController() k8s.Controller {
	return mc.kubeController
}
