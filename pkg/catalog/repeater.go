package catalog

import (
	"reflect"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/pkg/envoy"
)

const (
	updateAtMostEvery  = 3 * time.Second
	updateAtLeastEvery = 1 * time.Minute
)

// repeater rebroadcasts announcements from SMI, Secrets, Endpoints providers etc. to all connected proxies.
func (sc *MeshCatalog) repeater() {
	lastUpdateAt := time.Now().Add(-1 * updateAtMostEvery)
	for {
		cases, caseNames := sc.getCases()
		for {
			if chosenIdx, message, ok := reflect.Select(cases); ok {
				log.Info().Msgf("[repeater] Received announcement from %s", caseNames[chosenIdx])
				delta := time.Since(lastUpdateAt)
				if delta >= updateAtMostEvery {
					sc.broadcast(message)
					lastUpdateAt = time.Now()
				}
			}
		}
	}
}

func (sc *MeshCatalog) getCases() ([]reflect.SelectCase, []string) {
	var caseNames []string
	var cases []reflect.SelectCase
	for _, channelInterface := range sc.announcementChannels.ToSlice() {
		annCh := channelInterface.(announcementChannel)
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(annCh.channel)})
		caseNames = append(caseNames, annCh.announcer)
	}
	return cases, caseNames
}

func (sc *MeshCatalog) broadcast(message interface{}) {
	for _, proxyInterface := range sc.connectedProxies.ToSlice() {
		envoy := proxyInterface.(*envoy.Proxy)
		log.Debug().Msgf("[repeater] Broadcast announcement to envoy %s", envoy.GetCommonName())
		select {
		// send the message if possible - do not block
		case envoy.GetAnnouncementsChannel() <- message:
		default:
		}
	}
}
