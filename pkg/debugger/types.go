// Package debugger implements functionality to provide information to debug the control plane via the debug HTTP server.
package debugger

import (
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
)

var log = logger.New("debugger")

// DebugConfig implements the DebugServer interface.
type DebugConfig struct {
	certDebugger  *certificate.Manager
	xdsDebugger   XDSDebugger
	proxyRegistry *registry.ProxyRegistry
	kubeConfig    *rest.Config
	kubeClient    kubernetes.Interface
	computeClient compute.Interface
	msgBroker     *messaging.Broker
}

// XDSDebugger is an interface providing debugging server with methods introspecting XDS.
type XDSDebugger interface {
	// GetXDSLog returns a log of the XDS responses sent to Envoy proxies. It is keyed by proxy.GetName(), which is
	// of the form <identity>:<uuid>.
	GetXDSLog() map[string]map[envoy.TypeURI][]time.Time
}
