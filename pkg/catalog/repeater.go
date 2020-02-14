package catalog

import (
	"reflect"
	"time"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/golang/glog"
)

const (
	updateAtMostEvery  = 3 * time.Second
	updateAtLeastEvery = 1 * time.Minute
)

// repeater rebroadcasts announcements from SMI, Secrets, Endpoints providers etc. to all connected proxies.
func (sc *MeshCatalog) repeater() {
	lastUpdateAt := time.Now().Add(-5 * time.Minute)
	for {
		cases, caseNames := sc.getCases()
		for {
			if chosenIdx, message, ok := reflect.Select(cases); ok {
				glog.Infof("[repeater] Received announcement from %s", caseNames[chosenIdx])
				delta := time.Now().Sub(lastUpdateAt)
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
		proxy := proxyInterface.(*envoy.Proxy)
		glog.V(level.Debug).Infof("[repeater] Broadcast announcement to proxy %s", proxy.GetCommonName())
		select {
		// send the message if possible - do not block
		case proxy.GetAnnouncementsChannel() <- message:
		default:
		}
	}
}
