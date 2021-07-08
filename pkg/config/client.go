package config

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	configV1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configV1alpha1Informers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s"
)

const (
	// apiGroup is the k8s API group that this package interacts with
	apiGroup = "config.openservicemesh.io"
)

// NewConfigController returns a config.Controller struct related to functionality provided by the resources in the config.openservicemesh.io API group
func NewConfigController(kubeConfig *rest.Config, kubeController k8s.Controller, stop chan struct{}) (Controller, error) {
	configClient := configV1alpha1Client.NewForConfigOrDie(kubeConfig)
	informer := configV1alpha1Informers.NewSharedInformerFactory(configClient, k8s.DefaultKubeEventResyncInterval).Config().V1alpha1().MultiClusterServices().Informer()
	client := client{
		store:          informer.GetStore(),
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return kubeController.IsMonitoredNamespace(object.GetNamespace())
	}

	remoteServiceEventTypes := k8s.EventTypes{
		Add:    announcements.MultiClusterServiceAdded,
		Update: announcements.MultiClusterServiceUpdated,
		Delete: announcements.MultiClusterServiceDeleted,
	}
	informer.AddEventHandler(k8s.GetKubernetesEventHandlers("MultiClusterService", "Kube", shouldObserve, remoteServiceEventTypes))

	err := client.run(informer, stop)
	if err != nil {
		return client, errors.Errorf("Could not start %s client: %s", apiGroup, err)
	}

	return client, err
}

func (c client) run(informer cache.SharedIndexInformer, stop <-chan struct{}) error {
	log.Info().Msgf("%s client started", apiGroup)

	if informer == nil {
		return errInitInformers
	}

	go informer.Run(stop)

	log.Info().Msgf("Waiting for %s RemoteService informers' cache to sync", apiGroup)
	if !cache.WaitForCacheSync(stop, informer.HasSynced) {
		return errSyncingCaches
	}

	log.Info().Msgf("Cache sync finished for %s RemoteService informers", apiGroup)
	return nil
}

func (c client) ListMultiClusterServices() []*v1alpha1.MultiClusterService {
	var services []*v1alpha1.MultiClusterService

	for _, obj := range c.store.List() {
		mcs := obj.(*v1alpha1.MultiClusterService)
		if c.kubeController.IsMonitoredNamespace(mcs.Namespace) {
			services = append(services, mcs)
		}
	}
	return services
}

func (c client) GetMultiClusterService(name, namespace string) *v1alpha1.MultiClusterService {
	if !c.kubeController.IsMonitoredNamespace(namespace) {
		return nil
	}
	mcs, ok, err := c.store.GetByKey(namespace + "/" + name)
	if err != nil || !ok {
		log.Error().Err(err).Msgf("Error getting MultiClusterService %s in namespace %s from informer ", name, namespace)
		return nil
	}
	return mcs.(*v1alpha1.MultiClusterService)
}
