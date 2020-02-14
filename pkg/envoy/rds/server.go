package rds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

// NewRDSServer creates a new RDS server
func NewRDSServer(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec) *Server {
	glog.Info("[RDS] Create NewRDSServer")
	return &Server{
		ctx:      ctx,
		catalog:  catalog,
		meshSpec: meshSpec,
	}
}

// FetchRoutes implements envoy.RouteDiscoveryServiceServer
func (s *Server) FetchRoutes(context.Context, *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	panic("NotImplemented")
}

// DeltaRoutes implements envoy.RouteDiscoveryServiceServer
func (s *Server) DeltaRoutes(xds.RouteDiscoveryService_DeltaRoutesServer) error {
	panic("NotImplemented")
}
