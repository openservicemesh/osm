package config

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	configV1alpha3Client "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configV1alpha3Informers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
)

const (
	// apiGroup is the k8s API group that this package interacts with
	apiGroup = "config.openservicemesh.io"

	multiclusterInformerName = `MulticlusterService`
)

// NewConfigController returns a config.Controller struct related to functionality provided by the resources in the config.openservicemesh.io API group
func NewConfigController(kubeConfig *rest.Config, kubeController k8s.Controller, stop chan struct{}, msgBroker *messaging.Broker) (Controller, error) {
	configClient := configV1alpha3Client.NewForConfigOrDie(kubeConfig)
	informerFactory := configV1alpha3Informers.NewSharedInformerFactory(configClient, k8s.DefaultKubeEventResyncInterval)

	client := client{
		informer:       informerFactory.Config().V1alpha3().MultiClusterServices(),
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		return ok && kubeController.IsMonitoredNamespace(object.GetNamespace())
	}

	multiclusterServiceEventTypes := k8s.EventTypes{
		Add:    announcements.MultiClusterServiceAdded,
		Update: announcements.MultiClusterServiceUpdated,
		Delete: announcements.MultiClusterServiceDeleted,
	}
	client.informer.Informer().AddEventHandler(k8s.GetEventHandlerFuncs(shouldObserve, multiclusterServiceEventTypes, msgBroker))

	if err := client.run(stop); err != nil {
		return client, errors.Errorf("Could not start %s client: %s", apiGroup, err)
	}
	return client, nil
}

func (c client) run(stop <-chan struct{}) error {
	log.Info().Msgf("Starting informers for %s", apiGroup)

	if c.informer == nil {
		return errInitInformers
	}

	go c.informer.Informer().Run(stop)

	log.Info().Msgf("Waiting for %s %s informers' cache to sync", apiGroup, multiclusterInformerName)
	if !cache.WaitForCacheSync(stop, c.informer.Informer().HasSynced) {
		return errSyncingCaches
	}

	log.Info().Msgf("Cache sync finished for %s %s informers", apiGroup, multiclusterInformerName)
	return nil
}
