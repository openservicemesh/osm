package kubernetes

import (
	"os"
	"reflect"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
)

var emitLogs = os.Getenv(constants.EnvVarLogKubernetesEvents) == "true"

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetKubernetesEventHandlers()
type observeFilter func(obj interface{}) bool

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName string, providerName string, announcements chan announcements.Announcement, shouldObserve observeFilter, remap map[EventType]announcements.AnnouncementType, getObjID func(obj interface{}) interface{}) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    addEvent(informerName, providerName, announcements, shouldObserve, AddEvent, remap, getObjID),
		UpdateFunc: updateEvent(informerName, providerName, announcements, shouldObserve, UpdateEvent, remap, getObjID),
		DeleteFunc: deleteEvent(informerName, providerName, announcements, shouldObserve, DeleteEvent, remap, getObjID),
	}
}

func addEvent(informerName string, providerName string, announce chan announcements.Announcement, shouldObserve observeFilter, eventType EventType, remap map[EventType]announcements.AnnouncementType, getObjID func(obj interface{}) interface{}) func(obj interface{}) {
	return func(obj interface{}) {
		if !shouldObserve(obj) {
			logNotObservedNamespace(obj, eventType)
			return
		}

		logEvent(eventType, providerName, informerName, obj)
		if announce == nil {
			return
		}

		ann := announcements.Announcement{}
		if remap != nil {
			if announcementType, ok := remap[eventType]; ok {
				ann.Type = announcementType
			}
		}

		if getObjID != nil {
			ann.ReferencedObjectID = getObjID(obj)
		}

		announce <- ann
	}
}

func updateEvent(informerName string, providerName string, announce chan announcements.Announcement, shouldObserve observeFilter, eventType EventType, remap map[EventType]announcements.AnnouncementType, getObjID func(obj interface{}) interface{}) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		if !shouldObserve(newObj) {
			logNotObservedNamespace(newObj, eventType)
			return
		}
		logEvent(eventType, providerName, informerName, oldObj)
		if announce == nil {
			return
		}
		ann := announcements.Announcement{}
		if remap != nil {
			if announcementType, ok := remap[eventType]; ok {
				ann.Type = announcementType
			}
		}

		if getObjID != nil {
			ann.ReferencedObjectID = getObjID(oldObj)
		}

		announce <- ann
	}
}

func deleteEvent(informerName string, providerName string, announce chan announcements.Announcement, shouldObserve observeFilter, eventType EventType, remap map[EventType]announcements.AnnouncementType, getObjID func(obj interface{}) interface{}) func(obj interface{}) {
	return func(obj interface{}) {
		if !shouldObserve(obj) {
			logNotObservedNamespace(obj, eventType)
			return
		}
		logEvent(eventType, providerName, informerName, obj)
		if announce == nil {
			return
		}
		ann := announcements.Announcement{}
		if remap != nil {
			if announcementType, ok := remap[eventType]; ok {
				ann.Type = announcementType
			}
		}

		if getObjID != nil {
			ann.ReferencedObjectID = getObjID(obj)
		}

		announce <- ann
	}
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

func logNotObservedNamespace(obj interface{}, eventType EventType) {
	if emitLogs {
		log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring %s event", getNamespace(obj), eventType)
	}
}

func logEvent(eventType EventType, providerName, informerName string, obj interface{}) {
	if emitLogs {
		log.Trace().Msgf("[%s][%s] %s event: %+v", providerName, informerName, eventType, obj)
	}
}
