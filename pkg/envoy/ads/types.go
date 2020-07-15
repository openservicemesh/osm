package ads

import (
	"context"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/logger"
	"github.com/open-service-mesh/osm/pkg/smi"
)

var (
	log = logger.New("envoy/ads")
)

// Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	ctx          context.Context
	catalog      catalog.MeshCataloger
	meshSpec     smi.MeshSpec
	xdsHandlers  map[envoy.TypeURI]func(context.Context, catalog.MeshCataloger, smi.MeshSpec, *envoy.Proxy, *xds.DiscoveryRequest, configurator.Configurator) (*xds.DiscoveryResponse, error)
	xdsLog       map[certificate.CommonName]map[envoy.TypeURI][]time.Time
	enableDebug  bool
	osmNamespace string
	cfg          configurator.Configurator
}
