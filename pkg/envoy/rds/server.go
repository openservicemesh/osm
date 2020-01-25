package rds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

// Server implements the Envoy xDS Route Discovery Services
type Server struct {
	ctx           context.Context // root context
	catalog       catalog.MeshCataloger
	meshSpec      smi.MeshSpec
	announcements chan interface{}
}

// FetchRoutes implements envoy.RouteDiscoveryServiceServer
func (r *Server) FetchRoutes(context.Context, *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	panic("NotImplemented")
}

// DeltaRoutes implements envoy.RouteDiscoveryServiceServer
func (r *Server) DeltaRoutes(xds.RouteDiscoveryService_DeltaRoutesServer) error {
	panic("NotImplemented")
}

// NewRDSServer creates a new RDS server
func NewRDSServer(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, announcements chan interface{}) *Server {
	glog.Info("[RDS] Create NewRDSServer...")
	return &Server{
		ctx:           ctx,
		catalog:       catalog,
		meshSpec:      meshSpec,
		announcements: announcements,
	}
}
