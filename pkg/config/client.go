package config

import (
	"reflect"

	"github.com/pkg/errors"
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
	client, err := newConfigClient(
		configClient,
		kubeController,
		stop,
	)

	return client, err
}

// newConfigClient creates k8s clients for the resources in the config.openservicemesh.io API group
func newConfigClient(configClient configV1alpha1Client.Interface, kubeController k8s.Controller, stop chan struct{}) (client, error) {
	informerFactory := configV1alpha1Informers.NewSharedInformerFactory(configClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := informerCollection{
		multiClusterService: informerFactory.Config().V1alpha1().MultiClusterServices().Informer(),
	}

	cacheCollection := cacheCollection{
		multiClusterService: informerCollection.multiClusterService.GetStore(),
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

	remoteServiceEventTypes := k8s.EventTypes{
		Add:    announcements.MultiClusterServiceAdded,
		Update: announcements.MultiClusterServiceUpdated,
		Delete: announcements.MultiClusterServiceDeleted,
	}
	informerCollection.multiClusterService.AddEventHandler(k8s.GetKubernetesEventHandlers("MultiClusterService", "Kube", shouldObserve, remoteServiceEventTypes))

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

	go c.informers.multiClusterService.Run(stop)

	log.Info().Msgf("Waiting for %s RemoteService informers' cache to sync", apiGroup)
	if !cache.WaitForCacheSync(stop, c.informers.multiClusterService.HasSynced) {
		return errSyncingCaches
	}

	log.Info().Msgf("Cache sync finished for %s RemoteService informers", apiGroup)
	return nil
}

func (c client) ListMultiClusterServices() []*v1alpha1.MultiClusterService {
	var services []*v1alpha1.MultiClusterService

	for _, obj := range c.caches.multiClusterService.List() {
		mcservice := obj.(*v1alpha1.MultiClusterService)
		services = append(services, mcservice)
	}
	return services
}

func (c client) GetMultiClusterService(name, namespace string) []*v1alpha1.MultiClusterService {
	var services []*v1alpha1.MultiClusterService

	for _, obj := range c.caches.multiClusterService.List() {
		mcservice := obj.(*v1alpha1.MultiClusterService)
		if mcservice.Name == name && mcservice.Namespace == namespace {
			services = append(services, mcservice)
		}
	}

	if len(services) > 0 {
		return services
	}

	return nil
}
