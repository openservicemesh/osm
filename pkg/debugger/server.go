package debugger

import (
	"net/http"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GetHandlers implements DebugServer interface and returns the rest of URLs and the handling functions.
func (ds debugServer) GetHandlers() map[string]http.Handler {
	handlers := map[string]http.Handler{
		"/debug/certs":    ds.getCertHandler(),
		"/debug/xds":      ds.getXDSHandler(),
		"/debug/proxy":    ds.getProxies(),
		"/debug/policies": ds.getSMIPoliciesHandler(),
		"/debug/config":   ds.getOSMConfigHandler(),
	}

	// provides an index of the available /debug endpoints
	handlers["/debug"] = ds.getDebugIndex(handlers)

	return handlers
}

// NewDebugServer returns an implementation of DebugServer interface.
func NewDebugServer(certDebugger CertificateManagerDebugger, xdsDebugger XDSDebugger, meshCatalogDebugger MeshCatalogDebugger, kubeConfig *rest.Config) DebugServer {
	return debugServer{
		certDebugger:        certDebugger,
		xdsDebugger:         xdsDebugger,
		meshCatalogDebugger: meshCatalogDebugger,
		kubeConfig:          kubeConfig,
		kubeClient:          kubernetes.NewForConfigOrDie(kubeConfig),
	}
}
