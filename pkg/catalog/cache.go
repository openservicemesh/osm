package catalog

import (
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

func (mc *MeshCatalog) refreshCache() {
	log.Info().Msg("Refresh cache...")
	serviceAccountToServicesCache := make(map[endpoint.NamespacedServiceAccount][]endpoint.NamespacedService)
	for _, namespacesServiceAccounts := range mc.meshSpec.ListServiceAccounts() {
		for _, provider := range mc.endpointsProviders {
			// TODO (snchh) : remove this provider check once we have figured out the service account story for azure vms
			if provider.GetID() != constants.AzureProviderName {
				log.Trace().Msgf("[%s] Looking for Services for ServiceAccount=%s", provider.GetID(), namespacesServiceAccounts)
				newServices := provider.ListServicesForServiceAccount(namespacesServiceAccounts)
				if len(newServices) == 0 {
					log.Trace().Msgf("[%s] No services found for ServiceAccount=%s", provider.GetID(), namespacesServiceAccounts)
					continue
				}
				log.Trace().Msgf("[%s] Found services=%+v for ServiceAccount=%s", provider.GetID(), newServices, namespacesServiceAccounts)
				serviceAccountToServicesCache[namespacesServiceAccounts] = newServices
			}
		}
	}
	log.Info().Msgf("ServiceAccountToServices cache: %+v", serviceAccountToServicesCache)
	mc.servicesMutex.Lock()
	mc.serviceAccountToServicesCache = serviceAccountToServicesCache
	mc.servicesMutex.Unlock()
}
