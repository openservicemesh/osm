package kubernetes

import (
	"time"

	"github.com/openservicemesh/osm/pkg/logger"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	log = logger.New("kube-events")
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
)

// InformerCollection is the type holding the collection of informers we keep
type InformerCollection map[InformerKey]cache.SharedIndexInformer

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	meshName      string
	kubeClient    kubernetes.Interface
	informers     InformerCollection
	cacheSynced   chan interface{}
	announcements chan interface{}
}
