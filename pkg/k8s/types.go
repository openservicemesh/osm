package k8s

import (
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	policyv1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/logger"
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

// client is the type used to represent the k8s client for the native k8s resources
type client struct {
	meshName     string
	kubeClient   kubernetes.Interface
	policyClient policyv1alpha1Client.Interface
	informers    informerCollection
}
