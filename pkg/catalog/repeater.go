package catalog

import (
	"reflect"
	"time"

	mapset "github.com/deckarep/golang-set"

	"github.com/openservicemesh/osm/pkg/announcements"
)

const (
	updateAtMostEvery  = 3 * time.Second
	updateAtLeastEvery = 1 * time.Minute
)

// This is a set of announcement types that are explicitly handled
// and there is no need to call the legacy broadcastToAllProxies() for these
var alreadyHandled = mapset.NewSetFromSlice([]interface{}{
	announcements.PodDeleted,
})

// repeater is a goroutine, which rebroadcasts announcements from SMI, Secrets, Endpoints providers etc. to all connected proxies.
func (mc *MeshCatalog) repeater() {
	lastUpdateAt := time.Now().Add(-1 * updateAtMostEvery)
	for {
		cases, caseNames := mc.getCases()
		for {
			if chosenIdx, message, ok := reflect.Select(cases); ok {
				ann, ok := message.Interface().(announcements.Announcement)
				if !ok {
					log.Error().Msgf("Repeater received a interface{} message, which is not an Announcement")
					continue
				}

				log.Trace().Msgf("Handling announcement from %s: %+v", caseNames[chosenIdx], ann)

				mc.handleAnnouncement(ann)

				delta := time.Since(lastUpdateAt)
				if !alreadyHandled.Contains(ann.Type) && delta >= updateAtMostEvery {
					mc.broadcastToAllProxies(ann)
					lastUpdateAt = time.Now()
				}
			}
		}
	}
}

func (mc *MeshCatalog) handleAnnouncement(ann announcements.Announcement) {
	handlers, ok := mc.announcementHandlerPerType[ann.Type]
	if !ok {
		log.Error().Msgf("No handler for announcement of type %+v", ann.Type)
		return
	}

	for _, handler := range handlers {
		if err := handler(ann); err != nil {
			log.Error().Err(err).Msgf("Error handling announcement %s", ann.Type)
		}
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
