package config

import (
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/constants"
)

func (c client) ListMultiClusterServices() []v1alpha1.MultiClusterService {
	var services []v1alpha1.MultiClusterService

	for _, obj := range c.informer.Informer().GetStore().List() {
		mcs := obj.(*v1alpha1.MultiClusterService)
		if c.kubeController.IsMonitoredNamespace(mcs.Namespace) {
			services = append(services, *mcs)
		}
	}

	log.Trace().Str(constants.LogFieldContext, constants.LogContextMulticluster).Msgf("All Multicluster services: %+v", services)
	return services
}

func (c client) GetMultiClusterServiceByServiceAccount(serviceAccount, namespace string) []v1alpha1.MultiClusterService {
	var services []v1alpha1.MultiClusterService

	for _, svc := range c.ListMultiClusterServices() {
		if svc.Spec.ServiceAccount == serviceAccount && svc.Namespace == namespace {
			services = append(services, svc)
		}
	}

	log.Trace().Str(constants.LogFieldContext, constants.LogContextMulticluster).Msgf("Multicluster services for svc account %s/%s: %+v", namespace, serviceAccount, services)
	return services
}

func (c client) GetMultiClusterService(name, namespace string) *v1alpha1.MultiClusterService {
	if !c.kubeController.IsMonitoredNamespace(namespace) {
		return nil
	}
	mcs, ok, err := c.informer.Informer().GetStore().GetByKey(namespace + "/" + name)
	if err != nil || !ok {
		log.Error().Str(constants.LogFieldContext, constants.LogContextMulticluster).Err(err).Msgf("Error getting MultiClusterService %s in namespace %s from informer ", name, namespace)
		return nil
	}
	return mcs.(*v1alpha1.MultiClusterService)
}
