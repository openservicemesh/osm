package rds

import (
	"context"
	"github.com/deislabs/smc/pkg/envoy"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/log"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	serverName = "RDS"
)

// StreamRoutes handles streaming of route changes to the Envoy proxies connected
func (e *Server) StreamRoutes(server xds.RouteDiscoveryService_StreamRoutesServer) error {
	glog.Infof("[%s] Starting StreamRoutes", serverName)

	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, serverName)
	if err != nil {
		return errors.Wrapf(err, "[%s] Could not start StreamRoutes", serverName)
	}

	glog.Infof("[%s][stream] Client connected: Subject CN=%s", serverName, cn)

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(server.Context())
	proxy := e.catalog.RegisterProxy(cn, ip)
	defer e.catalog.UnregisterProxy(proxy.GetID())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		request, err := server.Recv()
		if err != nil {
			return errors.Wrap(err, "recv")
		}

		if request.TypeUrl != envoy.TypeRDS {
			glog.Errorf("[%s][stream] Unknown TypeUrl: %s", serverName, request.TypeUrl)
			return errUnknownTypeURL
		}

	Run:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-proxy.GetAnnouncementsChannel():
				// NOTE: This is deliberately only focused on providing MVP tools to run a TrafficRoute demo.
				glog.V(log.LvlInfo).Infof("[%s][stream] Received a change message! Updating all Envoy proxies.", serverName)
				// TODO: flesh out the ClientIdentity for this similar to eds.go
				trafficPolicies, err := e.catalog.ListTrafficRoutes("TBD")
				if err != nil {
					glog.Errorf("[%s][stream] Failed listing routes: %+v", serverName, err)
					return err
				}
				glog.Infof("[%s][stream] trafficPolicies: %+v", serverName, trafficPolicies)
				resp, err := e.newRouteDiscoveryResponse(trafficPolicies)
				if err != nil {
					glog.Errorf("[%s][stream] Failed composing a DiscoveryResponse: %+v", serverName, err)
					return err
				}
				if err := server.Send(resp); err != nil {
					glog.Errorf("[%s][stream] Error sending DiscoveryResponse: %+v", serverName, err)
				}
				break Run
			}
		}
	}
}
