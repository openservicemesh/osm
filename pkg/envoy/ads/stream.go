package ads

import (
	"context"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/utils"
)

// StreamAggregatedResources handles streaming of the clusters to the connected Envoy proxies
func (s *Server) StreamAggregatedResources(server discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, packageName)
	if err != nil {
		return errors.Wrap(err, "[%s] Could not start stream")
	}

	// TODO(draychev): check for envoy.ErrTooManyConnections

	glog.Infof("[%s] Client connected: Subject CN=%s", packageName, cn)

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(server.Context())
	proxy := envoy.NewProxy(cn, ip)
	s.catalog.RegisterProxy(proxy)
	defer s.catalog.UnregisterProxy(proxy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requests := make(chan v2.DiscoveryRequest)
	go receive(requests, &server)

	for {
		select {
		case <-ctx.Done():
			return nil

		case discoveryRequest, ok := <-requests:
			glog.Infof("[%s][incoming] Discovery Request %s from Envoy %s", packageName, discoveryRequest.TypeUrl, proxy.GetCommonName())
			if !ok {
				glog.Errorf("[%s] Proxy %s closed GRPC", packageName, proxy.GetCommonName())
				return errGrpcClosed
			}
			if discoveryRequest.ErrorDetail != nil {
				glog.Errorf("[%s] Discovery request error from proxy %s: %s", packageName, proxy.GetCommonName(), discoveryRequest.ErrorDetail)
				return errEnvoyError
			}
			if len(proxy.LastNonce) > 0 && discoveryRequest.ResponseNonce == proxy.LastNonce {
				glog.Warningf("[%s] Nothing changed", packageName)
				continue
			}
			resp, err := s.newAggregatedDiscoveryResponse(proxy, &discoveryRequest)
			if err != nil {
				glog.Errorf("[%s] Failed composing a DiscoveryResponse: %+v", packageName, err)
				return err
			}
			if err := server.Send(resp); err != nil {
				glog.Errorf("[%s] Error sending DiscoveryResponse: %+v", packageName, err)
			}

		case <-proxy.GetAnnouncementsChannel():
			glog.V(level.Info).Infof("[%s] Change detected - update all Envoys.", packageName)
			s.sendAllResponses(proxy, &server)
		}
	}
}
