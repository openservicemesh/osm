package server

import (
	"context"
	"fmt"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/models"
)

const (
	// ServerType is the type identifier for the ADS server
	ServerType = "ADS"
	// xdsServerCertificateCommonName is the common name of the certificate for the ADS server
	xdsServerCertificateCommonName = "ads"
)

// NewADSServer creates a new Aggregated Discovery Service server
func NewADSServer() *Server {
	server := Server{
		snapshotCache: cachev3.NewSnapshotCache(false, cachev3.IDHash{}, &scLogger{
			log: logger.New("envoy/snapshot-cache"),
		}),
		configVersion: make(map[string]uint64),
	}

	return &server
}

// SetCallbacks is a method used to set the callbacks that notify the rest of the system that a proxy, with the given
// unique connection id, has either connected or disconnected.
func (s *Server) SetCallbacks(cb streamCallback) {
	s.callbacks = cb
}

// Start starts the ADS server
func (s *Server) Start(ctx context.Context, certManager *certificate.Manager, cancel context.CancelFunc, port int) error {
	grpcServer, lis, err := NewGrpc(ServerType, port, xdsServerCertificateCommonName, certManager)
	if err != nil {
		return fmt.Errorf("error starting ADS server: %w", err)
	}

	xds_discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer.GetServer(), serverv3.NewServer(ctx, s.snapshotCache, s))

	err = grpcServer.GrpcServe(ctx, cancel, lis, nil)
	if err != nil {
		return fmt.Errorf("error starting ADS server: %w", err)
	}

	return nil
}

// UpdateProxy stores a group of resources as a new Snapshot with a new version in the cache.
// It also runs a consistency check on the snapshot (will warn if there are missing resources referenced in
// the snapshot)
func (s *Server) UpdateProxy(ctx context.Context, proxy *models.Proxy, snapshotResources map[string][]types.Resource) error {
	uuid := proxy.UUID.String()

	s.configVerMutex.Lock()
	s.configVersion[uuid]++
	configVersion := s.configVersion[uuid]
	s.configVerMutex.Unlock()

	snapshot, err := cache.NewSnapshot(fmt.Sprintf("%d", configVersion), snapshotResources)
	if err != nil {
		return err
	}

	if err := snapshot.Consistent(); err != nil {
		return err
	}

	return s.snapshotCache.SetSnapshot(ctx, uuid, snapshot)
}
