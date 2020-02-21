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
	for _, uri := range []envoy.TypeURI{envoy.TypeCDS, envoy.TypeEDS, envoy.TypeLDS, envoy.TypeRDS, envoy.TypeSDS} {
		request := &envoy_api_v2.DiscoveryRequest{TypeUrl: string(uri)}
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

func (s *Server) newAggregatedDiscoveryResponse(proxy *envoy.Proxy, request *envoy_api_v2.DiscoveryRequest) (*envoy_api_v2.DiscoveryResponse, error) {
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

	proxy.LastVersion = proxy.LastVersion + 1
	proxy.LastNonce = fmt.Sprintf("%d", time.Now().UnixNano())
	response.Nonce = proxy.LastNonce
	response.VersionInfo = fmt.Sprintf("v%d", proxy.LastVersion)

	glog.V(level.Trace).Infof("[%s] Constructed %s response: %+v", serverName, request.TypeUrl, response)

	return response, nil
}
