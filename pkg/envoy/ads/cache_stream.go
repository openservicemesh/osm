package ads

import (
	"context"
	"fmt"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// Routine which fulfills listening to proxy broadcasts
func (s *Server) watchForUpdates(ctx context.Context) {
	// Register for proxy config updates broadcasted by the message broker
	proxyUpdatePubSub := s.msgBroker.GetProxyUpdatePubSub()
	proxyUpdateChan := proxyUpdatePubSub.Sub(messaging.ProxyUpdateTopic)
	defer s.msgBroker.Unsub(proxyUpdatePubSub, proxyUpdateChan)

	for {
		select {
		case <-proxyUpdateChan:
			s.allPodUpdater()
			// TODO(2683): listen to specific pod updates and cert rotations
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) allPodUpdater() {
	for _, proxy := range s.proxyRegistry.ListConnectedProxies() {
		s.update(proxy)
	}
}

func (s *Server) update(proxy *envoy.Proxy) {
	// Queue update for this proxy/pod
	job := proxyResponseJob{
		proxy:     proxy,
		adsStream: nil, // Since it goes in the cache, stream is not needed
		request:   nil, // No request is used, as we fill all verticals
		xdsServer: s,
		typeURIs:  envoy.XDSResponseOrder,
		done:      make(chan struct{}),
	}
	s.workqueues.AddJob(&job)
}

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
