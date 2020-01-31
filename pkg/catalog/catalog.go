package catalog

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/logging"
	"github.com/deislabs/smc/pkg/smi"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(meshSpec smi.MeshSpec, certManager certificate.Manager, stop <-chan struct{}, endpointsProviders ...endpoint.Provider) *MeshCatalog {
	glog.Info("[catalog] Create a new Service MeshCatalog.")
	serviceCatalog := MeshCatalog{
		announcements: make(chan interface{}),

		endpointsProviders: endpointsProviders,
		meshSpec:           meshSpec,
		certManager:        certManager,

		// Caches
		servicesCache:    make(map[endpoint.ServiceName][]endpoint.Endpoint),
		certificateCache: make(map[endpoint.ServiceName]certificate.Certificater),
	}

	serviceCatalog.run(stop)

	return &serviceCatalog
}

// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
func (sc *MeshCatalog) RegisterNewEndpoint(smi.ClientIdentity) {
	// TODO(draychev): implement
	panic("NotImplemented")
}

func (sc *MeshCatalog) run(stop <-chan struct{}) {
	glog.Info("[catalog] Running the Service MeshCatalog...")
	allAnnouncementChans := []<-chan interface{}{
		// TODO(draychev): does the stop channel need to be here too?
		// stop,
		sc.certManager.GetSecretsChangeAnnouncementChan(),
	}
	for _, endpointProvider := range sc.endpointsProviders {
		fmt.Printf("--> %+v", endpointProvider)
		if endpointProvider != nil {
			allAnnouncementChans = append(allAnnouncementChans, endpointProvider.GetAnnouncementsChannel())
		}

	}

	cases := make([]reflect.SelectCase, len(allAnnouncementChans))

	go func() {
		glog.V(log.LvlTrace).Info("[catalog] Start announcements loop.")
		for {
			for i, ch := range allAnnouncementChans {
				cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
			}
			idx, _, ok := reflect.Select(cases)
			if !ok && idx == 0 {
				glog.Info("[catalog] Stop announcements loop.")
				return
			} else if !ok {
				glog.Error("[catalog] Announcement channel is closed.")
				continue
			}
			sc.announcements <- struct{}{}
			time.Sleep(1 * time.Second)
		}
	}()

	// NOTE(draychev): helpful while developing alpha MVP -- remove before releasing beta version.
	go func() {
		glog.V(log.LvlTrace).Info("[catalog] Start periodic cache refresh loop.")
		counter := 0
		for {
			select {
			case _, ok := <-stop:
				if !ok {
					glog.Info("[catalog] Stop periodic cache refresh loop.")
					return
				}
			default:
				glog.V(log.LvlTrace).Infof("----- Service MeshCatalog Periodic Cache Refresh %d -----", counter)
				counter++
				sc.refreshCache()
				// Announce so we trigger refresh of all connected Envoy proxies.
				sc.announcements <- struct{}{}
				time.Sleep(5 * time.Second)
			}

		}
	}()
}
