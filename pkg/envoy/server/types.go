// Package ads implements Envoy's Aggregated Discovery Service (ADS).
package ads

import (
	"context"
	"sync"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("envoy/server")
)

// streamCallback is the interface used to notify the rest of the system that a proxy has connected, with the unique
// connection id.
type streamCallback interface {
	ProxyConnected(ctx context.Context, connectionID int64) error
	ProxyDisconnected(connectionID int64)
}

// Server implements the Envoy xDS Aggregate Discovery Services
type Server struct {
	callbacks streamCallback

	// ---
	// SnapshotCache implementation structrues below
	// Used to maintain a unique ID per stream. Must be accessed with the atomic package.
	snapshotCache cachev3.SnapshotCache
	// When snapshot cache is enabled, we (currently) don't keep track of proxy information, however different
	// config versions have to be provided to the cache as we keep adding snapshots. The following map
	// tracks at which version we are at given a proxy UUID
	configVerMutex sync.Mutex
	configVersion  map[string]uint64
}
