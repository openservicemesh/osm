package catalog

import (
	"github.com/deislabs/smc/pkg/mesh"
	"github.com/deislabs/smc/pkg/mesh/providers"
	"github.com/deislabs/smc/pkg/providers/kube"
)

// Service is the struct for a service in the service catalog
type Service struct {
	name             mesh.ServiceName
	ips              []mesh.IP
	provider         mesh.ServiceProviderI
	kubernetesClient *kube.KubernetesProvider
}

// ServiceCatalog is the struct for the service catalog
type ServiceCatalog struct {
	servicesCache    map[mesh.ServiceName][]mesh.IP
	computeProviders map[providers.Provider]mesh.ComputeProviderI
	meshSpecProvider mesh.SpecI
}
