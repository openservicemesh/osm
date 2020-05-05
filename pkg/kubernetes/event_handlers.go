package kubernetes

import (
	"os"
	"reflect"

	"k8s.io/client-go/tools/cache"

	"github.com/open-service-mesh/osm/pkg/namespace"
)

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName string, providerName string, announcements chan interface{}, namespaceController namespace.Controller) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    Add(informerName, providerName, announcements, namespaceController),
		UpdateFunc: Update(informerName, providerName, announcements, namespaceController),
		DeleteFunc: Delete(informerName, providerName, announcements, namespaceController),
	}
}

// Add a new item to Kubernetes caches from an incoming Kubernetes event.
func Add(informerName string, providerName string, announce chan interface{}, namespaceController namespace.Controller) func(obj interface{}) {
	return func(obj interface{}) {
		ns := getNamespace(obj)
		if !namespaceController.IsMonitoredNamespace(ns) {
			log.Debug().Msgf("Not monitored: %s", ns)
			return
		}
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
func Update(informerName string, providerName string, announce chan interface{}, namespaceController namespace.Controller) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		ns := getNamespace(newObj)
		if !namespaceController.IsMonitoredNamespace(ns) {
			return
		}
		if reflect.DeepEqual(oldObj, newObj) {
			return
		}
		if os.Getenv("OSM_LOG_KUBERNETES_EVENTS") == "true" {
			log.Trace().Msgf("[%s][%s] Update event %+v", providerName, informerName, oldObj)
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
func Delete(informerName string, providerName string, announce chan interface{}, namespaceController namespace.Controller) func(obj interface{}) {
	return func(obj interface{}) {
		ns := getNamespace(obj)
		if !namespaceController.IsMonitoredNamespace(ns) {
			return
		}
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

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}
