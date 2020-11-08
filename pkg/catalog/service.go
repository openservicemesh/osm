package catalog

import (
	"strings"

	"github.com/openservicemesh/osm/pkg/service"
)

func (mc *MeshCatalog) GetServiceAccountsForService(svc service.MeshService) ([]service.K8sServiceAccount, error) {
	var serviceAccounts []service.K8sServiceAccount
	for _, provider := range mc.endpointsProviders {
		if providerServiceAccounts, err := provider.GetServiceAccountsForService(svc); err != nil {
			log.Warn().Msgf("Error getting K8s Service Accounts linked to Service %s from provider %s: %s", provider.GetID(), svc, err)

		} else {
			log.Trace().Msgf("Found K8s Service Accounts %v linked to Service %s from endpoint provider %s", serviceAccounts, svc, provider.GetID())
			serviceAccounts = append(serviceAccounts, providerServiceAccounts...)

		}
	}
	return serviceAccounts, nil
}

// GetServicesForServiceAccounts returns a list of services corresponding to a list service accounts
func (mc *MeshCatalog) GetServicesForServiceAccounts(saList []service.K8sServiceAccount) []service.MeshService {

	serviceList := []service.MeshService{}

	for _, sa := range saList {
		services, err := mc.GetServicesForServiceAccount(sa)
		if err != nil {
			log.Error().Msgf("Error getting services linked to Service Account %s: %v", sa, err)
			continue
		}
		for _, svc := range services {
			serviceList = append(serviceList, svc)
		}
	}

	return serviceList
}

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
