package catalog

import (
	"reflect"

	"github.com/golang/glog"
)

func (sc *MeshCatalog) ProxyProcessSignals() {
	glog.Info("Handle proxy broker signalling")

	for {
		select {
		case <-sc.msgBroker.stop:
			glog.Info("Stopping proxy broker")
			sc.msgBroker.Lock()
			for id, msgCh := range sc.msgBroker.proxyChanMap {
				glog.Infof("Closing channel %v for proxy %v", id, msgCh)
				close(msgCh)
			}
			sc.msgBroker.Unlock()
			glog.Info("Proxy broker exiting")
			return
		}
	}
}

func (sc *MeshCatalog) ProxyProcessAnnouncements() {
	var recvCh = []<-chan interface{}{}

	// Subscribe to announcements from SMI, Secrets, Endpoints providers
	recvCh = append(recvCh, sc.meshSpec.GetAnnouncementsChannel())
	recvCh = append(recvCh, sc.certManager.GetAnnouncementsChannel())
	for _, ep := range sc.endpointsProviders {
		recvCh = append(recvCh, ep.GetAnnouncementsChannel())
	}

	cases := make([]reflect.SelectCase, len(recvCh))

	for i, ch := range recvCh {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}

	// Keep receiving
	for {
		chosen, msg, ok := reflect.Select(cases)

		if ok {
			// This is an actual send and not a close on the channel
			// Publish the message to subscribers
			glog.Infof("Received new msg")
			sc.ProxyCount() // This takes a lock
			sc.msgBroker.Lock()
			for _, msgCh := range sc.msgBroker.proxyChanMap {
				select {
				case msgCh <- msg:
					glog.Infof("Publishing msg:[%v] on channel:[%v]", msg, msgCh)

				default:
					// nothing
				}
			}
			sc.msgBroker.Unlock()
		} else {
			glog.Infof("Channel %v closed", recvCh[chosen])
		}
	}

}

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (sc *MeshCatalog) ProxyRegister(id string) <-chan interface{} {
	msgCh := make(chan interface{})
	sc.msgBroker.Lock()
	sc.msgBroker.proxyChanMap[id] = msgCh
	sc.msgBroker.Unlock()
	glog.Infof("Registered proxy: %s, chan: %v", id, msgCh)
	return msgCh
}

func (sc *MeshCatalog) ProxyUnregister(id string) {
	sc.msgBroker.Lock()
	msgCh, ok := sc.msgBroker.proxyChanMap[id]
	sc.msgBroker.Unlock()
	if ok {
		close(msgCh)
		sc.msgBroker.Lock()
		delete(sc.msgBroker.proxyChanMap, id)
		sc.msgBroker.Unlock()
	} else {
		glog.Errorf("Failed to find channel for proxy %v", id)
	}

}

func (sc *MeshCatalog) ProxyCount() int {
	sc.msgBroker.Lock()
	defer sc.msgBroker.Unlock()
	glog.Infof("Proxy count = %v", len(sc.msgBroker.proxyChanMap))
	return len(sc.msgBroker.proxyChanMap)
}
