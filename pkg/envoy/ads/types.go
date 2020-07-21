package ads

import (
	"context"
	"time"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/smi"
)

var (
	log = logger.New("envoy/ads")
)

// Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	ctx          context.Context
	catalog      catalog.MeshCataloger
	meshSpec     smi.MeshSpec
	xdsHandlers  map[envoy.TypeURI]func(context.Context, catalog.MeshCataloger, smi.MeshSpec, *envoy.Proxy, *xds_discovery.DiscoveryRequest, configurator.Configurator) (*xds_discovery.DiscoveryResponse, error)
	xdsLog       map[certificate.CommonName]map[envoy.TypeURI][]time.Time
	enableDebug  bool
	osmNamespace string
	cfg          configurator.Configurator
}
