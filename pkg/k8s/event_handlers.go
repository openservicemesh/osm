package k8s

import (
	"reflect"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetEventHandlerFuncs()
type observeFilter func(obj interface{}) bool

// GetEventHandlerFuncs returns the ResourceEventHandlerFuncs object used to receive events when a k8s
// object is added/updated/deleted.
func GetEventHandlerFuncs(shouldObserve observeFilter, msgBroker *messaging.Broker) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				return
			}
			msg := events.PubSubMessage{
				Kind:   events.GetKind(obj),
				Type:   events.Added,
				NewObj: obj,
				OldObj: nil,
			}
			logResourceEvent(log, msg.Topic(), obj)
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(msg.Topic(), ns).Inc()
			msgBroker.GetQueue().AddRateLimited(msg)
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			if !shouldObserve(newObj) {
				return
			}
			msg := events.PubSubMessage{
				Kind:   events.GetKind(newObj),
				Type:   events.Updated,
				NewObj: newObj,
				OldObj: oldObj,
			}
			logResourceEvent(log, msg.Topic(), newObj)
			ns := getNamespace(newObj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(msg.Topic(), ns).Inc()
			msgBroker.GetQueue().AddRateLimited(msg)
		},

		DeleteFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				return
			}
			msg := events.PubSubMessage{
				Kind:   events.GetKind(obj),
				Type:   events.Deleted,
				NewObj: nil,
				OldObj: obj,
			}
			logResourceEvent(log, msg.Topic(), obj)
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(msg.Topic(), ns).Inc()
			msgBroker.GetQueue().AddRateLimited(msg)
		},
	}
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

func logResourceEvent(parent zerolog.Logger, event string, obj interface{}) {
	log := parent.With().Str("event", event).Logger()
	o, err := meta.Accessor(obj)
	if err != nil {
		log.Error().Err(err).Msg("error parsing object, ignoring")
		return
	}
	name := o.GetName()
	if o.GetNamespace() != "" {
		name = o.GetNamespace() + "/" + name
	}
	log.Debug().Str("resource_name", name).Msg("received kubernetes resource event")
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
			config.Spec.FeatureFlags = c.GetMeshConfig().Spec.FeatureFlags
			handleMetrics(config)
		},
	}
}
