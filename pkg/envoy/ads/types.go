package ads

import (
	"time"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("envoy/ads")
)

// Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	catalog      catalog.MeshCataloger
	xdsHandlers  map[envoy.TypeURI]func(catalog.MeshCataloger, *envoy.Proxy, *xds_discovery.DiscoveryRequest, configurator.Configurator, certificate.Manager) (*xds_discovery.DiscoveryResponse, error)
	xdsLog       map[certificate.CommonName]map[envoy.TypeURI][]time.Time
	enableDebug  bool
	osmNamespace string
	cfg          configurator.Configurator
	certManager  certificate.Manager
	ready        bool
}
