package rds

import (
	"context"

	"github.com/eapache/channels"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/mesh"
)

// RDS implements the Envoy xDS Route Discovery Services
type RDS struct {
	ctx          context.Context // root context
	catalog      catalog.ServiceCataloger
	meshTopology mesh.Topology
	announceChan *channels.RingChannel
}

// FetchRoutes implements envoy.RouteDiscoveryServiceServer
func (r *RDS) FetchRoutes(context.Context, *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	panic("NotImplemented")
}

// DeltaRoutes implements envoy.RouteDiscoveryServiceServer
func (r *RDS) DeltaRoutes(xds.RouteDiscoveryService_DeltaRoutesServer) error {
	panic("NotImplemented")
}

// NewRDSServer creates a new RDS server
func NewRDSServer(ctx context.Context, catalog catalog.ServiceCataloger, meshTopology mesh.Topology, announceChan *channels.RingChannel) *RDS {
	glog.Info("[RDS] Create NewRDSServer...")
	return &RDS{
		ctx:          ctx,
		catalog:      catalog,
		meshTopology: meshTopology,
		announceChan: announceChan,
	}
}
