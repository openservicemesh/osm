package catalog

import (
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// getServicesForServiceIdentity returns a list of services corresponding to a service identity
func (mc *MeshCatalog) getServicesForServiceIdentity(svcIdentity identity.ServiceIdentity) []service.MeshService {
	var services []service.MeshService

	for _, provider := range mc.serviceProviders {
		providerServices := provider.GetServicesForServiceIdentity(svcIdentity)
		log.Trace().Msgf("Found Services %v linked to Service Identity %s from provider %s", providerServices, svcIdentity, provider.GetID())
		services = append(services, providerServices...)
	}

	return services
}

// ListServiceIdentitiesForService lists the service identities associated with the given mesh service.
func (mc *MeshCatalog) ListServiceIdentitiesForService(svc service.MeshService) []identity.ServiceIdentity {
	// Currently OSM uses kubernetes service accounts as service identities
	var serviceIdentities []identity.ServiceIdentity
	for _, provider := range mc.serviceProviders {
		serviceIDs := provider.ListServiceIdentitiesForService(svc)
		serviceIdentities = append(serviceIdentities, serviceIDs...)
	}

	return serviceIdentities
}

// listMeshServices returns all services in the mesh
func (mc *MeshCatalog) listMeshServices() []service.MeshService {
	var services []service.MeshService

	for _, provider := range mc.serviceProviders {
		svcs := provider.ListServices()
		// TODO: handle duplicates when the codebase correctly supports multiple service providers
		services = append(services, svcs...)
	}

	return services
}
