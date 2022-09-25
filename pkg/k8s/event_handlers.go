package k8s

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// Function to filter K8s meta Objects by OSM's isMonitoredNamespace
func (c *Client) shouldObserve(obj interface{}) bool {
	switch v := obj.(type) {
	case *corev1.Namespace, *configv1alpha2.MeshConfig, *configv1alpha2.MeshRootCertificate, *configv1alpha2.ExtensionService:
		return true
	case metav1.Object:
		return c.IsMonitoredNamespace(v.GetNamespace())
	}
	return false
}

// defaultEventHandler returns the ResourceEventHandlerFuncs object used to receive events when a k8s
// object is added/updated/deleted.
func (c *Client) defaultEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.handleEvent(events.Added, nil, obj)
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			c.handleEvent(events.Updated, oldObj, newObj)
		},

		DeleteFunc: func(obj interface{}) {
			c.handleEvent(events.Deleted, obj, nil)
		},
	}
}

func (c *Client) handleEvent(event events.EventType, oldObj, newObj interface{}) {
	obj := newObj
	if event == events.Deleted {
		obj = oldObj
	}

	if !c.shouldObserve(obj) {
		return
	}

	msg := events.PubSubMessage{
		Kind:   events.GetKind(obj),
		Type:   event,
		NewObj: newObj,
		OldObj: oldObj,
	}
	logResourceEvent(msg.Topic(), obj)
	ns := getNamespace(obj)
	metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(msg.Topic(), ns).Inc()
	c.msgBroker.AddEvent(msg)
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

func logResourceEvent(event string, obj interface{}) {
	log := log.With().Str("event", event).Logger()
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
