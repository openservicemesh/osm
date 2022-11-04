package debugger

import (
	"net/http"
	"net/http/pprof"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// GetHandlers implements DebugConfig interface and returns the rest of URLs and the handling functions.
func (ds DebugConfig) GetHandlers() map[string]http.Handler {
	handlers := map[string]http.Handler{
		"/debug/certs":         ds.getCertHandler(),
		"/debug/xds":           ds.getXDSHandler(),
		"/debug/proxy":         ds.getProxies(),
		"/debug/namespaces":    ds.getMonitoredNamespacesHandler(),
		"/debug/feature-flags": ds.getFeatureFlags(),

		// Pprof handlers
		"/debug/pprof/":        http.HandlerFunc(pprof.Index),
		"/debug/pprof/cmdline": http.HandlerFunc(pprof.Cmdline),
		"/debug/pprof/profile": http.HandlerFunc(pprof.Profile),
		"/debug/pprof/symbol":  http.HandlerFunc(pprof.Symbol),
		"/debug/pprof/trace":   http.HandlerFunc(pprof.Trace),
	}

	// provides an index of the available /debug endpoints
	handlers["/debug"] = ds.getDebugIndex(handlers)

	return handlers
}

// NewDebugConfig returns an implementation of DebugConfig interface.
func NewDebugConfig(certDebugger *certificate.Manager, xdsDebugger XDSDebugger,
	proxyRegistry *registry.ProxyRegistry, kubeConfig *rest.Config, kubeClient kubernetes.Interface,
	computeClient catalog.Interface, msgBroker *messaging.Broker) DebugConfig {
	return DebugConfig{
		certDebugger:  certDebugger,
		xdsDebugger:   xdsDebugger,
		proxyRegistry: proxyRegistry,
		kubeClient:    kubeClient,
		computeClient: computeClient,

		// We need the Kubernetes config to be able to establish port forwarding to the Envoy pod we want to debug.
		kubeConfig: kubeConfig,

		msgBroker: msgBroker,
	}
}
