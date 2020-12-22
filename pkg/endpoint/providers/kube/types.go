package kube

import (
	"k8s.io/client-go/kubernetes"

	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("kube-provider")
)

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	providerIdent  string
	kubeClient     kubernetes.Interface
	kubeController k8s.Controller
}
