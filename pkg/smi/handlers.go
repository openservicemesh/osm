package smi

import (
	"reflect"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

type handlers struct {
	Client
}

// general resource handlers
func (h handlers) addFunc(obj interface{}) {
	h.announcements <- events.Event{
		Type:  events.Create,
		Value: obj,
	}
}

func (h handlers) updateFunc(oldObj, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	h.announcements <- events.Event{
		Type:  events.Update,
		Value: newObj,
	}
}

func (h handlers) deleteFunc(obj interface{}) {
	h.announcements <- events.Event{
		Type:  events.Delete,
		Value: obj,
	}
}
