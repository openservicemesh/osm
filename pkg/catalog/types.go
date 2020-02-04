package catalog

import (
	"sync"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/smi"
)

type MsgBroker struct {
	sync.Mutex

	stop       <-chan struct{}
	register   <-chan chan interface{}
	unregister <-chan chan interface{}

	proxyChanMap map[string]chan interface{}
}

// MeshCatalog is the struct for the service catalog
type MeshCatalog struct {
	sync.Mutex

	endpointsProviders []endpoint.Provider
	meshSpec           smi.MeshSpec
	certManager        certificate.Manager

	// Caches
	servicesCache    map[endpoint.ServiceName][]endpoint.Endpoint
	certificateCache map[endpoint.ServiceName]certificate.Certificater

	// Proxy broker
	msgBroker *MsgBroker
}

// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// ListEndpoints constructs a map of service to weighted handlers with all endpoints the given Envoy proxy should be aware of.
	ListEndpoints(smi.ClientIdentity) (map[endpoint.ServiceName][]endpoint.WeightedService, error)

	// ListTrafficRoutes constructs a list of all the traffic policies /routes the given Envoy proxy should be aware of.
	ListTrafficRoutes(smi.ClientIdentity) ([]endpoint.TrafficTargetPolicies, error)

	// GetCertificateForService returns the SSL Certificate for the given service.
	// This certificate will be used for service-to-service mTLS.
	GetCertificateForService(endpoint.ServiceName) (certificate.Certificater, error)

	//
	// Proxy messaging interfaces
	//

	// ProxyProcessSignals handles signals sent to the proxy broker, ex. to stop
	ProxyProcessSignals()

	// ProxyProcessAnnouncements processes channel announcements and relays it to subscribed proxies
	ProxyProcessAnnouncements()

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	ProxyRegister(string) <-chan interface{}

	// ProxyUnregister unregisters an existing proxy from the service mesh catalog
	ProxyUnregister(string)

	// ProxyCount returns the count of proxies subscribed with the service mesh catalog
	ProxyCount() int
}
