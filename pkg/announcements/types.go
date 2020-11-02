package announcements

type AnnouncementType int

const (
	EndpointDeleted AnnouncementType = iota + 1
)

type Announcement struct {
	Type               AnnouncementType
	ReferencedObjectID interface{}
}
