package catalog

import (
	"fmt"
	"strings"
	"time"

	envoy "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/mesh"
)

// NewServiceCatalog creates a new service catalog
func NewServiceCatalog(meshTopology mesh.Topology, stopChan chan struct{}, endpointsProviders ...mesh.EndpointsProvider) ServiceCataloger {
	// Run each provider -- starting the pub/sub system, which leverages the announceChan channel
	for _, provider := range endpointsProviders {
		if err := provider.Run(stopChan); err != nil {
			glog.Errorf("Could not start %s provider: %s", provider.GetID(), err)
			continue
		}
		glog.Infof("Started provider %s", provider.GetID())
	}
	glog.Info("[catalog] Create a new Service Catalog.")
	serviceCatalog := ServiceCatalog{
		servicesCache:      make(map[mesh.ServiceName][]mesh.IP),
		endpointsProviders: endpointsProviders,
		meshTopology:       meshTopology,
	}

	// NOTE(draychev): helpful while developing alpha MVP -- remove before releasing beta version.
	go func() {
		counter := 0
		for {
			glog.V(7).Infof("------------------------- Service Catalog Cache Refresh %d -------------------------", counter)
			counter++
			serviceCatalog.refreshCache()
			time.Sleep(5 * time.Second)
		}
	}()
	return &serviceCatalog
}

// GetWeightedService gets the backing delegated services for the given target service and their weights.
func (sc *ServiceCatalog) GetWeightedService(svcName mesh.ServiceName) ([]mesh.WeightedService, error) {
	var weightedServices []mesh.WeightedService
	for _, split := range sc.meshTopology.ListTrafficSplits() {
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
func (sc *ServiceCatalog) GetWeightedServices() (map[mesh.ServiceName][]mesh.WeightedService, error) {
	sc.Lock()
	defer sc.Unlock()
	glog.Info("[catalog] GetWeightedServices")
	byTargetService := make(map[mesh.ServiceName][]mesh.WeightedService) // TODO  trafficSplit name must match Envoy's cluster name
	backendWeight := make(map[string]int)

	for _, trafficSplit := range sc.meshTopology.ListTrafficSplits() {
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
func (sc *ServiceCatalog) GetServiceIPs(namespacedServiceName mesh.ServiceName) ([]mesh.IP, error) {
	sc.Lock()
	defer sc.Unlock()
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
	// TODO(draychev): split the namespace from the service name -- non-K8s services won't have namespace
	for _, namespacedServiceName := range sc.meshTopology.ListServices() {
		for _, provider := range sc.endpointsProviders {
			newIps := provider.GetIPs(namespacedServiceName)
			glog.Infof("[catalog] Found ips=%+v for service=%s for provider=%s", ipsToString(newIps), namespacedServiceName, provider.GetID())
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

func ipsToString(meshIPs []mesh.IP) []string {
	var ips []string
	for _, ip := range meshIPs {
		ips = append(ips, string(ip))
	}
	return ips
}

// ListEndpoints constructs a DiscoveryResponse with all endpoints the given Envoy proxy should be aware of.
// The bool return value indicates whether there have been any changes since the last invocation of this function.
func (sc *ServiceCatalog) ListEndpoints(mesh.ClientIdentity) (envoy.DiscoveryResponse, bool, error) {
	// TODO(draychev): implement
	panic("NotImplemented")
}

// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
func (sc *ServiceCatalog) RegisterNewEndpoint(mesh.ClientIdentity) {
	// TODO(draychev): implement
	panic("NotImplemented")
}

// ListEndpointsProviders retrieves the full list of endpoints providers registered with Service Catalog so far.
func (sc *ServiceCatalog) ListEndpointsProviders() []mesh.EndpointsProvider {
	// TODO(draychev): implement
	panic("NotImplemented")
}

// RegisterEndpointsProvider adds a new endpoints provider to the list within the Service Catalog.
func (sc *ServiceCatalog) RegisterEndpointsProvider(mesh.EndpointsProvider) error {
	// TODO(draychev): implement
	panic("NotImplemented")
}

// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
func (sc *ServiceCatalog) GetAnnouncementChannel() chan struct{} {
	// TODO(draychev): implement
	panic("NotImplemented")
}
