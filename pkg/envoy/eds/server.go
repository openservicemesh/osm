package eds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	serverName = "EDS"
)

// Server implements the Envoy xDS Endpoint Discovery Services
type Server struct {
	ctx           context.Context // root context
	catalog       catalog.MeshCataloger
	meshSpec      smi.MeshSpec
	announcements chan interface{}
}

// FetchEndpoints implements envoy.EndpointDiscoveryServiceServer
func (e *Server) FetchEndpoints(context.Context, *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	panic("NotImplemented")
}

// DeltaEndpoints implements envoy.EndpointDiscoveryServiceServer
func (e *Server) DeltaEndpoints(xds.EndpointDiscoveryService_DeltaEndpointsServer) error {
	panic("NotImplemented")
}

// NewEDSServer creates a new EDS server
func NewEDSServer(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, announcements chan interface{}) *Server {
	glog.Info("[EDS] Create NewEDSServer")
	return &Server{
		ctx:           ctx,
		catalog:       catalog,
		meshSpec:      meshSpec,
		announcements: announcements,
	}
}
