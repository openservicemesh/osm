package catalog

import (
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
)

// GetServicesForServiceAccount returns a list of services corresponding to a service account
func (mc *MeshCatalog) GetServicesForServiceAccount(serviceAccount service.K8sServiceAccount) ([]service.MeshService, error) {
	var services []service.MeshService
	for _, provider := range mc.endpointsProviders {

		// TODO (#88) : remove this provider check once we have figured out the service account story for azure vms
		if provider.GetID() == constants.AzureProviderName {
			continue
		}

		if svc, err := provider.GetServiceForServiceAccount(serviceAccount); err != nil {
			log.Warn().Msgf("Error getting K8s Services linked to Service Account %s from provider %s: %s", provider.GetID(), serviceAccount, err)
		} else {
			log.Trace().Msgf("Found K8s Service %s linked to Service Account %s from endpoint provider %s", svc, serviceAccount, provider.GetID())
			services = append(services, svc)
		}
	}

	if len(services) == 0 {
		return nil, errServiceNotFoundForAnyProvider
	}

	return services, nil
}
