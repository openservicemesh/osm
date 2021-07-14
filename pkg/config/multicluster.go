package config

import (
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
)

func (c client) ListMultiClusterServices() []*v1alpha1.MultiClusterService {
	var services []*v1alpha1.MultiClusterService

	for _, obj := range c.informer.Informer().GetStore().List() {
		mcs := obj.(*v1alpha1.MultiClusterService)
		if c.kubeController.IsMonitoredNamespace(mcs.Namespace) {
			services = append(services, mcs)
		}
	}

	log.Trace().Msgf("All Multicluster services: %+v", services)

	return services
}

func (c client) GetMultiClusterServiceByServiceAccount(serviceAccount, namespace string) []*v1alpha1.MultiClusterService {
	if !c.kubeController.IsMonitoredNamespace(namespace) {
		return nil
	}

	var services []*v1alpha1.MultiClusterService

	for _, svc := range c.ListMultiClusterServices() {
		if svc.Spec.ServiceAccount == serviceAccount && svc.Namespace == namespace {
			services = append(services, svc)
		}
	}

	log.Trace().Msgf("Multicluster services for svc account %s/%s: %+v", namespace, serviceAccount, services)

	return services
}
