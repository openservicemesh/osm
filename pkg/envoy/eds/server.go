package eds

import (
	"context"
	"time"

	"github.com/eapache/channels"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/mesh"
)

// EDS implements the Envoy xDS Endpoint Discovery Service
type EDS struct {
	ctx          context.Context // root context
	catalog      mesh.ServiceCataloger
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
func NewEDSServer(ctx context.Context, catalog mesh.ServiceCataloger, meshTopology mesh.Topology, announceChan *channels.RingChannel) *EDS {
	glog.Info("[EDS] Create NewEDSServer...")
	return &EDS{
		ctx:          ctx,
		catalog:      catalog,
		meshTopology: meshTopology,
		announceChan: announceChan,
	}
}

// StreamEndpoints handles streaming of Endpoint changes to the Envoy proxies connected
func (e *EDS) StreamEndpoints(server xds.EndpointDiscoveryService_StreamEndpointsServer) error {
	glog.Info("[EDS] Starting StreamEndpoints...")
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
		glog.Infof("error in handler %s", err)
		return err
	}
	return nil
}
