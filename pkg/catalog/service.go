package catalog

import (
	"strings"

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

		if providerServices, err := provider.GetServicesForServiceAccount(sa); err != nil {
			log.Warn().Msgf("Error getting K8s Services linked to Service Account %s from provider %s: %s", provider.GetID(), sa, err)
		} else {
			var svcs []string
			for _, svc := range providerServices {
				svcs = append(svcs, svc.String())
			}

			log.Trace().Msgf("Found K8s Services %s linked to Service Account %s from endpoint provider %s", strings.Join(svcs, ","), sa, provider.GetID())
			services = append(services, providerServices...)
		}
	}

	if len(services) == 0 {
		return nil, errServiceNotFoundForAnyProvider
	}

	return services, nil
}
