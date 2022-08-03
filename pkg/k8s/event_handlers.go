package k8s

import (
	"reflect"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"

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
