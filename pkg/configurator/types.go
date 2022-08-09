// Package configurator implements the Configurator interface that provides APIs to retrieve OSM control plane configurations.
package configurator

import (
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/k8s/informers"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("configurator")
)

// Client is the type used to represent the Kubernetes Client for the config.openservicemesh.io API group
type Client struct {
	osmNamespace   string
	informers      *informers.InformerCollection
	meshConfigName string
}

// Configurator is an interface for interacting with the Mesh Config, and the namespace it resides in.
type Configurator interface {
	GetMeshConfig() v1alpha2.MeshConfig
	GetOSMNamespace() string
}
