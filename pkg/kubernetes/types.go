// Package kubernetes implements the Kubernetes Controller interface to monitor and retrieve information regarding
// Kubernetes resources such as Namespaces, Services, Pods, Endpoints, and ServiceAccounts.
package kubernetes

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

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
)

// informerCollection is the type holding the collection of informers we keep
type informerCollection map[InformerKey]cache.SharedIndexInformer

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	meshName    string
	kubeClient  kubernetes.Interface
	informers   informerCollection
	cacheSynced chan interface{}
}

// Controller is the controller interface for K8s services
type Controller interface {
	// ListServices returns a list of all (monitored-namespace filtered) services in the mesh
	ListServices() []*corev1.Service

	// ListServiceAccounts returns a list of all (monitored-namespace filtered) service accounts in the mesh
	ListServiceAccounts() []*corev1.ServiceAccount

	// Returns a corev1 Service representation if the MeshService exists in cache, otherwise nil
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

	// ListServiceAccountsForService lists ServiceAccounts associated with the given service
	ListServiceAccountsForService(svc service.MeshService) ([]service.K8sServiceAccount, error)

	// GetEndpoints returns the endpoints for a given service, if found
	GetEndpoints(svc service.MeshService) (*corev1.Endpoints, error)
}
