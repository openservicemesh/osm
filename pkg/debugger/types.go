package debugger

import (
	"net/http"
	"time"

	"k8s.io/client-go/rest"

	"k8s.io/client-go/kubernetes"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/logger"
)

var log = logger.New("debugger")

// debugServer implements the DebugServer interface.
type debugServer struct {
	certDebugger        CertificateManagerDebugger
	xdsDebugger         XDSDebugger
	meshCatalogDebugger MeshCatalogDebugger
	kubeConfig          *rest.Config
	kubeClient          kubernetes.Interface
}

// CertificateManagerDebugger is an interface with methods for debugging certificate issuance.
type CertificateManagerDebugger interface {
	// ListIssuedCertificates returns the current list of certificates in OSM's cache.
	ListIssuedCertificates() []certificate.Certificater
}

// MeshCatalogDebugger is an interface with methods for debugging Mesh Catalog.
type MeshCatalogDebugger interface {
	// ListExpectedProxies lists the Envoy proxies yet to connect and the time their XDS certificate was issued.
	ListExpectedProxies() map[certificate.CommonName]time.Time

	// ListConnectedProxies lists the Envoy proxies already connected and the time they first connected.
	ListConnectedProxies() map[certificate.CommonName]*envoy.Proxy

	// ListDisconnectedProxies lists the Envoy proxies disconnected and the time last seen.
	ListDisconnectedProxies() map[certificate.CommonName]time.Time
}

// XDSDebugger is an interface providing debugging server with methods introspecting XDS.
type XDSDebugger interface {
	// GetXDSLog returns a log of the XDS responses sent to Envoy proxies.
	GetXDSLog() *map[certificate.CommonName]map[envoy.TypeURI][]time.Time
}

// DebugServer is the interface of the Debug HTTP server.
type DebugServer interface {
	// GetHandlers returns the HTTP handlers available for the debug server.
	GetHandlers() map[string]http.Handler
}
