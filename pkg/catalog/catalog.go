package catalog

import (
	"fmt"
	"strings"

	"github.com/deislabs/smc/pkg/mesh/providers"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/mesh"
)

// NewServiceCatalog creates a new service catalog
func NewServiceCatalog(computeProviders map[providers.Provider]mesh.ComputeProviderI, meshSpecProvider mesh.SpecI) mesh.ServiceCatalogI {
	glog.Info("[catalog] Crete new Service Catalog...")
	return ServiceCatalog{
		servicesCache:    make(map[mesh.ServiceName][]mesh.IP),
		computeProviders: computeProviders,
		meshSpecProvider: meshSpecProvider,
	}
}

// GetWeightedService gets the backing delegated services for the given target service and their weights.
func (sc ServiceCatalog) GetWeightedService(svcName mesh.ServiceName) ([]mesh.WeightedService, error) {
	var weightedServices []mesh.WeightedService
	for _, split := range sc.meshSpecProvider.ListTrafficSplits() {
		if mesh.ServiceName(split.Spec.Service) == svcName {
			for _, backend := range split.Spec.Backends {
				namespaced := fmt.Sprintf("%s/%s", split.Namespace, backend.Service)
				if ips, err := sc.GetServiceIPs(mesh.ServiceName(namespaced)); err != nil {
					ws := mesh.WeightedService{
						ServiceName: mesh.ServiceName(backend.Service),
						Weight:      backend.Weight,
						IPs:         ips,
					}
					weightedServices = append(weightedServices, ws)
				}
			}
		}
	}
	return weightedServices, nil
}

// GetWeightedServices gets all services and their delegated services and weights
func (sc ServiceCatalog) GetWeightedServices() (map[mesh.ServiceName][]mesh.WeightedService, error) {
	glog.Info("[catalog] GetWeightedServices")
	byTargetService := make(map[mesh.ServiceName][]mesh.WeightedService) // TODO  trafficSplit name must match Envoy's cluster name
	backendWeight := make(map[string]int)

	for _, trafficSplit := range sc.meshSpecProvider.ListTrafficSplits() {
		targetServiceName := mesh.ServiceName(trafficSplit.Spec.Service)
		var services []mesh.WeightedService
		glog.V(7).Infof("[EDS] Discovered TrafficSplit resource: %s/%s for service %s\n", trafficSplit.Namespace, trafficSplit.Name, targetServiceName)
		if trafficSplit.Spec.Backends == nil {
			glog.Errorf("[EDS] TrafficSplit %s/%s has no Backends in Spec; Skipping...", trafficSplit.Namespace, trafficSplit.Name)
			continue
		}
		for _, trafficSplitBackend := range trafficSplit.Spec.Backends {
			// TODO(draychev): PULL THIS FROM SERVICE REGISTRY
			// svcName := mesh.ServiceName(fmt.Sprintf("%s/%s", trafficSplit.Namespace, trafficSplitBackend.ServiceName))
			backendWeight[trafficSplitBackend.Service] = trafficSplitBackend.Weight
			weightedService := mesh.WeightedService{}
			weightedService.ServiceName = mesh.ServiceName(trafficSplitBackend.Service)
			weightedService.Weight = trafficSplitBackend.Weight
			var err error
			namespaced := fmt.Sprintf("%s/%s", trafficSplit.Namespace, trafficSplitBackend.Service)
			if weightedService.IPs, err = sc.GetServiceIPs(mesh.ServiceName(namespaced)); err != nil {
				glog.Errorf("[catalog] Error getting IPs for service %s: %s", namespaced, err)
				weightedService.IPs = []mesh.IP{}
			}
			services = append(services, weightedService)
		}
		byTargetService[targetServiceName] = services
	}
	return byTargetService, nil
}

// GetServiceIPs retrieves the IP addresses for the given service
func (sc ServiceCatalog) GetServiceIPs(namespacedServiceName mesh.ServiceName) ([]mesh.IP, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	glog.Infof("[catalog] GetServiceIPs %s", namespacedServiceName)
	if _, found := sc.servicesCache[namespacedServiceName]; !found {
		sc.refreshCache()
	}
	var ips []mesh.IP
	var found bool
	if ips, found = sc.servicesCache[namespacedServiceName]; !found {
		glog.Errorf("[catalog] Did not find any IPs for service %s", namespacedServiceName)
		return nil, errNotFound
	}
	glog.Infof("[catalog] Found IPs %s for service %s", strings.Join(ipsToString(ips), ","), namespacedServiceName)
	return ips, nil
}

func (sc *ServiceCatalog) refreshCache() {
	glog.Info("[catalog] Refresh cache...")
	servicesCache := make(map[mesh.ServiceName][]mesh.IP)
	for providerType, provider := range sc.computeProviders {
		// TODO(draychev): split the namespace from the service name -- non-K8s services won't have namespace
		for _, namespacedServiceName := range sc.meshSpecProvider.ListServices() {
			newIps := provider.GetIPs(namespacedServiceName)
			if providerName, err := providers.GetFriendlyName(providerType); err != nil {
				glog.Infof("[catalog] Found ips=%+v for service=%s for provider=%s", ipsToString(newIps), namespacedServiceName, providerName)
			} else {
				glog.Infof("[catalog] Found ips=%+v for service=%s for provider=%d", ipsToString(newIps), namespacedServiceName, providerType)
			}

			if existingIps, exists := servicesCache[namespacedServiceName]; exists {
				servicesCache[namespacedServiceName] = append(existingIps, newIps...)
			} else {
				servicesCache[namespacedServiceName] = newIps
			}
		}
	}
	glog.Infof("[catalog] Services cache: %+v", servicesCache)
	sc.servicesCache = servicesCache
}

func ipsToString(meshIPs []mesh.IP) []string {
	var ips []string
	for _, ip := range meshIPs {
		ips = append(ips, string(ip))
	}
	return ips
}
