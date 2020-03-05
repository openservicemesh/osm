package catalog

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

func (sc *MeshCatalog) listEndpointsForService(namespacedServiceName endpoint.ServiceName) ([]endpoint.Endpoint, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	glog.Infof("[%s] listEndpointsForService %s", packageName, namespacedServiceName)
	if _, found := sc.servicesCache[namespacedServiceName]; !found {
		sc.refreshCache()
	}
	var endpoints []endpoint.Endpoint
	var found bool
	if endpoints, found = sc.servicesCache[namespacedServiceName]; !found {
		glog.Errorf("[%s] Did not find any Endpoints for service %s", packageName, namespacedServiceName)
		return nil, errNotFound
	}
	glog.Infof("[%s] Found Endpoints %s for service %s", packageName, strings.Join(endpointsToString(endpoints), ","), namespacedServiceName)
	return endpoints, nil
}

// ListEndpoints constructs a map from service to weighted sub-services with all endpoints the given Envoy proxy should be aware of.
func (sc *MeshCatalog) ListEndpoints(clientID smi.ClientIdentity) (map[endpoint.ServiceName][]endpoint.WeightedService, error) {
	glog.Infof("[%s] Listing Endpoints for client: %s", packageName, clientID)
	byTargetService := make(map[endpoint.ServiceName][]endpoint.WeightedService)
	backendWeight := make(map[string]int)

	for _, trafficSplit := range sc.meshSpec.ListTrafficSplits() {
		namespacedTargerServiceName := endpoint.NamespacedService{
			Namespace: trafficSplit.Namespace,
			Service:   trafficSplit.Spec.Service,
		}
		targetServiceName := endpoint.ServiceName(namespacedTargerServiceName.String())
		var services []endpoint.WeightedService
		glog.V(level.Trace).Infof("[%s] Discovered TrafficSplit resource: %s/%s for service %s", packageName, trafficSplit.Namespace, trafficSplit.Name, targetServiceName)
		if trafficSplit.Spec.Backends == nil {
			glog.Errorf("[%s] TrafficSplit %s/%s has no Backends in Spec; Skipping...", packageName, trafficSplit.Namespace, trafficSplit.Name)
			continue
		}
		for _, trafficSplitBackend := range trafficSplit.Spec.Backends {
			// TODO(draychev): PULL THIS FROM SERVICE REGISTRY
			// svcName := mesh.ServiceName(fmt.Sprintf("%s/%s", trafficSplit.Namespace, trafficSplitBackend.ServiceName))
			namespaced := endpoint.NamespacedService{
				Namespace: trafficSplit.Namespace,
				Service:   trafficSplitBackend.Service,
			}
			backendWeight[trafficSplitBackend.Service] = trafficSplitBackend.Weight
			weightedService := endpoint.WeightedService{}
			weightedService.ServiceName = endpoint.ServiceName(namespaced.String())
			weightedService.Weight = trafficSplitBackend.Weight
			var err error
			if weightedService.Endpoints, err = sc.listEndpointsForService(endpoint.ServiceName(namespaced.String())); err != nil {
				glog.Errorf("[%s] Error getting Endpoints for service %s: %s", packageName, namespaced.String(), err)
				weightedService.Endpoints = []endpoint.Endpoint{}
			}
			services = append(services, weightedService)
		}
		byTargetService[targetServiceName] = services
	}
	glog.V(level.Trace).Infof("[%s] Constructed weighted services: %+v", packageName, byTargetService)
	return byTargetService, nil
}

func endpointsToString(endpoints []endpoint.Endpoint) []string {
	var epts []string
	for _, ept := range endpoints {
		epts = append(epts, fmt.Sprintf("%s:%d", ept.IP, ept.Port))
	}
	return epts
}
