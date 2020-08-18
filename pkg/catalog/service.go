package catalog

import (
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
)

// GetServicesForServiceAccount returns a list of services corresponding to a service account
func (mc *MeshCatalog) GetServicesForServiceAccount(sa service.K8sServiceAccount) ([]service.MeshService, error) {
	var services []service.MeshService
	for _, provider := range mc.endpointsProviders {

		// TODO (#88) : remove this provider check once we have figured out the service account story for azure vms
		if provider.GetID() == constants.AzureProviderName {
			continue
		}

		log.Trace().Msgf("[%s] Looking for Services for Name=%s", provider.GetID(), sa)
		svc, err := provider.GetServiceForServiceAccount(sa)
		if err != nil {
			log.Warn().Msgf("Error getting services from provider %s: %s", provider.GetID(), err)
		} else {
			log.Trace().Msgf("[%s] Found service %s for Name=%s", provider.GetID(), svc, sa)
			services = append(services, svc)
		}

	}

	if len(services) == 0 {
		return nil, errServiceNotFoundForAnyProvider
	}

	return services, nil
}
