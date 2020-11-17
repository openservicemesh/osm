package kubernetes

import (
	"os"
	"reflect"

	"k8s.io/client-go/tools/cache"

	a "github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
)

var emitLogs = os.Getenv(constants.EnvVarLogKubernetesEvents) == "true"

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetKubernetesEventHandlers()
type observeFilter func(obj interface{}) bool

// EventTypes is a struct helping pass the correct types to GetKubernetesEventHandlers
type EventTypes struct {
	Add    a.AnnouncementType
	Update a.AnnouncementType
	Delete a.AnnouncementType
}

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName, providerName string, announce chan a.Announcement, shouldObserve observeFilter, getObjID func(obj interface{}) interface{}, eventTypes EventTypes) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}

	sendAnnouncement := func(eventType a.AnnouncementType, obj interface{}) {
		if emitLogs {
			log.Trace().Msgf("[%s][%s] %s event: %+v", providerName, informerName, eventType, obj)
		}

		if announce == nil {
			return
		}

		ann := a.Announcement{
			Type: eventType,
		}

		// getObjID is a function which has enough context to establish a
		// ReferenceObjectID from the object for which this event occurred.
		// For example the ReferenceObjectID for a Pod would be the Pod's UID.
		// The getObjID function is optional;
		if getObjID != nil {
			ann.ReferencedObjectID = getObjID(obj)
		}

		announce <- ann
	}

	return cache.ResourceEventHandlerFuncs{

		AddFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				logNotObservedNamespace(obj, eventTypes.Add)
				return
			}
			sendAnnouncement(eventTypes.Add, obj)
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			if !shouldObserve(newObj) {
				logNotObservedNamespace(newObj, eventTypes.Update)
				return
			}
			sendAnnouncement(eventTypes.Update, oldObj)
		},

		DeleteFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				logNotObservedNamespace(obj, eventTypes.Delete)
				return
			}
			sendAnnouncement(eventTypes.Delete, obj)
		},
	}
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

func logNotObservedNamespace(obj interface{}, eventType a.AnnouncementType) {
	if emitLogs {
		log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring %s event", getNamespace(obj), eventType)
	}
}
