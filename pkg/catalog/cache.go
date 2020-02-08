package catalog

import (
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log"
)

func (sc *MeshCatalog) refreshCache() {
	glog.Info("[catalog] Refresh cache...")
	servicesCache := make(map[endpoint.ServiceName][]endpoint.Endpoint)
	// TODO(draychev): split the namespace from the service name -- non-K8s services won't have namespace

	for _, namespacedServiceName := range sc.meshSpec.ListServices() {
		for _, provider := range sc.endpointsProviders {
			newIps := provider.ListEndpointsForService(namespacedServiceName)
			if len(newIps) == 0 {
				glog.Infof("[catalog][%s] No IPs found for service=%s", provider.GetID(), namespacedServiceName)
				continue
			}
			glog.V(log.LvlTrace).Infof("[catalog][%s] Found IPs=%+v for service=%s", provider.GetID(), endpointsToString(newIps), namespacedServiceName)
			if existingIps, exists := servicesCache[namespacedServiceName]; exists {
				servicesCache[namespacedServiceName] = append(existingIps, newIps...)
			} else {
				servicesCache[namespacedServiceName] = newIps
			}
		}
	}
	glog.Infof("[catalog] Services cache: %+v", servicesCache)
	sc.Lock()
	sc.servicesCache = servicesCache
	sc.Unlock()
}
