package k8s

import (
	"reflect"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetEventHandlerFuncs()
type observeFilter func(obj interface{}) bool

// EventTypes is a struct helping pass the correct types to GetEventHandlerFuncs()
type EventTypes struct {
	Add    announcements.Kind
	Update announcements.Kind
	Delete announcements.Kind
}

// GetEventHandlerFuncs returns the ResourceEventHandlerFuncs object used to receive events when a k8s
// object is added/updated/deleted.
func GetEventHandlerFuncs(shouldObserve observeFilter, eventTypes EventTypes, msgBroker *messaging.Broker) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}

	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				return
			}
			logResourceEvent(log, eventTypes.Add, obj)
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Add.String(), ns).Inc()
			msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind:   eventTypes.Add,
				NewObj: obj,
				OldObj: nil,
			})
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			if !shouldObserve(newObj) {
				return
			}
			logResourceEvent(log, eventTypes.Update, newObj)
			ns := getNamespace(newObj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Update.String(), ns).Inc()
			msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind:   eventTypes.Update,
				NewObj: newObj,
				OldObj: oldObj,
			})
		},

		DeleteFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				return
			}
			logResourceEvent(log, eventTypes.Delete, obj)
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Delete.String(), ns).Inc()
			msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind:   eventTypes.Delete,
				NewObj: nil,
				OldObj: obj,
			})
		},
	}
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

func logResourceEvent(parent zerolog.Logger, event announcements.Kind, obj interface{}) {
	log := parent.With().Str("event", event.String()).Logger()
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
