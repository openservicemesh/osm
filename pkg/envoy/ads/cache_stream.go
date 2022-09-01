package ads

import (
	"context"
	"fmt"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/openservicemesh/osm/pkg/envoy"
)

// RecordFullSnapshot stores a group of resources as a new Snapshot with a new version in the cache.
// It also runs a consistency check on the snapshot (will warn if there are missing resources referenced in
// the snapshot)
func (s *Server) RecordFullSnapshot(proxy *envoy.Proxy, snapshotResources map[string][]types.Resource) error {
	snapshot, err := cache.NewSnapshot(
		fmt.Sprintf("%d", s.configVersion[proxy.UUID.String()]),
		snapshotResources,
	)
	if err != nil {
		return err
	}

	if err := snapshot.Consistent(); err != nil {
		log.Warn().Err(err).Str("proxy", proxy.String()).Msgf("Snapshot for proxy not consistent")
	}

	s.configVerMutex.Lock()
	defer s.configVerMutex.Unlock()
	s.configVersion[proxy.UUID.String()]++

	return s.snapshotCache.SetSnapshot(context.TODO(), proxy.UUID.String(), snapshot)
}
