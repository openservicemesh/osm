// Package ingress implements functionality to monitor and retrieve Kubernetes Ingress resources.
package ingress

import (
	networkingV1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/tools/cache"

	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("kube-ingress")
)

// APIVersion is the API version for the ingress resource
type APIVersion int

const (
	// IngressNetworkingV1 refers to the networking.k8s.io/v1 ingress API
	IngressNetworkingV1 APIVersion = iota

	// IngressNetworkingV1beta1 refers to the networking.k8s.io/v1beta1 ingress API
	IngressNetworkingV1beta1 APIVersion = iota
)

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	informer       cache.SharedIndexInformer
	cache          cache.Store
	cacheSynced    chan interface{}
	kubeController k8s.Controller
	apiVersion     APIVersion
}

// Monitor is the client interface for K8s Ingress resource
type Monitor interface {
	// GetAPIVersion returns the ingress API version
	GetAPIVersion() APIVersion

	// GetIngressNetworkingV1beta1 returns the ingress resources whose backends correspond to the service
	GetIngressNetworkingV1beta1(service.MeshService) ([]*networkingV1beta1.Ingress, error)
}
