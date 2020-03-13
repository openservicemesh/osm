package catalog

import (
	"github.com/golang/glog"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/log/level"
)

func (sc *MeshCatalog) refreshCache() {
	glog.Info("[catalog] Refresh cache...")
	servicesCache := make(map[endpoint.WeightedService][]endpoint.Endpoint)
	serviceAccountsCache := make(map[endpoint.NamespacedServiceAccount][]endpoint.NamespacedService)
	// TODO(draychev): split the namespace from the service name -- non-K8s services won't have namespace

	services := sc.meshSpec.ListServices()
	for _, service := range services {
		for _, provider := range sc.endpointsProviders {
			newIps := provider.ListEndpointsForService(endpoint.ServiceName(service.ServiceName.String()))
			if len(newIps) == 0 {
				glog.Infof("[catalog][%s] No IPs found for service=%s", provider.GetID(), service.ServiceName)
				continue
			}
			glog.V(level.Trace).Infof("[catalog][%s] Found IPs=%+v for service=%s", provider.GetID(), endpointsToString(newIps), service.ServiceName)
			if existingIps, exists := servicesCache[service]; exists {
				servicesCache[service] = append(existingIps, newIps...)
			} else {
				servicesCache[service] = newIps
			}
		}
	}

	for _, namespacesServiceAccounts := range sc.meshSpec.ListServiceAccounts() {
		for _, provider := range sc.endpointsProviders {
			// TODO (snchh) : remove this provider check once we have figured out the service account story for azure vms
			if provider.GetID() != constants.AzureProviderName {
				glog.V(level.Trace).Infof("[catalog][%s] Finding Services for servcie acccount =%s", provider.GetID(), namespacesServiceAccounts)
				newServices := provider.ListServicesForServiceAccount(namespacesServiceAccounts)
				if len(newServices) == 0 {
					glog.V(level.Trace).Infof("[catalog][%s] No services found for service account=%s", provider.GetID(), namespacesServiceAccounts)
					continue
				}
				glog.V(level.Trace).Infof("[catalog][%s] Found services=%+v for service account=%s", provider.GetID(), newServices, namespacesServiceAccounts)
				if existingServices, exists := serviceAccountsCache[namespacesServiceAccounts]; exists {
					// append only new services i.e. preventing duplication
					for _, service := range newServices {
						isPresent := false
						for _, existingService := range serviceAccountsCache[namespacesServiceAccounts] {
							if existingService.String() == service.String() {
								isPresent = true
							}
							if !isPresent {
								serviceAccountsCache[namespacesServiceAccounts] = append(existingServices, existingService)
							}
						}
					}
				} else {
					serviceAccountsCache[namespacesServiceAccounts] = newServices
				}
			}
		}
	}
	glog.Infof("[catalog] Services cache: %+v", servicesCache)
	glog.Infof("[catalog] ServiceAccounts cache: %+v", serviceAccountsCache)
	sc.servicesMutex.Lock()
	sc.servicesCache = servicesCache
	sc.serviceAccountsCache = serviceAccountsCache
	sc.servicesMutex.Unlock()
}
