package config

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	configV1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configV1alpha1Informers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"
	"github.com/openservicemesh/osm/pkg/k8s"
)

const (
	// apiGroup is the k8s API group that this package interacts with
	apiGroup = "config.openservicemesh.io"
)

// NewConfigController returns a config.Controller struct related to functionality provided by the resources in the config.openservicemesh.io API group
func NewConfigController(kubeConfig *rest.Config, kubeController k8s.Controller, stop chan struct{}) (Controller, error) {
	configClient := configV1alpha1Client.NewForConfigOrDie(kubeConfig)
	informerFactory := configV1alpha1Informers.NewSharedInformerFactory(configClient, k8s.DefaultKubeEventResyncInterval)

	client := client{
		informer:       informerFactory.Config().V1alpha1().MultiClusterServices(),
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
	client.informer.Informer().AddEventHandler(k8s.GetKubernetesEventHandlers("MultiClusterService", "Kube", shouldObserve, remoteServiceEventTypes))

	err := client.run(stop)
	if err != nil {
		return client, errors.Errorf("Could not start %s client: %s", apiGroup, err)
	}
	return client, err
}

func (c client) run(stop <-chan struct{}) error {
	log.Info().Msgf("%s client started", apiGroup)

	if c.informer == nil {
		return errInitInformers
	}

	go c.informer.Informer().Run(stop)

	log.Info().Msgf("Waiting for %s RemoteService informers' cache to sync", apiGroup)
	if !cache.WaitForCacheSync(stop, c.informer.Informer().HasSynced) {
		return errSyncingCaches
	}

	log.Info().Msgf("Cache sync finished for %s RemoteService informers", apiGroup)
	return nil
}
