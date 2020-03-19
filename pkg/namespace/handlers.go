package namespace

import (
	"reflect"

	"github.com/golang/glog"

	"github.com/open-service-mesh/osm/pkg/log/level"
)

type handlers struct {
	Client
}

// general resource handlers
func (h handlers) addFunc(obj interface{}) {
	glog.V(level.Trace).Infof("[NamespaceController] Add event: %+v", obj)
}

func (h handlers) updateFunc(oldObj, newObj interface{}) {
	glog.V(level.Trace).Infof("NamespaceController] Update event %+v", oldObj)
	if reflect.DeepEqual(oldObj, newObj) {
		glog.V(level.Trace).Info("NamespaceController] Update event: no change")
		return
	}
}

func (h handlers) deleteFunc(obj interface{}) {
	glog.V(level.Trace).Infof("NamespaceController] Delete event: %+v", obj)
}
