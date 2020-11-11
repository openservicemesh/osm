package kube

import (
	v1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/kubernetes"
)

var podEventTypeRemap = map[kubernetes.EventType]announcements.AnnouncementType{
	kubernetes.AddEvent:    announcements.PodAdded,
	kubernetes.DeleteEvent: announcements.PodDeleted,
	kubernetes.UpdateEvent: announcements.PodUpdated,
}

var endpointEventTypeRemap = map[kubernetes.EventType]announcements.AnnouncementType{
	kubernetes.AddEvent:    announcements.EndpointAdded,
	kubernetes.DeleteEvent: announcements.EndpointDeleted,
	kubernetes.UpdateEvent: announcements.EndpointUpdated,
}

func getPodUID(obj interface{}) interface{} {
	if pod, ok := obj.(*v1.Pod); ok {
		return pod.ObjectMeta.UID
	}
	log.Error().Msgf("Expected v1.Pod; Got %+v", obj)
	return nil
}

func getPodUIDFromEndpoint(obj interface{}) interface{} {
	if endpoints, ok := obj.(*v1.Endpoints); ok {
		for _, sub := range endpoints.Subsets {
			for _, addr := range sub.Addresses {
				if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
					return addr.TargetRef.UID
				}
			}
		}
	}
	log.Error().Msgf("Expected v1.Pod; Got %+v", obj)
	return nil
}
