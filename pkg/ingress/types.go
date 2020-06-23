package ingress

import (
	"github.com/open-service-mesh/osm/pkg/configurator"
	extensionsV1beta "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"

	"github.com/open-service-mesh/osm/pkg/logger"
	"github.com/open-service-mesh/osm/pkg/service"
)

var (
	log = logger.New("kube-ingress")
)

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	informer      cache.SharedIndexInformer
	cache         cache.Store
	cacheSynced   chan interface{}
	announcements chan interface{}
	configerator  configurator.Configurator
}

// Monitor is the client interface for K8s Ingress resource
type Monitor interface {
	// GetIngressResources returns the ingress resources whose backends correspond to the service
	GetIngressResources(service.NamespacedService) ([]*extensionsV1beta.Ingress, error)

	// GetAnnouncementsChannel returns the channel on which Ingress Monitor makes annoucements
	GetAnnouncementsChannel() <-chan interface{}
}
