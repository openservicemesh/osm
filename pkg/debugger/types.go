// Package debugger implements functionality to provide information to debug the control plane via the debug HTTP server.
package debugger

import (
	"time"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var log = logger.New("debugger")

// DebugConfig implements the DebugServer interface.
type DebugConfig struct {
	certDebugger        CertificateManagerDebugger
	xdsDebugger         XDSDebugger
	meshCatalogDebugger MeshCatalogDebugger
	proxyRegistry       *registry.ProxyRegistry
	kubeConfig          *rest.Config
	kubeClient          kubernetes.Interface
	kubeController      k8s.Controller
	configurator        configurator.Configurator
}

// CertificateManagerDebugger is an interface with methods for debugging certificate issuance.
type CertificateManagerDebugger interface {
	// ListIssuedCertificates returns the current list of certificates in OSM's cache.
	ListIssuedCertificates() []certificate.Certificater
}

// MeshCatalogDebugger is an interface with methods for debugging Mesh Catalog.
type MeshCatalogDebugger interface {
	// ListSMIPolicies lists the SMI policies detected by OSM.
	ListSMIPolicies() ([]*split.TrafficSplit, []service.K8sServiceAccount, []*spec.HTTPRouteGroup, []*access.TrafficTarget)

	// ListMonitoredNamespaces lists the namespaces that the control plan knows about.
	ListMonitoredNamespaces() []string
}

// XDSDebugger is an interface providing debugging server with methods introspecting XDS.
type XDSDebugger interface {
	// GetXDSLog returns a log of the XDS responses sent to Envoy proxies.
	GetXDSLog() *map[certificate.CommonName]map[envoy.TypeURI][]time.Time
}
