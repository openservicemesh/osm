package smi

import (
	"reflect"

	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"

	"github.com/deislabs/smc/pkg/log"
)

type handlers struct {
	Client
}

// general resource handlers
func (h handlers) addFunc(obj interface{}) {
	glog.V(log.LvlTrace).Infof("[%s] Add event: %+v", h.providerIdent, obj)
	h.announcements <- events.Event{
		Type:  events.Create,
		Value: obj,
	}
}

func (h handlers) updateFunc(oldObj, newObj interface{}) {
	glog.V(log.LvlTrace).Infof("[%s] Update event %+v", h.providerIdent, oldObj)
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	h.announcements <- events.Event{
		Type:  events.Update,
		Value: newObj,
	}
}

func (h handlers) deleteFunc(obj interface{}) {
	glog.V(log.LvlTrace).Infof("[%s] Delete event: %+v", h.providerIdent, obj)
	h.announcements <- events.Event{
		Type:  events.Delete,
		Value: obj,
	}
}
