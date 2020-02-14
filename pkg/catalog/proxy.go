package catalog

import (
	"net"
	"reflect"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/envoy"
)

func (sc *MeshCatalog) handleBrokerSingals() {
	glog.Info("Handle proxy broker signalling")

	for {
		select {
		case <-sc.messageBroker.stop:
			glog.Info("Stopping proxy broker")
			sc.messageBroker.Lock()
			for id, announcements := range sc.messageBroker.proxyChanMap {
				glog.Infof("Closing channel %v for proxy %v", id, announcements)
				close(announcements)
			}
			sc.messageBroker.Unlock()
			glog.Info("Proxy broker exiting")
			return
		}
	}
}

func (sc *MeshCatalog) broadcastAnnouncementToProxies() {
	var changeAnnouncements = []<-chan interface{}{}

	// Subscribe to announcements from SMI, Secrets, Endpoints providers
	changeAnnouncements = append(changeAnnouncements, sc.meshSpec.GetAnnouncementsChannel())
	changeAnnouncements = append(changeAnnouncements, sc.certManager.GetAnnouncementsChannel())
	for _, ep := range sc.endpointsProviders {
		changeAnnouncements = append(changeAnnouncements, ep.GetAnnouncementsChannel())
	}

	cases := make([]reflect.SelectCase, len(changeAnnouncements))

	for i, ch := range changeAnnouncements {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}

	for {
		chosen, message, ok := reflect.Select(cases)

		if ok {
			// This is an actual send and not a close on the channel
			// Publish the message to subscribers
			glog.Infof("Received new message")
			sc.messageBroker.Lock()
			for id, announcements := range sc.messageBroker.proxyChanMap {
				select {
				case announcements <- message:
					glog.Infof("Publishing announcement:[%v], proxy id:[%v], channel:[%v]", message, id, announcements)
				}
			}
			sc.messageBroker.Unlock()
		} else {
			glog.Infof("Channel %v closed", changeAnnouncements[chosen])
		}
	}

}

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (sc *MeshCatalog) RegisterProxy(cn certificate.CommonName, ip net.IP) envoy.Proxyer {
	announcements := make(chan interface{})
	proxy := envoy.NewProxy(cn, ip, announcements)
	sc.messageBroker.Lock()
	sc.messageBroker.proxyChanMap[proxy.GetID()] = announcements
	sc.messageBroker.Unlock()
	glog.Infof("Registered proxy: CN=%v, ip=%v, id=%s, channel= %v", proxy.GetCommonName(), proxy.GetIP(), proxy.GetID(), proxy.GetAnnouncementsChannel())
	return proxy
}

func (sc *MeshCatalog) UnregisterProxy(id envoy.ProxyID) {
	sc.messageBroker.Lock()
	announcements, ok := sc.messageBroker.proxyChanMap[id]
	sc.messageBroker.Unlock()
	if ok {
		close(announcements)
		sc.messageBroker.Lock()
		delete(sc.messageBroker.proxyChanMap, id)
		sc.messageBroker.Unlock()
	} else {
		glog.Errorf("Failed to find channel for proxy %v", id)
	}

}

func (sc *MeshCatalog) countRegisteredProxies() int {
	sc.messageBroker.Lock()
	defer sc.messageBroker.Unlock()
	glog.Infof("Proxy count = %v", len(sc.messageBroker.proxyChanMap))
	return len(sc.messageBroker.proxyChanMap)
}
