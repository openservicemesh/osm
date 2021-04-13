// Package ads implements Envoy's Aggregated Discovery Service (ADS).
package ads

import (
	"sync"
	"time"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/workerpool"
)

var (
	log = logger.New("envoy/ads")
)

// Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	catalog        catalog.MeshCataloger
	proxyRegistry  *registry.ProxyRegistry
	xdsHandlers    map[envoy.TypeURI]func(catalog.MeshCataloger, *envoy.Proxy, *xds_discovery.DiscoveryRequest, configurator.Configurator, certificate.Manager) ([]types.Resource, error)
	xdsLog         map[certificate.CommonName]map[envoy.TypeURI][]time.Time
	xdsMapLogMutex sync.Mutex
	osmNamespace   string
	cfg            configurator.Configurator
	certManager    certificate.Manager
	ready          bool
	workqueues     *workerpool.WorkerPool
}
