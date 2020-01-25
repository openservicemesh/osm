package rds

import (
	"context"
	"time"

	envoy "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy/rc"
)

type rdsStreamHandler struct {
	ctx    context.Context
	cancel context.CancelFunc

	*Server
}

// StreamRoutes handles streaming of route changes to the Envoy proxies connected
func (e *Server) StreamRoutes(server xds.RouteDiscoveryService_StreamRoutesServer) error {
	glog.Info("[RDS] Starting StreamRoutes...")
	ctx, cancel := context.WithCancel(context.Background())
	handler := &rdsStreamHandler{
		ctx:    ctx,
		cancel: cancel,
		Server:    e,
	}

	// Periodic Updates -- useful for debugging
	go func() {
		counter := 0
		for {
			glog.V(7).Infof("------------------------- Periodic Update %d -------------------------", counter)
			counter++
			e.announcements <- struct{}{}
			time.Sleep(5 * time.Second)
		}
	}()

	if err := handler.run(e.ctx, server); err != nil {
		glog.Infof("error in handler %s", err)
		return err
	}
	return nil
}

func (r *rdsStreamHandler) run(ctx context.Context, server envoy.RouteDiscoveryService_StreamRoutesServer) error {
	defer r.cancel()
	for {
		request, err := server.Recv()
		if err != nil {
			return errors.Wrap(err, "recv")
		}

		if request.TypeUrl != rc.RouteConfigurationURI {
			glog.Errorf("[RDS][stream] Unknown TypeUrl: %s", request.TypeUrl)
			return errUnknownTypeURL
		}

	Run:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-r.announcements:
				// NOTE: This is deliberately only focused on providing MVP tools to run a TrafficRoute demo.
				glog.V(1).Infof("[RDS][stream] Received a change announcement! Updating all Envoy proxies.")
				// TODO: flesh out the ClientIdentity for this similar to eds.go
				resp, _, err := r.catalog.ListTrafficRoutes("TBD")
				if err != nil {
					glog.Error("[RDS][stream] Failed composing a DiscoveryResponse: ", err)
					return err
				}
				if err := server.Send(resp); err != nil {
					glog.Error("[RDS][stream] Error sending DiscoveryResponse: ", err)
				}
				break Run
			}
		}
	}
}
