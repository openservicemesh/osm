package catalog

import (
	"fmt"
	"strings"
	"time"

	envoy "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	protobufTypes "github.com/gogo/protobuf/types"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy/cla"
	"github.com/deislabs/smc/pkg/mesh"
)

// NewServiceCatalog creates a new service catalog
func NewServiceCatalog(meshTopology mesh.Topology, endpointsProviders ...endpoint.Provider) ServiceCataloger {
	glog.Info("[catalog] Create a new Service Catalog.")
	serviceCatalog := ServiceCatalog{
		servicesCache:      make(map[mesh.ServiceName][]mesh.Endpoint),
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

// ListEndpoints constructs a DiscoveryResponse with all endpoints the given Envoy proxy should be aware of.
// The bool return value indicates whether there have been any changes since the last invocation of this function.
func (sc *ServiceCatalog) ListEndpoints(clientID mesh.ClientIdentity) (*envoy.DiscoveryResponse, bool, error) {
	glog.Info("Listing Endpoints for client: ", clientID)
	allServices, err := sc.getWeightedEndpointsPerService()
	if err != nil {
		glog.Error("Could not refresh weighted services: ", err)
		return nil, false, err
	}
	var protos []*protobufTypes.Any
	for targetServiceName, weightedServices := range allServices {
		loadAssignment := cla.NewClusterLoadAssignment(targetServiceName, weightedServices)

		proto, err := protobufTypes.MarshalAny(&loadAssignment)
		if err != nil {
			glog.Errorf("Error marshalling ClusterLoadAssignmentURI %+v: %s", loadAssignment, err)
			continue
		}
		protos = append(protos, proto)
		/*
				// TODO(draychev): this needs to happen per Envoy proxy - not for all of them
				sc.lastVersion = e.lastVersion + 1
				e.lastNonce = string(time.Now().Nanosecond())

			resp.Nonce = e.lastNonce
			resp.VersionInfo = fmt.Sprintf("v%d", e.lastVersion)
		*/

	}

	resp := &envoy.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   cla.ClusterLoadAssignmentURI,
	}

	return resp, false, nil
}

func (sc *ServiceCatalog) refreshCache() {
	glog.Info("[catalog] Refresh cache...")
	servicesCache := make(map[mesh.ServiceName][]mesh.Endpoint)
	// TODO(draychev): split the namespace from the service name -- non-K8s services won't have namespace
	for _, namespacedServiceName := range sc.meshTopology.ListServices() {
		for _, provider := range sc.endpointsProviders {
			newIps := provider.ListEndpointsForService(namespacedServiceName)
			glog.V(7).Infof("[catalog][%s] Found ips=%+v for service=%s", provider.GetID(), endpointsToString(newIps), namespacedServiceName)
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

func endpointsToString(endpoints []mesh.Endpoint) []string {
	var epts []string
	for _, ept := range endpoints {
		epts = append(epts, fmt.Sprintf("%s:%d", ept.IP, ept.Port))
	}
	return epts
}

func (sc *ServiceCatalog) getWeightedEndpointsPerService() (map[mesh.ServiceName][]mesh.WeightedService, error) {
	byTargetService := make(map[mesh.ServiceName][]mesh.WeightedService)
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
			if weightedService.Endpoints, err = sc.listEndpointsForService(mesh.ServiceName(namespaced)); err != nil {
				glog.Errorf("[catalog] Error getting Endpoints for service %s: %s", namespaced, err)
				weightedService.Endpoints = []mesh.Endpoint{}
			}
			services = append(services, weightedService)
		}
		byTargetService[targetServiceName] = services
	}
	glog.V(7).Infof("[catalog] Constructed weighted services: %+v", byTargetService)
	return byTargetService, nil
}

func (sc *ServiceCatalog) listEndpointsForService(namespacedServiceName mesh.ServiceName) ([]mesh.Endpoint, error) {
	sc.Lock()
	defer sc.Unlock()
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	glog.Infof("[catalog] listEndpointsForService %s", namespacedServiceName)
	if _, found := sc.servicesCache[namespacedServiceName]; !found {
		sc.refreshCache()
	}
	var endpoints []mesh.Endpoint
	var found bool
	if endpoints, found = sc.servicesCache[namespacedServiceName]; !found {
		glog.Errorf("[catalog] Did not find any Endpoints for service %s", namespacedServiceName)
		return nil, errNotFound
	}
	glog.Infof("[catalog] Found Endpoints %s for service %s", strings.Join(endpointsToString(endpoints), ","), namespacedServiceName)
	return endpoints, nil
}

// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
func (sc *ServiceCatalog) RegisterNewEndpoint(mesh.ClientIdentity) {
	// TODO(draychev): implement
	panic("NotImplemented")
}

// ListEndpointsProviders retrieves the full list of endpoints providers registered with Service Catalog so far.
func (sc *ServiceCatalog) ListEndpointsProviders() []endpoint.Provider {
	// TODO(draychev): implement
	panic("NotImplemented")
}

// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
func (sc *ServiceCatalog) GetAnnouncementChannel() chan struct{} {
	// TODO(draychev): implement
	panic("NotImplemented")
}
