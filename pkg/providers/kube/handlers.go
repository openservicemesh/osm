package kube

import (
	"reflect"

	"github.com/golang/glog"

	"github.com/open-service-mesh/osm/pkg/log/level"
)

// general resource handlers
func (c Client) addFunc(obj interface{}) {
	glog.V(level.Trace).Infof("[%s] Add event: %+v", c.providerIdent, obj)
	c.announcements <- Event{
		Type:  Create,
		Value: obj,
	}
}

func (c Client) updateFunc(oldObj, newObj interface{}) {
	glog.V(level.Trace).Infof("[%s] Update event %+v", c.providerIdent, oldObj)
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.announcements <- Event{
		Type:  Update,
		Value: newObj,
	}
}

func (c Client) deleteFunc(obj interface{}) {
	glog.V(level.Trace).Infof("[%s] Delete event: %+v", c.providerIdent, obj)
	c.announcements <- Event{
		Type:  Delete,
		Value: obj,
	}
}
