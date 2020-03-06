package catalog

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

// ListEndpoints constructs a map from service to weighted sub-services with all endpoints the given Envoy proxy should be aware of.
func (sc *MeshCatalog) ListEndpoints(clientID endpoint.NamespacedService) ([]endpoint.ServiceEndpoints, error) {
	glog.Info("[catalog] Listing Endpoints for client: ", clientID)
	// todo (sneha) : TBD if clientID is needed for filterning endpoints
	return sc.getWeightedEndpointsPerService(clientID)
}

func (sc *MeshCatalog) listEndpointsForService(service endpoint.Service) ([]endpoint.Endpoint, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	// todo (sneha) : TBD if clientID is needed for filterning endpoints
	glog.Infof("[catalog] listEndpointsForService %s", service.ServiceName)
	if _, found := sc.servicesCache[service]; !found {
		sc.refreshCache()
	}
	var endpoints []endpoint.Endpoint
	var found bool
	if endpoints, found = sc.servicesCache[service]; !found {
		glog.Errorf("[catalog] Did not find any Endpoints for service %s", service.ServiceName)
		return nil, errNotFound
	}
	glog.Infof("[catalog] Found Endpoints %s for service %s", strings.Join(endpointsToString(endpoints), ","), service.ServiceName)
	return endpoints, nil
}

func (sc *MeshCatalog) getWeightedEndpointsPerService(clientID endpoint.NamespacedService) ([]endpoint.ServiceEndpoints, error) {
	// todo (sneha) : TBD if clientID is needed for filterning endpoints
	var services []endpoint.ServiceEndpoints

	for _, trafficSplit := range sc.meshSpec.ListTrafficSplits() {
		glog.V(level.Trace).Infof("[Server][catalog] Discovered TrafficSplit resource: %s/%s\n", trafficSplit.Namespace, trafficSplit.Name)
		if trafficSplit.Spec.Backends == nil {
			glog.Errorf("[%s] TrafficSplit %s/%s has no Backends in Spec; Skipping...", packageName, trafficSplit.Namespace, trafficSplit.Name)
			continue
		}
		for _, trafficSplitBackend := range trafficSplit.Spec.Backends {
			// TODO(draychev): PULL THIS FROM SERVICE REGISTRY
			// svcName := mesh.ServiceName(fmt.Sprintf("%s/%s", trafficSplit.Namespace, trafficSplitBackend.ServiceName))
			namespacedServiceName := endpoint.NamespacedService{
				Namespace: trafficSplit.Namespace,
				Service:   trafficSplitBackend.Service,
			}
			serviceEndpoints := endpoint.ServiceEndpoints{}
			serviceEndpoints.Service = endpoint.Service{
				ServiceName: namespacedServiceName,
				Weight:      trafficSplitBackend.Weight,
			}
			var err error
			if serviceEndpoints.Endpoints, err = sc.listEndpointsForService(serviceEndpoints.Service); err != nil {
				glog.Errorf("[catalog] Error getting Endpoints for service %s: %s", namespacedServiceName, err)
				serviceEndpoints.Endpoints = []endpoint.Endpoint{}
			}
			services = append(services, serviceEndpoints)
		}
	}
	glog.V(level.Trace).Infof("[catalog] Constructed services with endpoints: %+v", services)
	return services, nil
}

func endpointsToString(endpoints []endpoint.Endpoint) []string {
	var epts []string
	for _, ept := range endpoints {
		epts = append(epts, fmt.Sprintf("%s:%d", ept.IP, ept.Port))
	}
	return epts
}
