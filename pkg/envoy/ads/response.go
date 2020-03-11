package ads

import (
	"strconv"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"

	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/log/level"
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

	glog.V(level.Trace).Infof("[%s] Invoking handler for %s with request: %+v", packageName, typeURL, request)
	response, err := handler(s.ctx, s.catalog, s.meshSpec, proxy, request)
	if err != nil {
		glog.Errorf("[%s] Responder for TypeUrl %s is not implemented", packageName, request.TypeUrl)
		return nil, errCreatingResponse
	}

	response.Nonce = proxy.SetNewNonce(typeURL)
	response.VersionInfo = strconv.FormatUint(proxy.IncrementLastSentVersion(typeURL), 10)

	glog.V(level.Trace).Infof("[%s] Constructed %s response: %+v", packageName, request.TypeUrl, response)

	return response, nil
}
