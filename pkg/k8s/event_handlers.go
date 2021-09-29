package k8s

import (
	"reflect"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetKubernetesEventHandlers()
type observeFilter func(obj interface{}) bool

// EventTypes is a struct helping pass the correct types to GetKubernetesEventHandlers
type EventTypes struct {
	Add    announcements.Kind
	Update announcements.Kind
	Delete announcements.Kind
}

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(shouldObserve observeFilter, eventTypes EventTypes) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}

	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				return
			}
			logResourceEvent(log, eventTypes.Add, obj)
			if eventTypes.Add == announcements.NamespaceAdded {
				metricsstore.DefaultMetricsStore.NamespaceCount.Inc()
			}
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Add.String(), ns).Inc()
			events.Publish(events.PubSubMessage{
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
			if eventTypes.Update == announcements.NamespaceUpdated {
				if !isMonitoredNamespace(oldObj) && isMonitoredNamespace(newObj) {
					metricsstore.DefaultMetricsStore.NamespaceCount.Inc()
				} else if isMonitoredNamespace(oldObj) && !isMonitoredNamespace(newObj) {
					metricsstore.DefaultMetricsStore.NamespaceCount.Dec()
				}
			}
			ns := getNamespace(newObj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Update.String(), ns).Inc()
			events.Publish(events.PubSubMessage{
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
			if eventTypes.Delete == announcements.NamespaceDeleted {
				metricsstore.DefaultMetricsStore.NamespaceCount.Dec()
			}
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Delete.String(), ns).Inc()
			events.Publish(events.PubSubMessage{
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

func isMonitoredNamespace(obj interface{}) bool {
	labels := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Labels").Interface()
	if labels == nil {
		return false
	}

	switch labels := labels.(type) {
	case map[string]string:
		mesh := labels[constants.OSMKubeResourceMonitorAnnotation]
		return mesh != ""
	default:
		return false
	}
}
