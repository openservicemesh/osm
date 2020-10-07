package injector

import (
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	envoyBootstrapConfigVolume = "envoy-bootstrap-config-volume"
)

var log = logger.New("sidecar-injector")

// webhook is the type used to represent the webhook for sidecar injection
type webhook struct {
	config         Config
	kubeClient     kubernetes.Interface
	certManager    certificate.Manager
	meshCatalog    catalog.MeshCataloger
	kubeController k8s.Controller
	osmNamespace   string
	cert           certificate.Certificater
	configurator   configurator.Configurator
}

// Config is the type used to represent the config options for the sidecar injection
type Config struct {
	// DefaultInjection defines whether sidecar injection is enabled by default
	DefaultInjection bool

	// ListenPort defines the port on which the sidecar injector listens
	ListenPort int

	InitContainerImage string

	SidecarImage string
}

// JSONPatchOperation is the type used to represenet a JSON Patch operation
type JSONPatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// InitContainer is the type used to represent information about the init container
type InitContainer struct {
	Name  string
	Image string
}

// EnvoySidecarData is the type used to represent information about the Envoy sidecar
type EnvoySidecarData struct {
	Name           string
	Image          string
	EnvoyNodeID    string
	EnvoyClusterID string
}

// Context needed to compose the Envoy bootstrap YAML.
type envoyBootstrapConfigMeta struct {
	EnvoyAdminPort int
	XDSClusterName string
	RootCert       string
	Cert           string
	Key            string

	// Host and port of the Envoy xDS server
	XDSHost string
	XDSPort int
}
