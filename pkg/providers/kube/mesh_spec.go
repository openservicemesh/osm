package kube

import (
	"fmt"

	"github.com/deislabs/smc/pkg/mesh"
	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	"github.com/golang/glog"
)

// GetTrafficSplitWeight retrieves the weight for the given service
func (kp *KubernetesProvider) GetTrafficSplitWeight(target mesh.ServiceName, delegate mesh.ServiceName) (int, error) {
	fmt.Printf("Here is kp: %+v", kp)
	fmt.Printf("Here is kp.Caches: %+v", kp.Caches)
	fmt.Printf("Here is kp.Caches.TrafficSplit: %+v", kp.Caches.TrafficSplit)
	item, exists, err := kp.Caches.TrafficSplit.Get(target)
	if err != nil {
		glog.Errorf("[kubernetes] Error retrieving %v from TrafficSplit cache", target)
		return 0, errRetrievingFromCache
	}
	if !exists {
		glog.Errorf("[kubernetes] %v does not exist in TrafficSplit cache", target)
		return 0, errNotInCache
	}
	ts := item.(v1alpha2.TrafficSplit)
	for _, be := range ts.Spec.Backends {
		if be.Service == string(delegate) {
			return be.Weight, nil
		}
	}
	glog.Errorf("[kubernetes] Was looking for delegate %s for target service %s but did not find it", delegate, target)
	return 0, errBackendNotFound
}

// ListTrafficSplits returns the list of traffic splits.
func (kp *KubernetesProvider) ListTrafficSplits() []*v1alpha2.TrafficSplit {
	var trafficSplits []*v1alpha2.TrafficSplit
	for _, splitIface := range kp.Caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		trafficSplits = append(trafficSplits, split)
	}
	return trafficSplits
}

// ListServices lists the services observed from the given compute provider
func (kp *KubernetesProvider) ListServices() []mesh.ServiceName {
	// TODO(draychev): split the namespace and the service name -- for non-kubernetes services we won't have namespace
	var services []mesh.ServiceName
	for _, splitIface := range kp.Caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, split.Spec.Service)
		services = append(services, mesh.ServiceName(namespacedServiceName))
		for _, backend := range split.Spec.Backends {
			namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, backend.Service)
			services = append(services, mesh.ServiceName(namespacedServiceName))
		}
	}
	return services
}
