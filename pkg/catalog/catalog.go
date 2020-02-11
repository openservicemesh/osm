package catalog

import (
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/smi"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(meshSpec smi.MeshSpec, certManager certificate.Manager, stop <-chan struct{}, endpointsProviders ...endpoint.Provider) *MeshCatalog {
	glog.Info("[catalog] Create a new Service MeshCatalog.")
	serviceCatalog := MeshCatalog{
		endpointsProviders: endpointsProviders,
		meshSpec:           meshSpec,
		certManager:        certManager,

		// Caches
		servicesCache:    make(map[endpoint.ServiceName][]endpoint.Endpoint),
		certificateCache: make(map[endpoint.ServiceName]certificate.Certificater),

		// Message broker / broadcaster for all connected proxies
		msgBroker: newMsgBroker(stop),
	}

	serviceCatalog.run(stop)

	return &serviceCatalog
}

func newMsgBroker(stop <-chan struct{}) *MsgBroker {
	return &MsgBroker{
		stop:         stop,
		proxyChanMap: make(map[envoy.ProxyID]chan interface{}),
	}
}

// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
func (sc *MeshCatalog) RegisterNewEndpoint(smi.ClientIdentity) {
	// TODO(draychev): implement
	panic("NotImplemented")
}

func (sc *MeshCatalog) run(stop <-chan struct{}) {
	glog.Info("[catalog] Running the Service MeshCatalog...")
	go sc.broadcastAnnouncementToProxies()
	go sc.handleBrokerSingals()
}
