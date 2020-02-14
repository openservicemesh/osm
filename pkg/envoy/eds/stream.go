package eds

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	serverName = "EDS"
)

// StreamEndpoints implements v2.EndpointDiscoveryService and handles streaming of Endpoint changes to the Envoy proxies connected.
func (s *Server) StreamEndpoints(server v2.EndpointDiscoveryService_StreamEndpointsServer) error {
	glog.Infof("[%s] Starting StreamEndpoints", serverName)

	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, serverName)
	if err != nil {
		return errors.Wrapf(err, "[%s] Could not start StreamEndpoints", serverName)
	}

	glog.Infof("[%s][stream] Client connected: Subject CN=%s", serverName, cn)

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(server.Context())
	proxy := s.catalog.RegisterProxy(cn, ip)
	defer s.catalog.UnregisterProxy(proxy.GetID())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		request, err := server.Recv()
		if err != nil {
			return errors.Wrap(err, "recv")
		}

		if request.TypeUrl != envoy.TypeEDS {
			glog.Errorf("[%s][stream] Unknown TypeUrl: %s", serverName, request.TypeUrl)
			return errUnknownTypeURL
		}

	Run:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-proxy.GetAnnouncementsChannel():
				// NOTE(draychev): This is deliberately only focused on providing MVP tools to run a TrafficSplit demo.
				glog.V(level.Info).Infof("[%s][stream] Received a change message! Updating all Envoy proxies.", serverName)
				// TODO(draychev): flesh out the ClientIdentity
				weightedServices, err := s.catalog.ListEndpoints("TBD")
				if err != nil {
					glog.Errorf("[%s][stream] Failed listing endpoints: %+v", serverName, err)
					return err
				}
				glog.Infof("[%s][stream] WeightedServices: %+v", serverName, weightedServices)
				resp, err := s.NewEndpointDiscoveryResponse(weightedServices)
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
