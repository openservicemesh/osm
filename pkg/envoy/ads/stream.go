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
	// TODO(draychev): this does not produce the correct IP address
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
			if !ok {
				glog.Errorf("[%s] Proxy %s closed GRPC", serverName, proxy.GetCommonName())
				return errGrpcClosed
			}
			if discoveryRequest.ErrorDetail != nil {
				glog.Errorf("[%s] Discovery request error from proxy %s: %s", serverName, proxy.GetCommonName(), discoveryRequest.ErrorDetail)
				return errEnvoyError
			}
			typeURL := envoy.TypeURI(discoveryRequest.TypeUrl)
			var lastNonce string
			if lastNonce, ok = proxy.LastNonce[typeURL]; !ok {
				lastNonce = ""
			}
			if len(proxy.LastNonce) > 0 && discoveryRequest.ResponseNonce == lastNonce {
				glog.V(level.Trace).Infof("[%s] Nothing changed since Nonce=%s", serverName, discoveryRequest.ResponseNonce)
				continue
			}
			if discoveryRequest.ResponseNonce != "" {
				glog.V(level.Trace).Infof("[%s] Received discovery request with Nonce=%s; matches=%t; proxy last Nonce=%s", serverName, discoveryRequest.ResponseNonce, discoveryRequest.ResponseNonce == lastNonce, lastNonce)
			}
			glog.Infof("[%s] Received discovery request <%s> from Envoy <%s> with Nonce=%s", serverName, discoveryRequest.TypeUrl, proxy.GetCommonName(), discoveryRequest.ResponseNonce)
			resp, err := s.newAggregatedDiscoveryResponse(proxy, &discoveryRequest)
			if err != nil {
				glog.Errorf("[%s] Error composing a DiscoveryResponse: %+v", serverName, err)
				continue
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
	for _, uri := range envoy.XDSResponseOrder {
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
	typeURL := envoy.TypeURI(request.TypeUrl)
	handler, ok := s.xdsHandlers[typeURL]
	if !ok {
		glog.Errorf("[%s] Responder for TypeUrl %s is not implemented", serverName, request.TypeUrl)
		return nil, errUnknownTypeURL
	}

	response, err := handler(proxy)
	if err != nil {
		glog.Errorf("[%s] Error creating %s response: %s", serverName, request.TypeUrl, err)
		return nil, errCreatingResponse
	}

	proxy.LastVersion = proxy.LastVersion + 1
	proxy.LastNonce[typeURL] = fmt.Sprintf("%d", time.Now().UnixNano())
	response.Nonce = proxy.LastNonce[typeURL]
	response.VersionInfo = fmt.Sprintf("v%d", proxy.LastVersion)
	proxy.LastUpdated = time.Now()

	glog.V(level.Trace).Infof("[%s] Constructed %s response.", serverName, request.TypeUrl)

	return response, nil
}
