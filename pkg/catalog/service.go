package catalog

import (
	"strings"

	"github.com/openservicemesh/osm/pkg/service"
)

// GetServicesForServiceAccount returns a list of services corresponding to a service account
func (mc *MeshCatalog) GetServicesForServiceAccount(sa service.K8sServiceAccount) ([]service.MeshService, error) {
	var services []service.MeshService
	for _, provider := range mc.endpointsProviders {
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

// ListServiceAccountsForService lists the service accounts associated with the given service
func (mc *MeshCatalog) ListServiceAccountsForService(svc service.MeshService) ([]service.K8sServiceAccount, error) {
	// Currently OSM uses kubernetes service accounts as service identities
	return mc.kubeController.ListServiceAccountsForService(svc)
}
