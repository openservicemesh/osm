package eds

import (
	"context"

	"github.com/eapache/channels"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/mesh"
)

const (
	serverName = "EDS"
)

// EDS implements the Envoy xDS Endpoint Discovery Services
type EDS struct {
	ctx          context.Context // root context
	catalog      catalog.ServiceCataloger
	meshTopology mesh.Topology
	announcements *channels.RingChannel
}

// FetchEndpoints implements envoy.EndpointDiscoveryServiceServer
func (e *EDS) FetchEndpoints(context.Context, *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	panic("NotImplemented")
}

// DeltaEndpoints implements envoy.EndpointDiscoveryServiceServer
func (e *EDS) DeltaEndpoints(xds.EndpointDiscoveryService_DeltaEndpointsServer) error {
	panic("NotImplemented")
}

// NewEDSServer creates a new EDS server
func NewEDSServer(ctx context.Context, catalog catalog.ServiceCataloger, meshTopology mesh.Topology, announcements *channels.RingChannel) *EDS {
	glog.Info("[EDS] Create NewEDSServer")
	return &EDS{
		ctx:          ctx,
		catalog:      catalog,
		meshTopology: meshTopology,
		announcements: announcements,
	}
}
