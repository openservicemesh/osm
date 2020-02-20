package catalog

import (
	"fmt"
	"net/http"
	"os"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/smi"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(meshSpec smi.MeshSpec, certManager certificate.Manager, stop <-chan struct{}, endpointsProviders ...endpoint.Provider) *MeshCatalog {
	glog.Info("[catalog] Create a new Service MeshCatalog.")
	sc := MeshCatalog{
		endpointsProviders: endpointsProviders,
		meshSpec:           meshSpec,
		certManager:        certManager,

		servicesCache:        make(map[endpoint.ServiceName][]endpoint.Endpoint),
		certificateCache:     make(map[endpoint.ServiceName]certificate.Certificater),
		connectedProxies:     mapset.NewSet(),
		announcementChannels: mapset.NewSet(),
		serviceAccountsCache: make(map[endpoint.ServiceAccount][]endpoint.ServiceName),
		targetServicesCache:  make(map[endpoint.ServiceName][]endpoint.ServiceName),
	}

	for _, announcementChannel := range sc.getAnnouncementChannels() {
		sc.announcementChannels.Add(announcementChannel)

	}

	go sc.repeater()
	return &sc
}

func (sc *MeshCatalog) GetDebugInfo() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(draychev): convert to CLI flag
		if value, ok := os.LookupEnv("SMC_ENABLE_DEBUG"); ok && value == "true" {
			_, _ = fmt.Fprintf(w, "hello\n")
		}
	})
}

// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
func (sc *MeshCatalog) RegisterNewEndpoint(smi.ClientIdentity) {
	// TODO(draychev): implement
	panic("NotImplemented")
}

func (sc *MeshCatalog) getAnnouncementChannels() []announcementChannel {
	ticking := make(chan interface{})
	announcementChannels := []announcementChannel{
		{"MeshSpec", sc.meshSpec.GetAnnouncementsChannel()},
		{"CertManager", sc.certManager.GetAnnouncementsChannel()},
		{"Ticker", ticking},
	}
	for _, ep := range sc.endpointsProviders {
		annCh := announcementChannel{ep.GetID(), ep.GetAnnouncementsChannel()}
		announcementChannels = append(announcementChannels, annCh)
	}

	go func() {
		ticker := time.NewTicker(updateAtLeastEvery)
		select {
		case tick := <-ticker.C:
			ticking <- tick
		}
	}()
	return announcementChannels
}
