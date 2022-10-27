package catalog

import (
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
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

// GetUpstreamServicesIncludeApex returns a list of all upstream services associated with the given list
// of services. An upstream service is associated with another service if it is a backend for an apex/root service
// in a TrafficSplit config. This function returns a list consisting of the given upstream services and all apex
// services associated with each of those services.
func (mc *MeshCatalog) GetUpstreamServicesIncludeApex(upstreamServices []service.MeshService) []service.MeshService {
	svcSet := mapset.NewSet()
	var allServices []service.MeshService

	// Each service could be a backend in a traffic split config. Construct a list
	// of all possible services the given list of services is associated with.
	for _, svc := range upstreamServices {
		if newlyAdded := svcSet.Add(svc); newlyAdded {
			allServices = append(allServices, svc)
		}

		for _, split := range mc.ListTrafficSplitsByOptions(smi.WithTrafficSplitBackendService(svc)) {
			apexMeshService := service.MeshService{
				Namespace:  svc.Namespace,
				Name:       split.Spec.Service,
				Port:       svc.Port,
				TargetPort: svc.TargetPort,
				Protocol:   svc.Protocol,
			}

			if newlyAdded := svcSet.Add(apexMeshService); newlyAdded {
				allServices = append(allServices, apexMeshService)
			}
		}
	}

	return allServices
}
