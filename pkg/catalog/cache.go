package catalog

import (
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log/level"
)

func (sc *MeshCatalog) refreshCache() {
	glog.Info("[catalog] Refresh cache...")
	servicesCache := make(map[endpoint.ServiceName][]endpoint.Endpoint)
	serviceAccountsCache := make(map[endpoint.ServiceAccount][]endpoint.ServiceName)
	// TODO(draychev): split the namespace from the service name -- non-K8s services won't have namespace

	services, targetServicesMap := sc.meshSpec.ListServices()
	for _, namespacedServiceName := range services {
		for _, provider := range sc.endpointsProviders {
			newIps := provider.ListEndpointsForService(namespacedServiceName)
			if len(newIps) == 0 {
				glog.Infof("[catalog][%s] No IPs found for service=%s", provider.GetID(), namespacedServiceName)
				continue
			}
			glog.V(level.Trace).Infof("[catalog][%s] Found IPs=%+v for service=%s", provider.GetID(), endpointsToString(newIps), namespacedServiceName)
			if existingIps, exists := servicesCache[namespacedServiceName]; exists {
				servicesCache[namespacedServiceName] = append(existingIps, newIps...)
			} else {
				servicesCache[namespacedServiceName] = newIps
			}
		}
	}

	for _, namespacesServiceAccounts := range sc.meshSpec.ListServiceAccounts() {
		for _, provider := range sc.endpointsProviders {
			// TODO (snchh) : remove this provider check once we have figured out the service account story for azure vms
			if provider.GetID() != "Azure" {
				glog.Infof("[catalog][%s] TEST Finding Services for servcie acccount =%s", provider.GetID(), namespacesServiceAccounts)
				newServices := provider.ListServicesForServiceAccount(namespacesServiceAccounts)
				if len(newServices) == 0 {
					glog.Infof("[catalog][%s] No services found for service account=%s", provider.GetID(), namespacesServiceAccounts)
					continue
				}
				glog.V(log.LvlTrace).Infof("[catalog][%s] Found services=%+v for service account=%s", provider.GetID(), newServices, namespacesServiceAccounts)
				if existingServices, exists := serviceAccountsCache[namespacesServiceAccounts]; exists {
					serviceAccountsCache[namespacesServiceAccounts] = append(existingServices, newServices...)
				} else {
					serviceAccountsCache[namespacesServiceAccounts] = newServices
				}
			}
		}
	}
	glog.Infof("[catalog] Services cache: %+v", servicesCache)
	glog.Infof("[catalog] ServiceAccounts cache: %+v", serviceAccountsCache)
	glog.Infof("[catalog] TargetServicesMap cache: %+v", targetServicesMap)
	sc.Lock()
	sc.servicesCache = servicesCache
	sc.serviceAccountsCache = serviceAccountsCache
	sc.targetServicesCache = targetServicesMap
	sc.Unlock()
}
