package configurator

import (
	"fmt"
	"reflect"

	"k8s.io/client-go/tools/cache"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/k8s/informers"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// NewConfigurator implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(informerCollection *informers.InformerCollection, osmNamespace, meshConfigName string, msgBroker *messaging.Broker) *Client {
	c := &Client{
		informers:      informerCollection,
		osmNamespace:   osmNamespace,
		meshConfigName: meshConfigName,
	}

	// configure listener
	meshConfigEventTypes := k8s.EventTypes{
		Add:    announcements.MeshConfigAdded,
		Update: announcements.MeshConfigUpdated,
		Delete: announcements.MeshConfigDeleted,
	}

	informerCollection.AddEventHandler(informers.InformerKeyMeshConfig, k8s.GetEventHandlerFuncs(nil, meshConfigEventTypes, msgBroker))
	informerCollection.AddEventHandler(informers.InformerKeyMeshConfig, c.metricsHandler())

	meshRootCertificateEventTypes := k8s.EventTypes{
		Add:    announcements.MeshRootCertificateAdded,
		Update: announcements.MeshRootCertificateUpdated,
		Delete: announcements.MeshRootCertificateDeleted,
	}
	informerCollection.AddEventHandler(informers.InformerKeyMeshRootCertificate, k8s.GetEventHandlerFuncs(nil, meshRootCertificateEventTypes, msgBroker))

	return c
}

func (c *Client) getMeshConfigCacheKey() string {
	return fmt.Sprintf("%s/%s", c.osmNamespace, c.meshConfigName)
}

// Returns the current MeshConfig
func (c *Client) getMeshConfig() configv1alpha2.MeshConfig {
	var meshConfig configv1alpha2.MeshConfig

	meshConfigCacheKey := c.getMeshConfigCacheKey()
	item, exists, err := c.informers.GetByKey(informers.InformerKeyMeshConfig, meshConfigCacheKey)
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

func (c *Client) metricsHandler() cache.ResourceEventHandlerFuncs {
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
