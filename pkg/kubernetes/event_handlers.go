package kubernetes

import (
	"os"
	"reflect"

	"k8s.io/client-go/tools/cache"
)

type namespaceFilter func(obj interface{}) bool

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName string, providerName string, announcements chan interface{}, nsFilter namespaceFilter) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    add(informerName, providerName, announcements, nsFilter),
		UpdateFunc: update(informerName, providerName, announcements, nsFilter),
		DeleteFunc: delete(informerName, providerName, announcements, nsFilter),
	}
}

// add a new item to Kubernetes caches from an incoming Kubernetes event.
func add(informerName string, providerName string, announce chan interface{}, nsFilter namespaceFilter) func(obj interface{}) {
	return func(obj interface{}) {
		if nsFilter == nil || !nsFilter(obj) {
			log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring ADD event", getNamespace(obj))
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
func update(informerName string, providerName string, announce chan interface{}, nsFilter namespaceFilter) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		if nsFilter == nil || !nsFilter(newObj) {
			log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring UPDATE event", getNamespace(newObj))
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
func delete(informerName string, providerName string, announce chan interface{}, nsFilter namespaceFilter) func(obj interface{}) {
	return func(obj interface{}) {
		if nsFilter == nil || !nsFilter(obj) {
			log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring DELETE event", getNamespace(obj))
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
