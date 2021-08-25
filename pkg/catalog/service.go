package catalog

import (
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// getServicesForServiceIdentity returns a list of services corresponding to a service identity
func (mc *MeshCatalog) getServicesForServiceIdentity(svcIdentity identity.ServiceIdentity) ([]service.MeshService, error) {
	var services []service.MeshService

	for _, provider := range mc.serviceProviders {
		providerServices, err := provider.GetServicesForServiceIdentity(svcIdentity)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting K8s Services linked to Service Identity %s from provider %s", svcIdentity, provider.GetID())
			continue
		}

		log.Trace().Msgf("Found Services %v linked to Service Identity %s from provider %s", providerServices, svcIdentity, provider.GetID())
		services = append(services, providerServices...)
	}

	if len(services) == 0 {
		return nil, errServiceNotFoundForAnyProvider
	}

	return services, nil
}

// ListServiceIdentitiesForService lists the service identities associated with the given mesh service.
func (mc *MeshCatalog) ListServiceIdentitiesForService(svc service.MeshService) ([]identity.ServiceIdentity, error) {
	// Currently OSM uses kubernetes service accounts as service identities
	var serviceIdentities []identity.ServiceIdentity
	for _, provider := range mc.serviceProviders {
		serviceIDs, err := provider.ListServiceIdentitiesForService(svc)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingServiceIdentitiesForService)).
				Msgf("Error getting ServiceIdentities for Service %s", svc)
			return nil, err
		}

		serviceIdentities = append(serviceIdentities, serviceIDs...)
	}

	return serviceIdentities, nil
}

// listMeshServices returns all services in the mesh
func (mc *MeshCatalog) listMeshServices() []service.MeshService {
	var services []service.MeshService

	for _, provider := range mc.serviceProviders {
		svcs, err := provider.ListServices()
		if err != nil {
			log.Error().Err(err).Msgf("Error listing services for provider %s", provider.GetID())
			continue
		}

		services = append(services, svcs...)
	}

	return services
}
