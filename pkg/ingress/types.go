package ingress

import (
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/service"
	extensionsV1beta "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

var (
	log = logger.New("kube-ingress")
)

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	informer            cache.SharedIndexInformer
	cache               cache.Store
	cacheSynced         chan interface{}
	announcements       chan interface{}
	namespaceController namespace.Controller
}

// Monitor is the client interface for K8s Ingress resource
type Monitor interface {
	// GetIngressResources returns the ingress resources whose backends correspond to the service
	GetIngressResources(service.MeshService) ([]*extensionsV1beta.Ingress, error)

	// GetAnnouncementsChannel returns the channel on which Ingress Monitor makes annoucements
	GetAnnouncementsChannel() <-chan interface{}
}
