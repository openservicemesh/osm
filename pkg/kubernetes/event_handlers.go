package kubernetes

import (
	"reflect"

	"github.com/golang/glog"
	"github.com/open-service-mesh/osm/pkg/log/level"
	"k8s.io/client-go/tools/cache"
)

func GetKubernetesEventHandlers(informerName string, providerName string, announcements chan interface{}) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    Add(informerName, providerName, announcements),
		UpdateFunc: Update(informerName, providerName, announcements),
		DeleteFunc: Delete(informerName, providerName, announcements),
	}
}

func Add(informerName string, providerName string, announce chan interface{}) func(obj interface{}) {
	return func(obj interface{}) {
		glog.V(level.Trace).Infof("[%s][%s] Add event: %+v", providerName, informerName, obj)
		if announce != nil {
			announce <- Event{
				Type:  CreateEvent,
				Value: obj,
			}
		}
	}
}

func Update(informerName string, providerName string, announce chan interface{}) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		glog.V(level.Trace).Infof("[%s][%s] Update event %+v", providerName, informerName, oldObj)
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

func Delete(informerName string, providerName string, announce chan interface{}) func(obj interface{}) {
	return func(obj interface{}) {
		glog.V(level.Trace).Infof("[%s][%s] Delete event: %+v", providerName, informerName, obj)
		if announce != nil {
			announce <- Event{
				Type:  DeleteEvent,
				Value: obj,
			}
		}
	}
}
