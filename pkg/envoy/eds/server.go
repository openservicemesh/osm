package eds

import (
	"context"
	"time"

	"github.com/eapache/channels"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/mesh"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	serverName = "EDS"
)

// EDS implements the Envoy xDS Endpoint Discovery Services
type EDS struct {
	ctx          context.Context // root context
	catalog      catalog.ServiceCataloger
	meshTopology mesh.Topology
	announceChan *channels.RingChannel
}

// FetchEndpoints is required by the EDS interface
func (e *EDS) FetchEndpoints(context.Context, *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	panic("NotImplemented")
}

// DeltaEndpoints is required by the EDS interface
func (e *EDS) DeltaEndpoints(xds.EndpointDiscoveryService_DeltaEndpointsServer) error {
	panic("NotImplemented")
}

// NewEDSServer creates a new EDS server
func NewEDSServer(ctx context.Context, catalog catalog.ServiceCataloger, meshTopology mesh.Topology, announceChan *channels.RingChannel) *EDS {
	glog.Info("[EDS] Create NewEDSServer")
	return &EDS{
		ctx:          ctx,
		catalog:      catalog,
		meshTopology: meshTopology,
		announceChan: announceChan,
	}
}

// StreamEndpoints handles streaming of Endpoint changes to the Envoy proxies connected
func (e *EDS) StreamEndpoints(server xds.EndpointDiscoveryService_StreamEndpointsServer) error {
	glog.Infof("[%s] Starting StreamEndpoints", serverName)

	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, serverName)
	if err != nil {
		return errors.Wrapf(err, "[%s] Could not start stream", serverName)
	}

	// TODO(draychev): Use the Subject Common Name to identify the Envoy proxy and determine what service it belongs to.
	glog.Infof("[%s][stream] Client connected: Subject CN=%+v", serverName, cn)

	ctx, cancel := context.WithCancel(context.Background())
	handler := &edsStreamHandler{
		ctx:    ctx,
		cancel: cancel,
		EDS:    e,
	}

	// Periodic Updates -- useful for debugging
	go func() {
		counter := 0
		for {
			glog.V(7).Infof("------------------------- Periodic Update %d -------------------------", counter)
			counter++
			e.announceChan.In() <- nil
			time.Sleep(5 * time.Second)
		}
	}()

	if err := handler.run(e.ctx, server); err != nil {
		glog.Infof("[%s] Handler failed with: %s", serverName, err)
		return err
	}
	return nil
}
