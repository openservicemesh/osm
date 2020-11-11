package kubernetes

import (
	"os"
	"reflect"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	eventAdd    = "ADD"
	eventUpdate = "UPDATE"
	eventDelete = "DELETE"
)

var emitLogs = os.Getenv(constants.EnvVarLogKubernetesEvents) == "true"

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetKubernetesEventHandlers()
type observeFilter func(obj interface{}) bool

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName string, providerName string, announcements chan announcements.Announcement, shouldObserve observeFilter) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    addEvent(informerName, providerName, announcements, shouldObserve, eventAdd),
		UpdateFunc: updateEvent(informerName, providerName, announcements, shouldObserve, eventUpdate),
		DeleteFunc: deleteEvent(informerName, providerName, announcements, shouldObserve, eventDelete),
	}
}

func addEvent(informerName string, providerName string, announce chan announcements.Announcement, shouldObserve observeFilter, eventType string) func(obj interface{}) {
	return func(obj interface{}) {
		if !shouldObserve(obj) {
			logNotObservedNamespace(obj, eventType)
			return
		}
		logEvent(eventType, providerName, informerName, obj)
		if announce != nil {
			announce <- announcements.Announcement{}
		}
	}
}

func updateEvent(informerName string, providerName string, announce chan announcements.Announcement, shouldObserve observeFilter, eventType string) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		if !shouldObserve(newObj) {
			logNotObservedNamespace(newObj, eventType)
			return
		}
		logEvent(eventType, providerName, informerName, oldObj)
		if announce != nil {
			announce <- announcements.Announcement{}
		}
	}
}

func deleteEvent(informerName string, providerName string, announce chan announcements.Announcement, shouldObserve observeFilter, eventType string) func(obj interface{}) {
	return func(obj interface{}) {
		if !shouldObserve(obj) {
			logNotObservedNamespace(obj, eventType)
			return
		}
		logEvent(eventType, providerName, informerName, obj)
		if announce != nil {
			announce <- announcements.Announcement{}
		}
	}
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

func logNotObservedNamespace(obj interface{}, eventType string) {
	if emitLogs {
		log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring %s event", getNamespace(obj), eventType)
	}
}

func logEvent(eventType, providerName, informerName string, obj interface{}) {
	if emitLogs {
		log.Trace().Msgf("[%s][%s] %s event: %+v", providerName, informerName, eventType, obj)
	}
}
