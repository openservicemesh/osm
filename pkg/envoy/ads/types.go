// Package ads implements Envoy's Aggregated Discovery Service (ADS).
package ads

import (
	"sync"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/workerpool"
)

var (
	log = logger.New("envoy/ads")
)

// Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	catalog       catalog.MeshCataloger
	proxyRegistry *registry.ProxyRegistry
	xdsHandlers   map[envoy.TypeURI]func(catalog.MeshCataloger, *envoy.Proxy, *certificate.Manager, *registry.ProxyRegistry) ([]types.Resource, error)
	// xdsLog is a map of key proxy.GetName(), which is of the form <identity>:<uuid> to map xds TypeURI to a slice of timestamps
	xdsLog         map[string]map[envoy.TypeURI][]time.Time
	xdsMapLogMutex sync.Mutex
	osmNamespace   string
	certManager    *certificate.Manager
	ready          bool
	workqueues     *workerpool.WorkerPool
	kubecontroller k8s.Controller

	// ---
	// SnapshotCache implementation structrues below
	// Used to maintain a unique ID per stream. Must be accessed with the atomic package.
	snapshotCache cachev3.SnapshotCache
	// When snapshot cache is enabled, we (currently) don't keep track of proxy information, however different
	// config versions have to be provided to the cache as we keep adding snapshots. The following map
	// tracks at which version we are at given a proxy UUID
	configVerMutex sync.Mutex
	configVersion  map[string]uint64

	msgBroker *messaging.Broker
}
