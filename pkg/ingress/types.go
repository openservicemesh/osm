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

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	informer       cache.SharedIndexInformer
	cache          cache.Store
	cacheSynced    chan interface{}
	kubeController k8s.Controller
}

// Monitor is the client interface for K8s Ingress resource
type Monitor interface {
	// GetIngressResources returns the ingress resources whose backends correspond to the service
	GetIngressResources(service.MeshService) ([]*networkingV1beta1.Ingress, error)
}
