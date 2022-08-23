// Package k8s implements the Kubernetes Controller interface to monitor and retrieve information regarding
// Kubernetes resources such as Namespaces, Services, Pods, Endpoints, and ServiceAccounts.
package k8s

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	policyv1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/service"
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
)

const (
	// kindSvcAccount is the ServiceAccount kind
	kindSvcAccount = "ServiceAccount"
)

// Client is the type used to represent the k8s client for the native k8s resources
type Client struct {
	policyClient   policyv1alpha1Client.Interface
	informers      *informers.InformerCollection
	msgBroker      *messaging.Broker
	osmNamespace   string
	meshConfigName string
}

// Controller is the controller interface for K8s services
type Controller interface {
	PassthroughInterface
	// ListServices returns a list of all (monitored-namespace filtered) services in the mesh
	ListServices() []*corev1.Service

	// ListServiceAccounts returns a list of all (monitored-namespace filtered) service accounts in the mesh
	ListServiceAccounts() []*corev1.ServiceAccount

	// GetService returns a corev1 Service representation if the MeshService exists in cache, otherwise nil
	GetService(service.MeshService) *corev1.Service

	// IsMonitoredNamespace returns whether a namespace with the given name is being monitored
	// by the mesh
	IsMonitoredNamespace(string) bool

	// ListMonitoredNamespaces returns the namespaces monitored by the mesh
	ListMonitoredNamespaces() ([]string, error)

	// GetNamespace returns k8s namespace present in cache
	GetNamespace(string) *corev1.Namespace

	// ListPods returns a list of pods part of the mesh
	ListPods() []*corev1.Pod

	// ListServiceIdentitiesForService lists ServiceAccounts associated with the given service
	ListServiceIdentitiesForService(service.MeshService) ([]identity.K8sServiceAccount, error)

	// GetEndpoints returns the endpoints for a given service, if found
	GetEndpoints(service.MeshService) (*corev1.Endpoints, error)

	// GetPodForProxy returns the pod for the given proxy
	GetPodForProxy(*envoy.Proxy) (*v1.Pod, error)

	ServiceToMeshServices(svc corev1.Service) []service.MeshService
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
	GetMeshConfig() configv1alpha2.MeshConfig
	GetOSMNamespace() string
	UpdateIngressBackendStatus(obj *policyv1alpha1.IngressBackend) (*policyv1alpha1.IngressBackend, error)
	UpdateUpstreamTrafficSettingStatus(obj *policyv1alpha1.UpstreamTrafficSetting) (*policyv1alpha1.UpstreamTrafficSetting, error)

	GetTargetPortForServicePort(types.NamespacedName, uint16) (uint16, error)

	// ListEgressPoliciesForSourceIdentity lists the Egress policies for the given source identity
	ListEgressPoliciesForSourceIdentity(identity.K8sServiceAccount) []*policyv1alpha1.Egress

	// GetIngressBackendPolicy returns the IngressBackend policy for the given backend MeshService
	GetIngressBackendPolicy(service.MeshService) *policyv1alpha1.IngressBackend

	// ListRetryPolicies returns the Retry policies for the given source identity
	ListRetryPolicies(identity.K8sServiceAccount) []*policyv1alpha1.Retry

	// GetUpstreamTrafficSetting returns the UpstreamTrafficSetting resource that matches the given options
	GetUpstreamTrafficSetting(trafficpolicy.UpstreamTrafficSettingGetOpt) *policyv1alpha1.UpstreamTrafficSetting
}
