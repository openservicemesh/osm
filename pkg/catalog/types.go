package catalog

import (
	"sync"

	"github.com/deislabs/smc/pkg/mesh"
)

// ServiceCatalog is the struct for the service catalog
type ServiceCatalog struct {
	sync.Mutex
	servicesCache      map[mesh.ServiceName][]mesh.IP
	endpointsProviders []mesh.EndpointsProvider
	meshTopology       mesh.Topology
}
