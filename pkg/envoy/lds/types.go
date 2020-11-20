package lds

import (
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("envoy/lds")
)

// listenerBuilder is a type containing data to build the listener configurations
type listenerBuilder struct {
	svcAccount  service.K8sServiceAccount
	meshCatalog catalog.MeshCataloger
	cfg         configurator.Configurator
}
