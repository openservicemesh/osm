package k8s

import (
	"reflect"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetKubernetesEventHandlers()
type observeFilter func(obj interface{}) bool

// EventTypes is a struct helping pass the correct types to GetKubernetesEventHandlers
type EventTypes struct {
	Add    announcements.AnnouncementType
	Update announcements.AnnouncementType
	Delete announcements.AnnouncementType
}

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName, providerName string, shouldObserve observeFilter, eventTypes EventTypes) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}

	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				return
			}
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Add.String(), ns).Inc()
			events.Publish(events.PubSubMessage{
				AnnouncementType: eventTypes.Add,
				NewObj:           obj,
				OldObj:           nil,
			})
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			if !shouldObserve(newObj) {
				return
			}
			ns := getNamespace(newObj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Update.String(), ns).Inc()
			events.Publish(events.PubSubMessage{
				AnnouncementType: eventTypes.Update,
				NewObj:           newObj,
				OldObj:           oldObj,
			})
		},

		DeleteFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				return
			}
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Delete.String(), ns).Inc()
			events.Publish(events.PubSubMessage{
				AnnouncementType: eventTypes.Delete,
				NewObj:           nil,
				OldObj:           obj,
			})
		},
	}
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}
