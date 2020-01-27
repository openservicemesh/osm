package eds

import (
	"context"
	"net"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/cla"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	serverName = "EDS"
)

// StreamEndpoints implements v2.EndpointDiscoveryService and handles streaming of Endpoint changes to the Envoy proxies connected.
func (e *Server) StreamEndpoints(server v2.EndpointDiscoveryService_StreamEndpointsServer) error {
	glog.Infof("[%s] Starting StreamEndpoints", serverName)

	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, serverName)
	if err != nil {
		return errors.Wrapf(err, "[%s] Could not start StreamEndpoints", serverName)
	}

	// TODO(draychev): Use the Subject Common Name to identify the Envoy proxy and determine what service it belongs to.
	glog.Infof("[%s][stream] Client connected: Subject CN=%+v", serverName, cn)

	// Register the newly connected Envoy proxy.
	connectedProxyIPAddress := net.IP("TBD")
	connectedProxyCertCommonName := certificate.CommonName("TBD")
	proxy := envoy.NewProxy(connectedProxyCertCommonName, connectedProxyIPAddress)
	e.catalog.RegisterProxy(proxy)

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(server.Context())
	e.catalog.RegisterProxy(envoy.NewProxy(cn, ip))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Periodic Updates -- useful for debugging
	go func() {
		counter := 0
		for {
			glog.V(7).Infof("------------------------- %s Periodic Update %d -------------------------", serverName, counter)
			counter++
			e.announcements <- struct{}{}
			time.Sleep(5 * time.Second)
		}
	}()

	for {
		request, err := server.Recv()
		if err != nil {
			return errors.Wrap(err, "recv")
		}

		if request.TypeUrl != cla.ClusterLoadAssignmentURI {
			glog.Errorf("[%s][stream] Unknown TypeUrl: %s", serverName, request.TypeUrl)
			return errUnknownTypeURL
		}

	Run:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-e.announcements:
				// NOTE(draychev): This is deliberately only focused on providing MVP tools to run a TrafficSplit demo.
				glog.V(1).Infof("[%s][stream] Received a change announcement! Updating all Envoy proxies.", serverName)
				// TODO(draychev): flesh out the ClientIdentity
				weightedServices, err := e.catalog.ListEndpoints("TBD")
				if err != nil {
					glog.Errorf("[%s][stream] Failed listing endpoints: %+v", serverName, err)
					return err
				}
				glog.Infof("[%s][stream] WeightedServices: %+v", serverName, weightedServices)
				resp, err := e.newDiscoveryResponse(weightedServices)
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
