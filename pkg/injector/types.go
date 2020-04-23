package injector

import (
	"k8s.io/client-go/kubernetes"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/logger"
	"github.com/open-service-mesh/osm/pkg/namespace"
)

const (
	// OSM Annotations
	annotationInject  = "openservicemesh.io/sidecar-injection"
	annotationService = "openservicemesh.io/osm-service"

	envoyBootstrapConfigVolume = "envoy-bootstrap-config-volume"
)

var (
	log = logger.New("sidecar-injector")
)

// Webhook is the type used to represent the webhook for sidecar injection
type Webhook struct {
	config              Config
	kubeClient          kubernetes.Interface
	certManager         certificate.Manager
	meshCatalog         catalog.MeshCataloger
	namespaceController namespace.Controller
	osmNamespace        string
}

// Config is the type used to represent the config options for the sidecar injection
type Config struct {
	// EnableTLS defines whether TLS is used or not
	EnableTLS bool

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

// InitContainerData is the type used to represent information about the init container
type InitContainerData struct {
	Name  string
	Image string
}

// EnvoySidecarData is the type used to represent information about the Envoy sidecar
type EnvoySidecarData struct {
	Name    string
	Image   string
	Service string
}
