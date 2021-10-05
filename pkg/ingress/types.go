// Package ingress implements functionality to monitor and retrieve Kubernetes Ingress resources.
package ingress

import (
	networkingV1 "k8s.io/api/networking/v1"
	networkingV1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("kube-ingress")
)

// client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type client struct {
	kubeClient      kubernetes.Interface
	informerV1      cache.SharedIndexInformer
	cacheV1         cache.Store
	informerV1beta1 cache.SharedIndexInformer
	cacheV1Beta1    cache.Store
	kubeController  k8s.Controller
	cfg             configurator.Configurator
	certProvider    certificate.Manager
	msgBroker       *messaging.Broker
}

// Monitor is the client interface for K8s Ingress resource
type Monitor interface {
	// GetIngressNetworkingV1beta1 returns the networking.k8s.io/v1beta1 ingress resources whose backends correspond to the service
	GetIngressNetworkingV1beta1(service.MeshService) ([]*networkingV1beta1.Ingress, error)

	// GetIngressNetworkingV1 returns the networking.k8s.io/v1 ingress resources whose backends correspond to the service
	GetIngressNetworkingV1(service.MeshService) ([]*networkingV1.Ingress, error)
}
