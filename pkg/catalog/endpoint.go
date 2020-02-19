package catalog

import (
	"fmt"
	"strings"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/smi"
)

func (sc *MeshCatalog) listEndpointsForService(namespacedServiceName endpoint.ServiceName) ([]endpoint.Endpoint, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	glog.Infof("[catalog] listEndpointsForService %s", namespacedServiceName)
	if _, found := sc.servicesCache[namespacedServiceName]; !found {
		sc.refreshCache()
	}
	var endpoints []endpoint.Endpoint
	var found bool
	if endpoints, found = sc.servicesCache[namespacedServiceName]; !found {
		glog.Errorf("[catalog] Did not find any Endpoints for service %s", namespacedServiceName)
		return nil, errNotFound
	}
	glog.Infof("[catalog] Found Endpoints %s for service %s", strings.Join(endpointsToString(endpoints), ","), namespacedServiceName)
	return endpoints, nil
}

// ListEndpoints constructs a map from service to weighted sub-services with all endpoints the given Envoy proxy should be aware of.
func (sc *MeshCatalog) ListEndpoints(clientID smi.ClientIdentity) (map[endpoint.ServiceName][]endpoint.WeightedService, error) {
	glog.Info("[catalog] Listing Endpoints for client: ", clientID)
	byTargetService := make(map[endpoint.ServiceName][]endpoint.WeightedService)
	backendWeight := make(map[string]int)

	for _, trafficSplit := range sc.meshSpec.ListTrafficSplits() {
		targetServiceName := endpoint.ServiceName(fmt.Sprintf("%s/%s", trafficSplit.Namespace, trafficSplit.Spec.Service))
		var services []endpoint.WeightedService
		glog.V(level.Trace).Infof("[catalog] Discovered TrafficSplit resource: %s/%s for service %s\n", trafficSplit.Namespace, trafficSplit.Name, targetServiceName)
		if trafficSplit.Spec.Backends == nil {
			glog.Errorf("[catalog] TrafficSplit %s/%s has no Backends in Spec; Skipping...", trafficSplit.Namespace, trafficSplit.Name)
			continue
		}
		for _, trafficSplitBackend := range trafficSplit.Spec.Backends {
			// TODO(draychev): PULL THIS FROM SERVICE REGISTRY
			// svcName := mesh.ServiceName(fmt.Sprintf("%s/%s", trafficSplit.Namespace, trafficSplitBackend.ServiceName))
			namespaced := fmt.Sprintf("%s/%s", trafficSplit.Namespace, trafficSplitBackend.Service)
			backendWeight[trafficSplitBackend.Service] = trafficSplitBackend.Weight
			weightedService := endpoint.WeightedService{}
			weightedService.ServiceName = endpoint.ServiceName(namespaced)
			weightedService.Weight = trafficSplitBackend.Weight
			var err error
			if weightedService.Endpoints, err = sc.listEndpointsForService(endpoint.ServiceName(namespaced)); err != nil {
				glog.Errorf("[catalog] Error getting Endpoints for service %s: %s", namespaced, err)
				weightedService.Endpoints = []endpoint.Endpoint{}
			}
			services = append(services, weightedService)
		}
		byTargetService[targetServiceName] = services
	}
	glog.V(level.Trace).Infof("[catalog] Constructed weighted services: %+v", byTargetService)
	return byTargetService, nil
}

func endpointsToString(endpoints []endpoint.Endpoint) []string {
	var epts []string
	for _, ept := range endpoints {
		epts = append(epts, fmt.Sprintf("%s:%d", ept.IP, ept.Port))
	}
	return epts
}
