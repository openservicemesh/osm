package kubernetes

import (
	"reflect"

	"github.com/golang/glog"
	"github.com/open-service-mesh/osm/pkg/log/level"
	"k8s.io/client-go/tools/cache"
)

<<<<<<< HEAD
// GetKubernetesEventHandlers creates Kubernetes events handlers.
=======
>>>>>>> kubernetes: Carving out Kubernetes event handlers in their own package so we can reuse them
func GetKubernetesEventHandlers(informerName string, providerName string, announcements chan interface{}) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    Add(informerName, providerName, announcements),
		UpdateFunc: Update(informerName, providerName, announcements),
		DeleteFunc: Delete(informerName, providerName, announcements),
	}
}

<<<<<<< HEAD
// Add a new item to Kubernetes caches from an incoming Kubernetes event.
=======
>>>>>>> kubernetes: Carving out Kubernetes event handlers in their own package so we can reuse them
func Add(informerName string, providerName string, announce chan interface{}) func(obj interface{}) {
	return func(obj interface{}) {
		glog.V(level.Trace).Infof("[%s][%s] Add event: %+v", providerName, informerName, obj)
		announce <- Event{
			Type:  CreateEvent,
			Value: obj,
		}
	}
}

<<<<<<< HEAD
// Update caches with an incoming Kubernetes event.
=======
>>>>>>> kubernetes: Carving out Kubernetes event handlers in their own package so we can reuse them
func Update(informerName string, providerName string, announce chan interface{}) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		glog.V(level.Trace).Infof("[%s][%s] Update event %+v", providerName, informerName, oldObj)
		if reflect.DeepEqual(oldObj, newObj) {
			return
		}
		announce <- Event{
			Type:  UpdateEvent,
			Value: newObj,
		}
	}
}

<<<<<<< HEAD
// Delete Kubernetes cache from an incoming Kubernetes event.
=======
>>>>>>> kubernetes: Carving out Kubernetes event handlers in their own package so we can reuse them
func Delete(informerName string, providerName string, announce chan interface{}) func(obj interface{}) {
	return func(obj interface{}) {
		glog.V(level.Trace).Infof("[%s][%s] Delete event: %+v", providerName, informerName, obj)
		announce <- Event{
			Type:  DeleteEvent,
			Value: obj,
		}
	}
}
