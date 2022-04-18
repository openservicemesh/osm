package configurator

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configInformers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// NewConfigurator implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(meshConfigClientSet configClientset.Interface, stop <-chan struct{}, osmNamespace, meshConfigName string,
	msgBroker *messaging.Broker) Configurator {
	return newConfigurator(meshConfigClientSet, stop, osmNamespace, meshConfigName, msgBroker)
}

func newConfigurator(meshConfigClientSet configClientset.Interface, stop <-chan struct{}, osmNamespace string, meshConfigName string,
	msgBroker *messaging.Broker) *client {
	listOption := configInformers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, meshConfigName).String()
	})
	informerFactory := configInformers.NewSharedInformerFactoryWithOptions(
		meshConfigClientSet,
		k8s.DefaultKubeEventResyncInterval,
		configInformers.WithNamespace(osmNamespace),
		listOption,
	)
	informer := informerFactory.Config().V1alpha3().MeshConfigs().Informer()
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
	informer.AddEventHandler(c.metricsHandler())

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
func (c *client) getMeshConfig() configv1alpha3.MeshConfig {
	var meshConfig configv1alpha3.MeshConfig

	meshConfigCacheKey := c.getMeshConfigCacheKey()
	item, exists, err := c.cache.GetByKey(meshConfigCacheKey)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMeshConfigFetchFromCache)).Msgf("Error getting MeshConfig from cache with key %s", meshConfigCacheKey)
		return meshConfig
	}

	if !exists {
		log.Warn().Msgf("MeshConfig %s does not exist. Default config values will be used.", meshConfigCacheKey)
		return meshConfig
	}

	meshConfig = *item.(*configv1alpha3.MeshConfig)
	return meshConfig
}

func (c *client) metricsHandler() cache.ResourceEventHandlerFuncs {
	handleMetrics := func(obj interface{}) {
		config := obj.(*configv1alpha3.MeshConfig)

		// This uses reflection to iterate over the feature flags to avoid
		// enumerating them here individually. This code assumes the following:
		// - MeshConfig.Spec.FeatureFlags is a struct, not a pointer to a struct
		// - Each field of the FeatureFlags type is a separate feature flag of
		//   type bool
		// - Each field defines a `json` struct tag that only contains an
		//   alphanumeric field name without any other directive like `omitempty`
		flags := reflect.ValueOf(config.Spec.FeatureFlags)
		for i := 0; i < flags.NumField(); i++ {
			var val float64
			if flags.Field(i).Bool() {
				val = 1
			}
			name := flags.Type().Field(i).Tag.Get("json")
			metricsstore.DefaultMetricsStore.FeatureFlagEnabled.WithLabelValues(name).Set(val)
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc: handleMetrics,
		UpdateFunc: func(_, newObj interface{}) {
			handleMetrics(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			config := obj.(*configv1alpha3.MeshConfig).DeepCopy()
			// Ensure metrics reflect however the rest of the control plane
			// handles when the MeshConfig doesn't exist. If this happens not to
			// be the "real" MeshConfig, handleMetrics() will simply ignore it.
			config.Spec.FeatureFlags = c.GetFeatureFlags()
			handleMetrics(config)
		},
	}
}
