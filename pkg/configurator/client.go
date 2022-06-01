package configurator

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configInformers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// NewConfigurator implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(configClient configClientset.Interface, stop <-chan struct{}, osmNamespace, meshConfigName string,
	msgBroker *messaging.Broker) (Configurator, error) {
	return newConfigurator(configClient, stop, osmNamespace, meshConfigName, msgBroker)
}

func newConfigurator(configClient configClientset.Interface, stop <-chan struct{}, osmNamespace string, meshConfigName string,
	msgBroker *messaging.Broker) (*client, error) {
	listOption := configInformers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, meshConfigName).String()
	})

	meshConfigInformerFactory := configInformers.NewSharedInformerFactoryWithOptions(
		configClient,
		k8s.DefaultKubeEventResyncInterval,
		configInformers.WithNamespace(osmNamespace),
		listOption,
	)

	// informerFactory without listOptions
	configInformerFactory := configInformers.NewSharedInformerFactoryWithOptions(
		configClient,
		k8s.DefaultKubeEventResyncInterval,
		configInformers.WithNamespace(osmNamespace),
	)

	informerCollection := informerCollection{
		meshConfig:          meshConfigInformerFactory.Config().V1alpha2().MeshConfigs().Informer(),
		meshRootCertificate: configInformerFactory.Config().V1alpha2().MeshRootCertificates().Informer(),
	}

	cacheCollection := cacheCollection{
		meshConfig:          informerCollection.meshConfig.GetStore(),
		meshRootCertificate: informerCollection.meshRootCertificate.GetStore(),
	}

	c := &client{
		informers:      &informerCollection,
		caches:         &cacheCollection,
		osmNamespace:   osmNamespace,
		meshConfigName: meshConfigName,
	}

	// configure listener
	meshConfigEventTypes := k8s.EventTypes{
		Add:    announcements.MeshConfigAdded,
		Update: announcements.MeshConfigUpdated,
		Delete: announcements.MeshConfigDeleted,
	}
	informerCollection.meshConfig.AddEventHandler(k8s.GetEventHandlerFuncs(nil, meshConfigEventTypes, msgBroker))
	informerCollection.meshConfig.AddEventHandler(c.metricsHandler())

	meshRootCertificateEventTypes := k8s.EventTypes{
		Add:    announcements.MeshRootCertificateAdded,
		Update: announcements.MeshRootCertificateUpdated,
		Delete: announcements.MeshRootCertificateDeleted,
	}
	informerCollection.meshRootCertificate.AddEventHandler(k8s.GetEventHandlerFuncs(nil, meshRootCertificateEventTypes, msgBroker))

	err := c.run(stop)
	if err != nil {
		return c, errors.Errorf("Could not start %s informer clients: %s", configv1alpha2.SchemeGroupVersion, err)
	}

	return c, nil
}

func (c *client) run(stop <-chan struct{}) error {
	log.Info().Msgf("Starting informer clients for API group %s", configv1alpha2.SchemeGroupVersion)

	if c.informers == nil {
		return errors.New("config.openservicemesh.io informers not initialized")
	}

	sharedInformers := map[string]cache.SharedInformer{
		"MeshConfig":          c.informers.meshConfig,
		"MeshRootCertificate": c.informers.meshRootCertificate,
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
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrConfigInformerInitCache)).Msg("Failed initial cache sync for config.openservicemesh.io informers")
		return errors.New("Failed initial cache sync for config.openservicemesh.io informers")
	}

	log.Info().Msgf("Cache sync finished for %v informers in API group %s", informerNames, configv1alpha2.SchemeGroupVersion)
	return nil
}

func (c *client) getMeshConfigCacheKey() string {
	return fmt.Sprintf("%s/%s", c.osmNamespace, c.meshConfigName)
}

// Returns the current MeshConfig
func (c *client) getMeshConfig() configv1alpha2.MeshConfig {
	var meshConfig configv1alpha2.MeshConfig

	meshConfigCacheKey := c.getMeshConfigCacheKey()
	item, exists, err := c.caches.meshConfig.GetByKey(meshConfigCacheKey)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMeshConfigFetchFromCache)).Msgf("Error getting MeshConfig from cache with key %s", meshConfigCacheKey)
		return meshConfig
	}

	if !exists {
		log.Warn().Msgf("MeshConfig %s does not exist. Default config values will be used.", meshConfigCacheKey)
		return meshConfig
	}

	meshConfig = *item.(*configv1alpha2.MeshConfig)
	return meshConfig
}

func (c *client) metricsHandler() cache.ResourceEventHandlerFuncs {
	handleMetrics := func(obj interface{}) {
		config := obj.(*configv1alpha2.MeshConfig)

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
			config := obj.(*configv1alpha2.MeshConfig).DeepCopy()
			// Ensure metrics reflect however the rest of the control plane
			// handles when the MeshConfig doesn't exist. If this happens not to
			// be the "real" MeshConfig, handleMetrics() will simply ignore it.
			config.Spec.FeatureFlags = c.GetFeatureFlags()
			handleMetrics(config)
		},
	}
}
