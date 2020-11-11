package catalog

import (
	"reflect"
	"time"

	"github.com/openservicemesh/osm/pkg/announcements"
)

const (
	updateAtMostEvery  = 3 * time.Second
	updateAtLeastEvery = 1 * time.Minute
)

// repeater rebroadcasts announcements from SMI, Secrets, Endpoints providers etc. to all connected proxies.
func (mc *MeshCatalog) repeater() {
	lastUpdateAt := time.Now().Add(-1 * updateAtMostEvery)
	for {
		cases, caseNames := mc.getCases()
		for {
			if chosenIdx, message, ok := reflect.Select(cases); ok {
				if ann, ok := message.Interface().(announcements.Announcement); ok {
					mc.handleAnnouncement(ann)

					log.Trace().Msgf("Received announcement from %s", caseNames[chosenIdx])
					delta := time.Since(lastUpdateAt)
					if delta >= updateAtMostEvery {
						mc.broadcastToAllProxies(ann)
						lastUpdateAt = time.Now()
					}
				}
			}
		}
	}
}

func (mc *MeshCatalog) handleAnnouncement(ann announcements.Announcement) {
	if ann.Type == announcements.EndpointDeleted {
		log.Trace().Msgf("Handling announcement: %+v", ann)
		// TODO: implement (https://github.com/openservicemesh/osm/issues/1719)
	}
}

func (mc *MeshCatalog) getCases() ([]reflect.SelectCase, []string) {
	var caseNames []string
	var cases []reflect.SelectCase
	for _, annCh := range mc.getAnnouncementChannels() {
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(annCh.channel)})
		caseNames = append(caseNames, annCh.announcer)
	}
	return cases, caseNames
}

func (mc *MeshCatalog) broadcastToAllProxies(message announcements.Announcement) {
	mc.connectedProxiesLock.Lock()
	for _, connectedEnvoy := range mc.connectedProxies {
		log.Debug().Msgf("[repeater] Broadcast announcement to Envoy with CN %s", connectedEnvoy.proxy.GetCommonName())
		select {
		// send the message if possible - do not block
		case connectedEnvoy.proxy.GetAnnouncementsChannel() <- message:
		default:
		}
	}
	mc.connectedProxiesLock.Unlock()
}
