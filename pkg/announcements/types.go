package announcements

// AnnouncementType is used to record the type of announcement
type AnnouncementType string

func (at AnnouncementType) String() string {
	return string(at)
}

const (
	// PodAdded is the type of announcement emitted when we observe an addition of a Kubernetes Pod
	PodAdded AnnouncementType = "pod-added"

	// PodDeleted the type of announcement emitted when we observe the deletion of a Kubernetes Pod
	PodDeleted AnnouncementType = "pod-deleted"

	// PodUpdated is the type of announcement emitted when we observe an update to a Kubernetes Pod
	PodUpdated AnnouncementType = "pod-updated"
)

// Announcement is a struct for messages between various components of OSM signaling a need for a change in Envoy proxy configuration
type Announcement struct {
	Type               AnnouncementType
	ReferencedObjectID interface{}
}
