// Package kubernetes implements the Kubernetes Controller interface to monitor and retrieve information regarding
// Kubernetes resources such as Namespaces, Services, Pods, Endpoints, and ServiceAccounts.
package kubernetes

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	networkingV1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	policyV1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/logger"
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
	DefaultKubeEventResyncInterval = 5 * time.Minute

	// providerName is the name of the Kubernetes event provider
	providerName = "Kubernetes"
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
	// IngressesV1Beta1 lookup identifier
	IngressesV1Beta1 InformerKey = "IngressesV1Beta1"
	// IngressesV1 lookup identifier
	IngressesV1 InformerKey = "IngressesV1"
	// EgressPolicies lookup identifier
	EgressPolicies = "EgressPolicies"
)

// informerCollection is the type holding the collection of informers we keep
type informerCollection map[InformerKey]cache.SharedIndexInformer

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	meshName     string
	kubeClient   kubernetes.Interface
	policyClient policyV1alpha1Client.Interface
	informers    informerCollection
	cacheSynced  chan interface{}
}

// BasicController is the controller interface for K8s objects.
type BasicController interface {
	// ListServices returns a list of all (monitored-namespace filtered) services in the mesh
	ListServices() []*corev1.Service

	// ListServiceAccounts returns a list of all (monitored-namespace filtered) service accounts in the mesh
	ListServiceAccounts() []*corev1.ServiceAccount

	// GetService returns a corev1 Service representation if the MeshService exists in cache, otherwise nil
	GetService(svc service.MeshService) *corev1.Service

	// IsMonitoredNamespace returns whether a namespace with the given name is being monitored
	// by the mesh
	IsMonitoredNamespace(string) bool

	// ListMonitoredNamespaces returns the namespaces monitored by the mesh
	ListMonitoredNamespaces() ([]string, error)

	// GetNamespace returns k8s namespace present in cache
	GetNamespace(ns string) *corev1.Namespace

	// ListPods returns a list of pods part of the mesh
	ListPods() []*corev1.Pod

	// ListServiceIdentitiesForService lists ServiceAccounts associated with the given service
	ListServiceIdentitiesForService(svc service.MeshService) ([]identity.K8sServiceAccount, error)

	// GetEndpoints returns the endpoints for a given service, if found
	GetEndpoints(svc service.MeshService) (*corev1.Endpoints, error)

	// IsMetricsEnabled returns true if the pod in the mesh is correctly annotated for prometheus scrapping
	IsMetricsEnabled(*corev1.Pod) bool

	// GetIngressNetworkingV1beta1 returns the networking.k8s.io/v1beta1 ingress resources whose backends correspond to the service
	GetIngressNetworkingV1beta1(service.MeshService) ([]*networkingV1beta1.Ingress, error)

	// GetIngressNetworkingV1 returns the networking.k8s.io/v1 ingress resources whose backends correspond to the service
	GetIngressNetworkingV1(service.MeshService) ([]*networkingV1.Ingress, error)
}

// EgressController implements required methods for the Egress Policy CRD.
type EgressController interface {
	// ListEgressPoliciesForSourceIdentity lists the Egress policies for the given source identity based on service accounts
	ListEgressPoliciesForSourceIdentity(source identity.K8sServiceAccount) []*policyV1alpha1.Egress
}

// Controller implements the full set of controller methods, including those for CRD's.
type Controller interface {
	EgressController
	BasicController
}
