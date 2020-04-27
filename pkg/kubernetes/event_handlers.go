package kubernetes

import (
	"os"
	"reflect"

	"k8s.io/client-go/tools/cache"
)

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName string, providerName string, announcements chan interface{}) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    Add(informerName, providerName, announcements),
		UpdateFunc: Update(informerName, providerName, announcements),
		DeleteFunc: Delete(informerName, providerName, announcements),
	}
}

// Add a new item to Kubernetes caches from an incoming Kubernetes event.
func Add(informerName string, providerName string, announce chan interface{}) func(obj interface{}) {
	return func(obj interface{}) {
		if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
			log.Trace().Msgf("[%s][%s] Add event: %+v", providerName, informerName, obj)
		}
		if announce != nil {
			announce <- Event{
				Type:  CreateEvent,
				Value: obj,
			}
		}
	}
}

// Update caches with an incoming Kubernetes event.
func Update(informerName string, providerName string, announce chan interface{}) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
			log.Trace().Msgf("[%s][%s] Update event %+v", providerName, informerName, oldObj)
		}
		if reflect.DeepEqual(oldObj, newObj) {
			return
		}
		if announce != nil {
			announce <- Event{
				Type:  UpdateEvent,
				Value: newObj,
			}
		}
	}
}

// Delete Kubernetes cache from an incoming Kubernetes event.
func Delete(informerName string, providerName string, announce chan interface{}) func(obj interface{}) {
	return func(obj interface{}) {
		if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
			log.Trace().Msgf("[%s][%s] Delete event: %+v", providerName, informerName, obj)
		}
		if announce != nil {
			announce <- Event{
				Type:  DeleteEvent,
				Value: obj,
			}
		}
	}
}
