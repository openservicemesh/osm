package kubernetes

import (
	"os"
	"reflect"

	"k8s.io/client-go/tools/cache"
)

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetKubernetesEventHandlers()
type observeFilter func(obj interface{}) bool

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName string, providerName string, announcements chan interface{}, shouldObserve observeFilter) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    add(informerName, providerName, announcements, shouldObserve),
		UpdateFunc: update(informerName, providerName, announcements, shouldObserve),
		DeleteFunc: delete(informerName, providerName, announcements, shouldObserve),
	}
}

// add a new item to Kubernetes caches from an incoming Kubernetes event.
func add(informerName string, providerName string, announce chan interface{}, shouldObserve observeFilter) func(obj interface{}) {
	return func(obj interface{}) {
		if !shouldObserve(obj) {
			if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
				log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring ADD event", getNamespace(obj))
			}
			return
		}
		if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
			log.Trace().Msgf("[%s][%s] add event: %+v", providerName, informerName, obj)
		}
		if announce != nil {
			announce <- Event{
				Type:  CreateEvent,
				Value: obj,
			}
		}
	}
}

// update caches with an incoming Kubernetes event.
func update(informerName string, providerName string, announce chan interface{}, shouldObserve observeFilter) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		if !shouldObserve(newObj) {
			if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
				log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring UPDATE event", getNamespace(newObj))
			}
			return
		}
		if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
			log.Trace().Msgf("[%s][%s] update event %+v", providerName, informerName, oldObj)
		}
		if announce != nil {
			announce <- Event{
				Type:  UpdateEvent,
				Value: newObj,
			}
		}
	}
}

// delete Kubernetes cache from an incoming Kubernetes event.
func delete(informerName string, providerName string, announce chan interface{}, shouldObserve observeFilter) func(obj interface{}) {
	return func(obj interface{}) {
		if !shouldObserve(obj) {
			if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
				log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring DELETE event", getNamespace(obj))
			}
			return
		}
		if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
			log.Trace().Msgf("[%s][%s] delete event: %+v", providerName, informerName, obj)
		}
		if announce != nil {
			announce <- Event{
				Type:  DeleteEvent,
				Value: obj,
			}
		}
	}
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}
