package kube

import (
	"github.com/openservicemesh/osm/pkg/config"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("kube-provider")
)

// Client is a struct for all components necessary to connect to and maintain state of a Kubernetes cluster.
type Client struct {
	providerIdent  string
	kubeController k8s.Controller
	configClient   config.Controller
	configurator   configurator.Configurator
}
