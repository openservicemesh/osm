package ads

import (
	"context"
	"fmt"
	"time"

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
	cn, err := utils.ValidateClient(server.Context(), nil, serverName)
	if err != nil {
		return errors.Wrap(err, "[%s] Could not start stream")
	}

	// TODO(draychev): check for envoy.ErrTooManyConnections

	glog.Infof("[%s] Client connected: Subject CN=%s", serverName, cn)

	// Register the newly connected proxy w/ the catalog.
	ip := utils.GetIPFromContext(server.Context())
	proxy := s.catalog.RegisterProxy(cn, ip)
	defer s.catalog.UnregisterProxy(proxy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requests := make(chan v2.DiscoveryRequest)
	go receive(requests, &server)

	s.sendAllResponses(proxy, &server)

	for {
		select {
		case <-ctx.Done():
			return nil

		case discoveryRequest, ok := <-requests:
			glog.Infof("[%s][incoming] Discovery Request %s from Envoy %s", serverName, discoveryRequest.TypeUrl, proxy.GetCommonName())
			if !ok {
				glog.Errorf("[%s] Proxy %s closed GRPC", serverName, proxy.GetCommonName())
				return errGrpcClosed
			}
			if discoveryRequest.ErrorDetail != nil {
				glog.Errorf("[%s] Discovery request error from proxy %s: %s", serverName, proxy.GetCommonName(), discoveryRequest.ErrorDetail)
				return errEnvoyError
			}
			if len(s.lastNonce) > 0 && discoveryRequest.ResponseNonce == s.lastNonce {
				glog.Warningf("[%s] Nothing changed", serverName)
				continue
			}
			resp, err := s.newAggregatedDiscoveryResponse(proxy, &discoveryRequest)
			if err != nil {
				glog.Errorf("[%s] Failed composing a DiscoveryResponse: %+v", serverName, err)
				return err
			}
			if err := server.Send(resp); err != nil {
				glog.Errorf("[%s] Error sending DiscoveryResponse: %+v", serverName, err)
			}

		case <-proxy.GetAnnouncementsChannel():
			glog.V(level.Info).Infof("[%s] Change detected - update all Envoys.", serverName)
			s.sendAllResponses(proxy, &server)
		}
	}
}

func (s *Server) sendAllResponses(proxy *envoy.Proxy, server *discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) {
	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for _, uri := range []envoy.TypeURI{envoy.TypeCDS, envoy.TypeEDS, envoy.TypeLDS, envoy.TypeRDS, envoy.TypeSDS} {
		request := &v2.DiscoveryRequest{TypeUrl: string(uri)}
		discoveryResponse, err := s.newAggregatedDiscoveryResponse(proxy, request)
		if err != nil {
			glog.Error(err)
			continue
		}
		if err := (*server).Send(discoveryResponse); err != nil {
			glog.Errorf("[%s] Error sending DiscoveryResponse %s: %+v", serverName, uri, err)
		}

	}
}

func (s *Server) newAggregatedDiscoveryResponse(proxy *envoy.Proxy, request *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	glog.V(level.Info).Infof("[%s] Received discovery request: %s", serverName, request.TypeUrl)
	handler, ok := s.xdsHandlers[envoy.TypeURI(request.TypeUrl)]
	if !ok {
		glog.Errorf("Responder for TypeUrl %s is not implemented", request.TypeUrl)
		return nil, errUnknownTypeURL
	}

	response, err := handler(proxy)
	if err != nil {
		glog.Errorf("Error creating %s response: %s", request.TypeUrl, err)
		return nil, errCreatingResponse
	}

	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	response.Nonce = s.lastNonce
	response.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)
	glog.V(level.Trace).Infof("[%s] Constructed %s response: %+v", serverName, request.TypeUrl, response)
	return response, nil
}
