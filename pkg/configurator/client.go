package configurator

import (
	"fmt"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	informers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"
	"github.com/openservicemesh/osm/pkg/messaging"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s"
)

// NewConfigurator implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(meshConfigClientSet versioned.Interface, stop <-chan struct{}, osmNamespace, meshConfigName string,
	msgBroker *messaging.Broker) Configurator {
	return newConfigurator(meshConfigClientSet, stop, osmNamespace, meshConfigName, msgBroker)
}

func newConfigurator(meshConfigClientSet versioned.Interface, stop <-chan struct{}, osmNamespace string, meshConfigName string,
	msgBroker *messaging.Broker) *client {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		meshConfigClientSet,
		k8s.DefaultKubeEventResyncInterval,
		informers.WithNamespace(osmNamespace),
	)
	informer := informerFactory.Config().V1alpha1().MeshConfigs().Informer()
	c := &client{
		informer:       informer,
		cache:          informer.GetStore(),
		osmNamespace:   osmNamespace,
		meshConfigName: meshConfigName,
	}

	// configure listener
	eventTypes := k8s.EventTypes{
		Add:    announcements.MeshConfigAdded,
		Update: announcements.MeshConfigUpdated,
		Delete: announcements.MeshConfigDeleted,
	}
	informer.AddEventHandler(k8s.GetEventHandlerFuncs(nil, eventTypes, msgBroker))

	c.run(stop)

	return c
}

func (c *client) run(stop <-chan struct{}) {
	go c.informer.Run(stop) // run the informer synchronization
	log.Debug().Msgf("Started OSM MeshConfig informer")
	log.Debug().Msg("[MeshConfig client] Waiting for MeshConfig informer's cache to sync")
	if !cache.WaitForCacheSync(stop, c.informer.HasSynced) {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMeshConfigInformerInitCache)).Msg("Failed initial cache sync for MeshConfig informer")
		return
	}

	log.Debug().Msg("[MeshConfig client] Cache sync for MeshConfig informer finished")
}

func (c *client) getMeshConfigCacheKey() string {
	return fmt.Sprintf("%s/%s", c.osmNamespace, c.meshConfigName)
}

// Returns the current MeshConfig
func (c *client) getMeshConfig() *v1alpha1.MeshConfig {
	meshConfigCacheKey := c.getMeshConfigCacheKey()
	item, exists, err := c.cache.GetByKey(meshConfigCacheKey)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMeshConfigFetchFromCache)).Msgf("Error getting MeshConfig from cache with key %s", meshConfigCacheKey)
		return &v1alpha1.MeshConfig{}
	}

	var meshConfig *v1alpha1.MeshConfig
	if !exists {
		log.Warn().Msgf("MeshConfig %s does not exist. Default config values will be used.", meshConfigCacheKey)
		meshConfig = &v1alpha1.MeshConfig{}
	} else {
		meshConfig = item.(*v1alpha1.MeshConfig)
	}

	return meshConfig
}
