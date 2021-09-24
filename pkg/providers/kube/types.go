package kube

import (
	"github.com/openservicemesh/osm/pkg/config"
	"github.com/openservicemesh/osm/pkg/configurator"
	k8sInterfaces "github.com/openservicemesh/osm/pkg/k8s/interfaces"
	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// providerName is the name of the Kubernetes client that implements service.Provider and endpoint.Provider interfaces
	providerName = "kubernetes"
)

var (
	log = logger.New("kube-provider")
)

// client is the type used to represent the k8s client for endpoints and service provider
type client struct {
	kubeController   k8sInterfaces.Controller
	configClient     config.Controller
	meshConfigurator configurator.Configurator
}
