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
type EventType int

const (
	// CreateEvent is a type of a Kubernetes API event.
	CreateEvent EventType = iota + 1

	// UpdateEvent is a type of a Kubernetes API event.
	UpdateEvent

	// DeleteEvent is a type of a Kubernetes API event.
	DeleteEvent
)

const (
	// DefaultKubeEventResyncInterval is the default resync interval for k8s events
	DefaultKubeEventResyncInterval = 30 * time.Second

	// ProviderName is used for provider logging
	ProviderName = "Kubernetes"
)

// Event is the combined type and actual object we received from Kubernetes
type Event struct {
	Type  EventType
	Value interface{}
}

// InformerKey stores the different Informers we keep for K8s resources
type InformerKey string

const (
	// Namespaces lookup identifier
	Namespaces InformerKey = "Namespaces"
	// Services lookup identifier
	Services InformerKey = "Services"
	// Pods lookup identifier
	Pods InformerKey = "Pods"
)

// InformerCollection is the type holding the collection of informers we keep
type InformerCollection map[InformerKey]cache.SharedIndexInformer

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	meshName      string
	kubeClient    kubernetes.Interface
	informers     InformerCollection
	cacheSynced   chan interface{}
	announcements map[InformerKey]chan interface{}
}

// Controller is the controller interface for K8s services
type Controller interface {
	// ListServices returns a list of all (monitored-namespace filtered) services in the mesh
	ListServices() []*corev1.Service

	// Returns a corev1 Service representation if the MeshService exists in cache, otherwise nil
	GetService(svc service.MeshService) *corev1.Service

	// IsMonitoredNamespace returns whether a namespace with the given name is being monitored
	// by the mesh
	IsMonitoredNamespace(string) bool

	// ListMonitoredNamespaces returns the namespaces monitored by the mesh
	ListMonitoredNamespaces() ([]string, error)

	// GetNamespace returns k8s namespace present in cache
	GetNamespace(ns string) *corev1.Namespace

	// Returns the announcement channel for a certain Informer ID
	GetAnnouncementsChannel(informerID InformerKey) <-chan interface{}

	// ListPods returns a list of pods part of the mesh
	ListPods() []*corev1.Pod
}
