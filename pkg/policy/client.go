package policy

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
	policyInformers "github.com/openservicemesh/osm/pkg/gen/client/policy/informers/externalversions"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// kindSvcAccount is the ServiceAccount kind
	kindSvcAccount = "ServiceAccount"
)

// NewPolicyController returns a policy.Controller interface related to functionality provided by the resources in the policy.openservicemesh.io API group
func NewPolicyController(kubeController k8s.Controller, policyClient policyClientset.Interface, stop chan struct{}, msgBroker *messaging.Broker) (Controller, error) {
	return newClient(kubeController, policyClient, stop, msgBroker)
}

func newClient(kubeController k8s.Controller, policyClient policyClientset.Interface, stop chan struct{}, msgBroker *messaging.Broker) (client, error) {
	informerFactory := policyInformers.NewSharedInformerFactory(policyClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := informerCollection{
		egress:                 informerFactory.Policy().V1alpha1().Egresses().Informer(),
		ingressBackend:         informerFactory.Policy().V1alpha1().IngressBackends().Informer(),
		retry:                  informerFactory.Policy().V1alpha1().Retries().Informer(),
		upstreamTrafficSetting: informerFactory.Policy().V1alpha1().UpstreamTrafficSettings().Informer(),
	}

	cacheCollection := cacheCollection{
		egress:                 informerCollection.egress.GetStore(),
		ingressBackend:         informerCollection.ingressBackend.GetStore(),
		retry:                  informerCollection.retry.GetStore(),
		upstreamTrafficSetting: informerCollection.upstreamTrafficSetting.GetStore(),
	}

	client := client{
		informers:      &informerCollection,
		caches:         &cacheCollection,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return kubeController.IsMonitoredNamespace(object.GetNamespace())
	}

	egressEventTypes := k8s.EventTypes{
		Add:    announcements.EgressAdded,
		Update: announcements.EgressUpdated,
		Delete: announcements.EgressDeleted,
	}
	informerCollection.egress.AddEventHandler(k8s.GetEventHandlerFuncs(shouldObserve, egressEventTypes, msgBroker))
	ingressBackendEventTypes := k8s.EventTypes{
		Add:    announcements.IngressBackendAdded,
		Update: announcements.IngressBackendUpdated,
		Delete: announcements.IngressBackendDeleted,
	}
	informerCollection.ingressBackend.AddEventHandler(k8s.GetEventHandlerFuncs(shouldObserve, ingressBackendEventTypes, msgBroker))

	retryEventTypes := k8s.EventTypes{
		Add:    announcements.RetryPolicyAdded,
		Update: announcements.RetryPolicyUpdated,
		Delete: announcements.RetryPolicyDeleted,
	}
	informerCollection.retry.AddEventHandler(k8s.GetEventHandlerFuncs(shouldObserve, retryEventTypes, msgBroker))

	upstreamTrafficSettingEventTypes := k8s.EventTypes{
		Add:    announcements.UpstreamTrafficSettingAdded,
		Update: announcements.UpstreamTrafficSettingUpdated,
		Delete: announcements.UpstreamTrafficSettingDeleted,
	}
	informerCollection.upstreamTrafficSetting.AddEventHandler(k8s.GetEventHandlerFuncs(shouldObserve, upstreamTrafficSettingEventTypes, msgBroker))

	err := client.run(stop)
	if err != nil {
		return client, errors.Errorf("Could not start %s informer clients: %s", policyV1alpha1.SchemeGroupVersion, err)
	}

	return client, err
}

func (c client) run(stop <-chan struct{}) error {
	log.Info().Msgf("Starting informer clients for API group %s", policyV1alpha1.SchemeGroupVersion)

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[string]cache.SharedInformer{
		"Egress":                 c.informers.egress,
		"IngressBackend":         c.informers.ingressBackend,
		"Retry":                  c.informers.retry,
		"UpstreamTrafficSetting": c.informers.upstreamTrafficSetting,
	}

	var informerNames []string
	var hasSynced []cache.InformerSynced
	for name, informer := range sharedInformers {
		if informer == nil {
			log.Error().Msgf("Informer for '%s' not initialized, ignoring it", name) // TODO: log with errcode
			continue
		}
		informerNames = append(informerNames, name)
		log.Info().Msgf("Starting informer: %s", name)
		go informer.Run(stop)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	log.Info().Msgf("Waiting for informers %v caches to sync", informerNames)
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	log.Info().Msgf("Cache sync finished for %v informers in API group %s", informerNames, policyV1alpha1.SchemeGroupVersion)
	return nil
}

// ListEgressPoliciesForSourceIdentity lists the Egress policies for the given source identity based on service accounts
func (c client) ListEgressPoliciesForSourceIdentity(source identity.K8sServiceAccount) []*policyV1alpha1.Egress {
	var policies []*policyV1alpha1.Egress

	for _, egressIface := range c.caches.egress.List() {
		egressPolicy := egressIface.(*policyV1alpha1.Egress)

		if !c.kubeController.IsMonitoredNamespace(egressPolicy.Namespace) {
			continue
		}

		for _, sourceSpec := range egressPolicy.Spec.Sources {
			if sourceSpec.Kind == kindSvcAccount && sourceSpec.Name == source.Name && sourceSpec.Namespace == source.Namespace {
				policies = append(policies, egressPolicy)
			}
		}
	}

	return policies
}

// GetIngressBackendPolicy returns the IngressBackend policy for the given backend MeshService
func (c client) GetIngressBackendPolicy(svc service.MeshService) *policyV1alpha1.IngressBackend {
	for _, ingressBackendIface := range c.caches.ingressBackend.List() {
		ingressBackend := ingressBackendIface.(*policyV1alpha1.IngressBackend)

		if ingressBackend.Namespace != svc.Namespace {
			continue
		}

		// Return the first IngressBackend corresponding to the given MeshService.
		// Multiple IngressBackend policies for the same backend will be prevented
		// using a validating webhook.
		for _, backend := range ingressBackend.Spec.Backends {
			if backend.Name == svc.Name {
				return ingressBackend
			}
		}
	}

	return nil
}

// ListRetryPolicies returns the retry policies for the given source identity based on service accounts.
func (c client) ListRetryPolicies(source identity.K8sServiceAccount) []*policyV1alpha1.Retry {
	var retries []*policyV1alpha1.Retry

	for _, retryInterface := range c.caches.retry.List() {
		retry := retryInterface.(*policyV1alpha1.Retry)
		if !c.kubeController.IsMonitoredNamespace(retry.Namespace) {
			continue
		}
		if retry.Spec.Source.Kind == kindSvcAccount && retry.Spec.Source.Name == source.Name && retry.Spec.Source.Namespace == source.Namespace {
			retries = append(retries, retry)
		}
	}

	return retries
}

// GetUpstreamTrafficSetting returns the UpstreamTrafficSetting resource that matches the given options
func (c client) GetUpstreamTrafficSetting(options UpstreamTrafficSettingGetOpt) *policyV1alpha1.UpstreamTrafficSetting {
	if options.MeshService == nil && options.NamespacedName == nil {
		log.Error().Msgf("No option specified to get UpstreamTrafficSetting resource")
		return nil
	}

	if options.NamespacedName != nil {
		// Filter by namespaced name
		resource, exists, err := c.caches.upstreamTrafficSetting.GetByKey(options.NamespacedName.String())
		if exists && err == nil {
			return resource.(*policyV1alpha1.UpstreamTrafficSetting)
		}
		return nil
	}

	// Filter by MeshService
	for _, resource := range c.caches.upstreamTrafficSetting.List() {
		upstreamTrafficSetting := resource.(*policyV1alpha1.UpstreamTrafficSetting)

		if upstreamTrafficSetting.Namespace == options.MeshService.Namespace &&
			upstreamTrafficSetting.Spec.Host == options.MeshService.FQDN() {
			return upstreamTrafficSetting
		}
	}

	return nil
}
