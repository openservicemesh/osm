package eds

import (
	"context"

	envoy "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy/cla"
)

type edsStreamHandler struct {
	// TODO(draychev):implement --> lastVersion int
	// TODO(draychev):implement --> lastNonce   string

	ctx    context.Context
	cancel context.CancelFunc

	*EDS
}

func (e *edsStreamHandler) run(ctx context.Context, server envoy.EndpointDiscoveryService_StreamEndpointsServer) error {
	defer e.cancel()
	for {
		request, err := server.Recv()
		if err != nil {
			return errors.Wrap(err, "recv")
		}

		if request.TypeUrl != cla.ClusterLoadAssignmentURI {
			glog.Errorf("[EDS][stream] Unknown TypeUrl: %s", request.TypeUrl)
			return errUnknownTypeURL
		}

	Run:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-e.announceChan.Out():
				// NOTE(draychev): This is deliberately only focused on providing MVP tools to run a TrafficSplit demo.
				glog.V(1).Infof("[EDS][stream] Received a change announcement! Updating all Envoy proxies.")
				// TODO(draychev): flesh out the ClientIdentity
				resp, _, err := e.catalog.ListEndpoints("TBD")
				if err != nil {
					glog.Error("[EDS][stream] Failed composing a DiscoveryResponse: ", err)
					return err
				}
				if err := server.Send(resp); err != nil {
					glog.Error("[EDS][stream] Error sending DiscoveryResponse: ", err)
				}
				break Run
			}
		}
	}
}
