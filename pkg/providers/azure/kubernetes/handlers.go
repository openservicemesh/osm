package azure

import (
	"reflect"

	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

type handlers struct {
	Client
}

// general resource handlers
func (h handlers) addFunc(obj interface{}) {
	glog.V(9).Infof("[%s] Add event: %+v", h.providerIdent, obj)
	h.announcements <- events.Event{
		Type:  events.Create,
		Value: obj,
	}
}

func (h handlers) updateFunc(oldObj, newObj interface{}) {
	glog.V(9).Infof("[%s] Update event %+v", h.providerIdent, oldObj)
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	h.announcements <- events.Event{
		Type:  events.Update,
		Value: newObj,
	}
}

func (h handlers) deleteFunc(obj interface{}) {
	glog.V(9).Infof("[%s] Delete event: %+v", h.providerIdent, obj)
	h.announcements <- events.Event{
		Type:  events.Delete,
		Value: obj,
	}
}
