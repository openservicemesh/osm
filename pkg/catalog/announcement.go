package catalog

// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
func (sc *MeshCatalog) GetAnnouncementChannel() chan interface{} {
	return sc.announcements
}
