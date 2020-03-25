package kube

import (
	"reflect"

	"github.com/golang/glog"
	"k8s.io/client-go/tools/cache"

	"github.com/open-service-mesh/osm/pkg/log/level"
)

func (c Client) getResourceEventHandlers(informerName string) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addFunc(informerName),
		UpdateFunc: c.updateFunc(informerName),
		DeleteFunc: c.deleteFunc(informerName),
	}
}

func (c Client) addFunc(informerName string) func(obj interface{}) {
	return func(obj interface{}) {
		glog.V(level.Trace).Infof("[%s][%s] Add event: %+v", c.providerIdent, informerName, obj)
		c.announcements <- Event{
			Type:  Create,
			Value: obj,
		}
	}
}

func (c Client) updateFunc(informerName string) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		glog.V(level.Trace).Infof("[%s][%s] Update event %+v", c.providerIdent, informerName, oldObj)
		if reflect.DeepEqual(oldObj, newObj) {
			return
		}
		c.announcements <- Event{
			Type:  Update,
			Value: newObj,
		}
	}
}

func (c Client) deleteFunc(informerName string) func(obj interface{}) {
	return func(obj interface{}) {
		glog.V(level.Trace).Infof("[%s][%s] Delete event: %+v", c.providerIdent, informerName, obj)
		c.announcements <- Event{
			Type:  Delete,
			Value: obj,
		}
	}
}
