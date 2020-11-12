package kube

import (
	v1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/kubernetes"
)

// podEventTypeRemap provides 1:1 mapping from a Kubernetes Pod event type to an AnnouncementType
// This will be provided to kubernetes.GetKubernetesEventHandlers() so that Announcements
// are dispatched with appropriate type.
var podEventTypeRemap = map[kubernetes.EventType]announcements.AnnouncementType{
	kubernetes.AddEvent:    announcements.PodAdded,
	kubernetes.DeleteEvent: announcements.PodDeleted,
	kubernetes.UpdateEvent: announcements.PodUpdated,
}

func getPodUID(obj interface{}) interface{} {
	if pod, ok := obj.(*v1.Pod); ok {
		return pod.ObjectMeta.UID
	}
	log.Error().Msgf("Expected v1.Pod; Got %+v", obj)
	return nil
}
