package config

import (
	"reflect"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/kubernetes"

	configV1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configV1alpha1Informers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"

	"github.com/openservicemesh/osm/pkg/announcements"
)

const (
	// apiGroup is the k8s API group that this package interacts with
	apiGroup = "config.openservicemesh.io"
)

// NewConfigController returns a config.Controller interface related to functionality provided by the resources in the config.openservicemesh.io API group
func NewConfigController(kubeConfig *rest.Config, kubeController kubernetes.Controller, stop chan struct{}) (Controller, error) {
	configClient := configV1alpha1Client.NewForConfigOrDie(kubeConfig)
	client, err := newConfigClient(
		configClient,
		kubeController,
		stop,
	)

	return client, err
}

// newConfigClient creates k8s clients for the resources in the config.openservicemesh.io API group
func newConfigClient(configClient configV1alpha1Client.Interface, kubeController kubernetes.Controller, stop chan struct{}) (client, error) {
	informerFactory := configV1alpha1Informers.NewSharedInformerFactory(configClient, kubernetes.DefaultKubeEventResyncInterval)

	informerCollection := informerCollection{
		remoteService: informerFactory.Config().V1alpha1().MultiClusterServices().Informer(),
	}

	cacheCollection := cacheCollection{
		remoteService: informerCollection.remoteService.GetStore(),
	}

	client := client{
		informers:      &informerCollection,
		caches:         &cacheCollection,
		cacheSynced:    make(chan interface{}),
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return kubeController.IsMonitoredNamespace(ns)
	}

	remoteServiceEventTypes := kubernetes.EventTypes{
		Add:    announcements.MultiClusterServiceAdded,
		Update: announcements.MultiClusterServiceUpdated,
		Delete: announcements.MultiClusterServiceDeleted,
	}
	informerCollection.remoteService.AddEventHandler(kubernetes.GetKubernetesEventHandlers("MultiClusterService", "Policy", shouldObserve, remoteServiceEventTypes))

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

	go c.informers.remoteService.Run(stop)

	log.Info().Msgf("Waiting for %s RemoteService informers' cache to sync", apiGroup)
	if !cache.WaitForCacheSync(stop, c.informers.remoteService.HasSynced) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	log.Info().Msgf("Cache sync finished for %s RemoteService informers", apiGroup)
	return nil
}
