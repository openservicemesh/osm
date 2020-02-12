package lds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"github.com/deislabs/smc/pkg/catalog"
)

const (
	maxConnections = 10000
	typeUrl        = "type.googleapis.com/envoy.api.v2.Listener"
	serverName     = "LDS"
)

// NewLDSServer creates a new LDS server
func NewLDSServer(catalog catalog.MeshCataloger) *Server {
	return &Server{
		connectionNum: 0,
		catalog:       catalog,
		closing:       make(chan bool),
	}
}

// DeltaListeners implements xds.ListenerDiscoveryServiceServer
func (s *Server) DeltaListeners(xds.ListenerDiscoveryService_DeltaListenersServer) error {
	panic("NotImplemented")
}

// FetchListeners implements xds.ListenerDiscoveryServiceServer
func (s *Server) FetchListeners(ctx context.Context, discReq *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	panic("NotImplemented")
}
