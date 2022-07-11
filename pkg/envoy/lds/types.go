// Package lds implements Envoy's Listener Discovery Service (LDS).
package lds

import (
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("envoy/lds")
)

// listenerBuilder is a type containing data to build the listener configurations
type listenerBuilder struct {
	serviceIdentity identity.ServiceIdentity
	meshCatalog     catalog.MeshCataloger
	cfg             configurator.Configurator
	statsHeaders    map[string]string
	trustDomain     string
}
