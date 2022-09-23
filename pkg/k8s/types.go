// Package k8s implements the Kubernetes Controller interface to monitor and retrieve information regarding
// Kubernetes resources such as Namespaces, Services, Pods, Endpoints, and ServiceAccounts.
package k8s

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	configv1alpha2Client "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	policyv1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/models"
)

var (
	log = logger.New("kube-controller")
)

// EventType is the type of event we have received from Kubernetes
type EventType string

func (et EventType) String() string {
	return string(et)
}

const (
	// AddEvent is a type of a Kubernetes API event.
	AddEvent EventType = "ADD"

	// UpdateEvent is a type of a Kubernetes API event.
	UpdateEvent EventType = "UPDATE"

	// DeleteEvent is a type of a Kubernetes API event.
	DeleteEvent EventType = "DELETE"
)

const (
	// DefaultKubeEventResyncInterval is the default resync interval for k8s events
	// This is set to 0 because we do not need resyncs from k8s client, and have our
	// own Ticker to turn on periodic resyncs.
	DefaultKubeEventResyncInterval = 0 * time.Second
)

// InformerKey stores the different Informers we keep for K8s resources
type InformerKey string

const (
	// Namespaces lookup identifier
	Namespaces InformerKey = "Namespaces"
	// Services lookup identifier
	Services InformerKey = "Services"
	// Pods lookup identifier
	Pods InformerKey = "Pods"
	// Endpoints lookup identifier
	Endpoints InformerKey = "Endpoints"
	// ServiceAccounts lookup identifier
	ServiceAccounts InformerKey = "ServiceAccounts"
	// MeshConfig lookup identifier
	MeshConfig InformerKey = "MeshConfig"
	// MeshRootCertificate lookup identifier
	MeshRootCertificate InformerKey = "MeshRootCertificate"
	// ExtensionService lookup identifier
	ExtensionService InformerKey = "ExtensionService"
	// Egress lookup identifier
	Egress InformerKey = "Egress"
	// IngressBackend lookup identifier
	IngressBackend InformerKey = "IngressBackend"
	// Retry lookup identifier
	Retry InformerKey = "Retry"
	// UpstreamTrafficSetting lookup identifier
	UpstreamTrafficSetting InformerKey = "UpstreamTrafficSetting"
	// Telemetry lookup identifier
	Telemetry InformerKey = "Telemetry"
	// TrafficSplit lookup identifier
	TrafficSplit InformerKey = "TrafficSplit"
	// HTTPRouteGroup lookup identifier
	HTTPRouteGroup InformerKey = "HTTPRouteGroup"
	// TCPRoute lookup identifier
	TCPRoute InformerKey = "TCPRoute"
	// TrafficTarget lookup identifier
	TrafficTarget InformerKey = "TrafficTarget"
)

// Client is the type used to represent the k8s client for the native k8s resources
type Client struct {
	policyClient   policyv1alpha1Client.Interface
	configClient   configv1alpha2Client.Interface
	kubeClient     kubernetes.Interface
	informers      *informers.InformerCollection
	msgBroker      *messaging.Broker
	osmNamespace   string
	meshConfigName string
}

// Controller is the controller interface for K8s services
type Controller interface {
	PassthroughInterface
	// GetSecret returns the secret for a given namespace and secret name
	GetSecret(string, string) *models.Secret

	// ListSecrets returns a list of secrets
	ListSecrets() []*models.Secret

	// UpdateSecret updates the given secret
	UpdateSecret(context.Context, *models.Secret) error

	// ListServices returns a list of all (monitored-namespace filtered) services in the mesh
	ListServices() []*corev1.Service

	// ListServiceAccounts returns a list of all (monitored-namespace filtered) service accounts in the mesh
	ListServiceAccounts() []*corev1.ServiceAccount

	// GetService returns a corev1 Service representation if the MeshService exists in cache, otherwise nil
	GetService(name, namespace string) *corev1.Service

	// ListNamespaces returns the namespaces monitored by the mesh
	ListNamespaces() ([]*corev1.Namespace, error)

	// GetNamespace returns k8s namespace present in cache
	GetNamespace(string) *corev1.Namespace

	// ListPods returns a list of pods part of the mesh
	ListPods() []*corev1.Pod

	// GetEndpoints returns the endpoints for a given service, if found
	GetEndpoints(name, namespace string) (*corev1.Endpoints, error)

	// GetPodForProxy returns the pod that the given proxy is attached to, based on the UUID and service identity.
	GetPodForProxy(proxy *models.Proxy) (*corev1.Pod, error)
}

// PassthroughInterface is the interface for methods that are implemented by the k8s.Client, but are not considered
// specific to kubernetes, and thus do not need further abstraction, and can be used throughout the code base without
// fear of coupling to k8s. That is to say that another implementation that may exist for a bare metal control plane
// would be expected to implement these methods as well. In this way, for instance, a *policyv1alpha1.IngressBackend
// is not considered an object uniquely specific to kubernetes, but an object tied to OSM.
// A good rule of thumb is that any CRUD operations (get,delete,create,update,etc) on CRD's we define belong here, since
// we control the definition it is reasonable to assume a non-k8s implementation would be obligated to implement as
// well.
type PassthroughInterface interface {
	// IsMonitoredNamespace returns whether a namespace with the given name is being monitored
	// by the mesh
	IsMonitoredNamespace(string) bool

	GetMeshConfig() configv1alpha2.MeshConfig
	GetMeshRootCertificate(mrcName string) *configv1alpha2.MeshRootCertificate
	ListMeshRootCertificates() ([]*configv1alpha2.MeshRootCertificate, error)
	UpdateMeshRootCertificate(obj *configv1alpha2.MeshRootCertificate) (*configv1alpha2.MeshRootCertificate, error)
	UpdateMeshRootCertificateStatus(obj *configv1alpha2.MeshRootCertificate) (*configv1alpha2.MeshRootCertificate, error)
	GetOSMNamespace() string
	UpdateIngressBackendStatus(obj *policyv1alpha1.IngressBackend) (*policyv1alpha1.IngressBackend, error)
	UpdateUpstreamTrafficSettingStatus(obj *policyv1alpha1.UpstreamTrafficSetting) (*policyv1alpha1.UpstreamTrafficSetting, error)

	// ListEgressPolicies lists the all Egress policies
	ListEgressPolicies() []*policyv1alpha1.Egress

	// ListIngressBackends lists the all IngressBackend policies
	ListIngressBackendPolicies() []*policyv1alpha1.IngressBackend

	// ListRetryPolicies returns the all retry policies
	ListRetryPolicies() []*policyv1alpha1.Retry

	// ListUpstreamTrafficSettings returns all UpstreamTrafficSetting resources
	ListUpstreamTrafficSettings() []*policyv1alpha1.UpstreamTrafficSetting

	// GetUpstreamTrafficSetting returns the UpstreamTrafficSetting resources with namespaced name
	GetUpstreamTrafficSetting(*types.NamespacedName) *policyv1alpha1.UpstreamTrafficSetting

	// ListTrafficSplits lists SMI TrafficSplit resources
	ListTrafficSplits() []*split.TrafficSplit

	// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
	ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup

	// GetHTTPRouteGroup returns an SMI HTTPRouteGroup resource given its name of the form <namespace>/<name>
	GetHTTPRouteGroup(string) *spec.HTTPRouteGroup

	// ListTCPTrafficSpecs lists SMI TCPRoute resources
	ListTCPTrafficSpecs() []*spec.TCPRoute

	// GetTCPRoute returns an SMI TCPRoute resource given its name of the form <namespace>/<name>
	GetTCPRoute(string) *spec.TCPRoute

	// ListTrafficTargets lists SMI TrafficTarget resources. An optional filter can be applied to filter the
	// returned list
	ListTrafficTargets() []*access.TrafficTarget

	// GetTelemetryPolicy returns the Telemetry policy for the given proxy instance.
	// It returns the most specific match if multiple matching policies exist, in the following
	// order of preference: 1. selector match, 2. namespace match, 3. global match
	GetTelemetryPolicy(*models.Proxy) *policyv1alpha1.Telemetry
}
