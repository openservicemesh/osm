package debugger

import (
	"net/http"
	"time"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha2"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
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

	// ListSMIPolicies lists the SMI policies detected by OSM.
	ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.NamespacedServiceAccount, []*spec.HTTPRouteGroup, []*target.TrafficTarget, []*corev1.Service)
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
