package policy

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	policyV1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
	policyV1alpha1Informers "github.com/openservicemesh/osm/pkg/gen/client/policy/informers/externalversions"
	"github.com/openservicemesh/osm/pkg/service"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
)

const (
	// egressSourceKindSvcAccount is the ServiceAccount kind for a source defined in Egress policy
	egressSourceKindSvcAccount = "ServiceAccount"
)

// NewPolicyController returns a policy.Controller interface related to functionality provided by the resources in the policy.openservicemesh.io API group
func NewPolicyController(kubeConfig *rest.Config, kubeController k8s.Controller, stop chan struct{}) (Controller, error) {
	policyClient := policyV1alpha1Client.NewForConfigOrDie(kubeConfig)

	client, err := newPolicyClient(
		policyClient,
		kubeController,
		stop,
	)

	return client, err
}

// newPolicyClient creates k8s clients for the resources in the policy.openservicemesh.io API group
func newPolicyClient(policyClient policyV1alpha1Client.Interface, kubeController k8s.Controller, stop chan struct{}) (client, error) {
	informerFactory := policyV1alpha1Informers.NewSharedInformerFactory(policyClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := informerCollection{
		egress:         informerFactory.Policy().V1alpha1().Egresses().Informer(),
		ingressBackend: informerFactory.Policy().V1alpha1().IngressBackends().Informer(),
	}

	cacheCollection := cacheCollection{
		egress:         informerCollection.egress.GetStore(),
		ingressBackend: informerCollection.ingressBackend.GetStore(),
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
	informerCollection.egress.AddEventHandler(k8s.GetKubernetesEventHandlers("Egress", "Policy", shouldObserve, egressEventTypes))
	ingressBackendEventTypes := k8s.EventTypes{
		Add:    announcements.IngressBackendAdded,
		Update: announcements.IngressBackendUpdated,
		Delete: announcements.IngressBackendDeleted,
	}
	informerCollection.ingressBackend.AddEventHandler(k8s.GetKubernetesEventHandlers("IngressBackend", "Policy", shouldObserve, ingressBackendEventTypes))

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
		"Egress":         c.informers.egress,
		"IngressBackend": c.informers.ingressBackend,
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
			if sourceSpec.Kind == egressSourceKindSvcAccount && sourceSpec.Name == source.Name && sourceSpec.Namespace == source.Namespace {
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

		if !c.kubeController.IsMonitoredNamespace(ingressBackend.Namespace) {
			continue
		}

		// Return the first IngressBackend corresponding to the given MeshService.
		// Multiple IngressBackend policies for the same backend will be prevented
		// using a validating webhook.
		for _, backend := range ingressBackend.Spec.Backends {
			if ingressBackend.Namespace == svc.Namespace && backend.Name == svc.Name {
				return ingressBackend
			}
		}
	}

	return nil
}
