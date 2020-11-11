package announcements

// AnnouncementType is used to record the type of announcement
type AnnouncementType string

func (at AnnouncementType) String() string {
	return string(at)
}

const (
	// EndpointAdded is the type of announcement emitted when we observe an addition of a Kubernetes Endpoint
	EndpointAdded AnnouncementType = "endpoint-added"

	// EndpointDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Endpoint
	EndpointDeleted AnnouncementType = "endpoint-deleted"

	// EndpointUpdated is the type of announcement emitted when we observe an update to a Kubernetes Endpoint
	EndpointUpdated AnnouncementType = "endpoint-updated"

	// PodAdded is the type of announcement emitted when we observe an addition of a Kubernetes Pod
	PodAdded AnnouncementType = "pod-added"

	// PodDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Pod
	PodDeleted AnnouncementType = "pod-deleted"

	// PodUpdated is the type of announcement emitted when we observe an update to a Kubernetes Pod
	PodUpdated AnnouncementType = "pod-updated"
)

// Announcement is a struct for accouncements
type Announcement struct {
	Type               AnnouncementType
	ReferencedObjectID interface{}
}
