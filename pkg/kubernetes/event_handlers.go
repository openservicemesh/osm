package kubernetes

import (
	"os"
	"reflect"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/dispatcher"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

var emitLogs = os.Getenv(constants.EnvVarLogKubernetesEvents) == "true"

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetKubernetesEventHandlers()
type observeFilter func(obj interface{}) bool

// EventTypes is a struct helping pass the correct types to GetKubernetesEventHandlers
type EventTypes struct {
	Add    dispatcher.AnnouncementType
	Update dispatcher.AnnouncementType
	Delete dispatcher.AnnouncementType
}

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName, providerName string, shouldObserve observeFilter, eventTypes EventTypes) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}

	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				logNotObservedNamespace(obj, eventTypes.Add, informerName, providerName)
				return
			}
			dispatcher.GetPubSubInstance().Publish(dispatcher.PubSubMessage{
				AnnouncementType: eventTypes.Add,
				NewObj:           obj,
				OldObj:           nil,
			})
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Add.String(), ns).Inc()
			updateEventSpecificMetrics(eventTypes.Add)
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			if !shouldObserve(newObj) {
				logNotObservedNamespace(newObj, eventTypes.Update, informerName, providerName)
				return
			}
			dispatcher.GetPubSubInstance().Publish(dispatcher.PubSubMessage{
				AnnouncementType: eventTypes.Update,
				NewObj:           oldObj,
				OldObj:           newObj,
			})
			ns := getNamespace(newObj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Update.String(), ns).Inc()
		},

		DeleteFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				logNotObservedNamespace(obj, eventTypes.Delete, informerName, providerName)
				return
			}
			dispatcher.GetPubSubInstance().Publish(dispatcher.PubSubMessage{
				AnnouncementType: eventTypes.Delete,
				NewObj:           nil,
				OldObj:           obj,
			})
			ns := getNamespace(obj)
			metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(eventTypes.Delete.String(), ns).Inc()
			updateEventSpecificMetrics(eventTypes.Delete)
		},
	}
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

func logNotObservedNamespace(obj interface{}, eventType dispatcher.AnnouncementType, informerName, providerName string) {
	if emitLogs {
		log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring %s event; informer=%s; provider=%s", getNamespace(obj), eventType, informerName, providerName)
	}
}
