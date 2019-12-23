package kube

import (
	"reflect"

	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

type handlers struct {
	provider *KubernetesProvider
}

// general resource handlers
func (h handlers) addFunc(obj interface{}) {
	glog.V(9).Infof("[kubernetes] Add event: %+v", obj)
	h.provider.announceChan.In() <- events.Event{
		Type:  events.Create,
		Value: obj,
	}
}

func (h handlers) updateFunc(oldObj, newObj interface{}) {
	glog.V(9).Infof("[kubernetes] Update event %+v", oldObj)
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	h.provider.announceChan.In() <- events.Event{
		Type:  events.Update,
		Value: newObj,
	}
}

func (h handlers) deleteFunc(obj interface{}) {
	glog.V(9).Infof("[kubernetes] Delete event: %+v", obj)
	h.provider.announceChan.In() <- events.Event{
		Type:  events.Delete,
		Value: obj,
	}
}
