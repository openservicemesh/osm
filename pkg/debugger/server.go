package debugger

import (
	"net/http"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/open-service-mesh/osm/pkg/configurator"
)

// GetHandlers implements DebugServer interface and returns the rist of URLs and the handling functions.
func (ds debugServer) GetHandlers() map[string]http.Handler {
	handlers := map[string]http.Handler{
		"/debug/certs":    ds.getCertHandler(),
		"/debug/xds":      ds.getXDSHandler(),
		"/debug/proxy":    ds.getProxies(),
		"/debug/policies": ds.getSMIPoliciesHandler(),
	}

	// provides an index of the available /debug endpoints
	handlers["/debug"] = ds.getDebugIndex(handlers)

	return handlers
}

// NewDebugServer returns an implementation of DebugServer interface.
func NewDebugServer(certDebugger CertificateManagerDebugger, xdsDebugger XDSDebugger, meshCatalogDebugger MeshCatalogDebugger, kubeConfig *rest.Config, kubeClient kubernetes.Interface, cfg configurator.Configurator) DebugServer {
	return debugServer{
		certDebugger:        certDebugger,
		xdsDebugger:         xdsDebugger,
		meshCatalogDebugger: meshCatalogDebugger,
		kubeClient:          kubeClient,

		// We need the Kubernetes config to be able to establish port forwarding to the Envoy pod we want to debug.
		kubeConfig: kubeConfig,
	}
}
