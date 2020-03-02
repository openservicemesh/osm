package ads

import (
	"fmt"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
)

func (s *Server) sendAllResponses(proxy *envoy.Proxy, server *envoy_service_discovery_v2.AggregatedDiscoveryService_StreamAggregatedResourcesServer) {
	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for _, uri := range envoy.XDSResponseOrder {
		request := &envoy_api_v2.DiscoveryRequest{TypeUrl: string(uri)}
		discoveryResponse, err := s.newAggregatedDiscoveryResponse(proxy, request)
		if err != nil {
			glog.Error(err)
			continue
		}
		if err := (*server).Send(discoveryResponse); err != nil {
			glog.Errorf("[%s] Error sending DiscoveryResponse %s: %+v", packageName, uri, err)
		}
	}
}

func (s *Server) newAggregatedDiscoveryResponse(proxy *envoy.Proxy, request *envoy_api_v2.DiscoveryRequest) (*envoy_api_v2.DiscoveryResponse, error) {
	typeURL := envoy.TypeURI(request.TypeUrl)
	handler, ok := s.xdsHandlers[typeURL]
	if !ok {
		glog.Errorf("[%s] Responder for TypeUrl %s is not implemented", packageName, request.TypeUrl)
		return nil, errUnknownTypeURL
	}

	response, err := handler(s.ctx, s.catalog, s.meshSpec, proxy)
	if err != nil {
		glog.Errorf("[%s] Responder for TypeUrl %s is not implemented", packageName, request.TypeUrl)
		return nil, errCreatingResponse
	}

	proxy.LastVersion[typeURL] = proxy.LastVersion[typeURL] + 1
	proxy.LastNonce[typeURL] = fmt.Sprintf("%d", time.Now().UnixNano())
	response.Nonce = proxy.LastNonce[typeURL]
	response.VersionInfo = fmt.Sprintf("v%+v", proxy.LastVersion)

	glog.V(level.Trace).Infof("[%s] Constructed %s response.", packageName, request.TypeUrl)

	return response, nil
}
