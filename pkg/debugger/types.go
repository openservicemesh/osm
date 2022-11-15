// Package debugger implements functionality to provide information to debug the control plane via the debug HTTP server.
package debugger

import (
	"time"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/models"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/certificate"
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
	computeClient DebuggerInfraClient
	msgBroker     *messaging.Broker
}

// XDSDebugger is an interface providing debugging server with methods introspecting XDS.
type XDSDebugger interface {
	// GetXDSLog returns a log of the XDS responses sent to Envoy proxies. It is keyed by proxy.GetName(), which is
	// of the form <identity>:<uuid>.
	GetXDSLog() map[string]map[envoy.TypeURI][]time.Time
}
type DebuggerInfraClient interface {
	// GetProxyConfig takes the given proxy, port forwards to the pod from this proxy, and returns the envoy config
	GetProxyConfig(proxy *models.Proxy, configType string, kubeConfig *rest.Config) (string, error)

	// ListNamespaces returns the namespaces monitored by the mesh
	ListNamespaces() ([]string, error)

	// GetMeshConfig returns the current MeshConfig
	GetMeshConfig() v1alpha2.MeshConfig
}
