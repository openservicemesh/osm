package policy

import (
	"reflect"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	policyV1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
	policyV1alpha1Informers "github.com/openservicemesh/osm/pkg/gen/client/policy/informers/externalversions"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
)

const (
	// apiGroup is the k8s API group that this package interacts with
	apiGroup = "policy.openservicemesh.io"

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
		egress: informerFactory.Policy().V1alpha1().Egresses().Informer(),
	}

	cacheCollection := cacheCollection{
		egress: informerCollection.egress.GetStore(),
	}

	client := client{
		informers:      &informerCollection,
		caches:         &cacheCollection,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return kubeController.IsMonitoredNamespace(ns)
	}

	egressEventTypes := k8s.EventTypes{
		Add:    announcements.EgressAdded,
		Update: announcements.EgressUpdated,
		Delete: announcements.EgressDeleted,
	}
	informerCollection.egress.AddEventHandler(k8s.GetKubernetesEventHandlers("Egress", "Policy", shouldObserve, egressEventTypes))

	err := client.run(stop)
	if err != nil {
		return client, errors.Errorf("Could not start %s client: %s", apiGroup, err)
	}

	return client, err
}

func (c client) run(stop <-chan struct{}) error {
	log.Info().Msgf("%s client started", apiGroup)

	if c.informers == nil {
		return errInitInformers
	}

	go c.informers.egress.Run(stop)

	log.Info().Msgf("Waiting for %s Egress informers' cache to sync", apiGroup)
	if !cache.WaitForCacheSync(stop, c.informers.egress.HasSynced) {
		return errSyncingCaches
	}

	log.Info().Msgf("Cache sync finished for %s Egress informers", apiGroup)
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
