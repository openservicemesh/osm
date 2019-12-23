package eds

import (
	"context"
	"fmt"
	"time"

	envoy "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	protobufTypes "github.com/gogo/protobuf/types"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type edsStreamHandler struct {
	lastVersion int
	lastNonce   string

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

		if request.TypeUrl != clusterLoadAssignment {
			glog.Errorf("unknown TypeUrl %s", request.TypeUrl)
			return errUnknownTypeURL
		}

	Run:
		for {
			select {
			case <-ctx.Done():
				return nil
			case _ = <-e.announceChan.Out():
				glog.V(1).Infof("[EDS] Received a change announcement...")
				if err := e.updateEnvoyProxies(server, request.TypeUrl); err != nil {
					return errors.Wrap(err, "error sending")
				}
				break Run

			}
		}
	}
}

func (e *edsStreamHandler) updateEnvoyProxies(server envoy.EndpointDiscoveryService_StreamEndpointsServer, url string) error {
	glog.Info("[stream] Update all envoy proxies...")
	allServices, err := e.catalog.GetWeightedServices()
	if err != nil {
		glog.Error("Could not refresh weighted services: ", err)
		return err
	}

	for targetServiceName, weightedServices := range allServices {
		cla := newClusterLoadAssignment(targetServiceName, weightedServices)
		var protos []*protobufTypes.Any
		if proto, err := protobufTypes.MarshalAny(&cla); err != nil {
			glog.Errorf("Error marshalling ClusterLoadAssignment %+v: %s", cla, err)
		} else {
			protos = append(protos, proto)
		}
		resp := &envoy.DiscoveryResponse{
			Resources: protos,
			TypeUrl:   url,
		}

		e.lastVersion = e.lastVersion + 1
		e.lastNonce = string(time.Now().Nanosecond())
		resp.Nonce = e.lastNonce
		resp.VersionInfo = fmt.Sprintf("v%d", e.lastVersion)
		glog.Infof("[stream] Sending ClusterLoadAssignment to proxies: %+v", resp)
		err := server.Send(resp)
		if err != nil {
			glog.Error("[stream] Error sending ClusterLoadAssignment: ", err)
		}
	}

	return nil
}
