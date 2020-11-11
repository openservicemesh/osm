package announcements

// AnnouncementType is used to record the type of annoucements
type AnnouncementType int

const (
	// EndpointDeleted is to observe endpoint deletion announcements
	EndpointDeleted AnnouncementType = iota + 1
)

// Announcement is a struct for accouncements
type Announcement struct {
	Type               AnnouncementType
	ReferencedObjectID interface{}
}
