package ads

import (
	"strconv"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"github.com/open-service-mesh/osm/pkg/envoy"
)

func (s *Server) sendAllResponses(proxy *envoy.Proxy, server *envoy_service_discovery_v2.AggregatedDiscoveryService_StreamAggregatedResourcesServer) {
	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for _, uri := range envoy.XDSResponseOrder {
		request := &envoy_api_v2.DiscoveryRequest{TypeUrl: string(uri)}
		discoveryResponse, err := s.newAggregatedDiscoveryResponse(proxy, request)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create ADS discovery response")
			continue
		}
		if err := (*server).Send(discoveryResponse); err != nil {
			log.Error().Err(err).Msgf("[%s] Error sending DiscoveryResponse %s", packageName, uri)
		}
	}
}

func (s *Server) newAggregatedDiscoveryResponse(proxy *envoy.Proxy, request *envoy_api_v2.DiscoveryRequest) (*envoy_api_v2.DiscoveryResponse, error) {
	typeURL := envoy.TypeURI(request.TypeUrl)
	handler, ok := s.xdsHandlers[typeURL]
	if !ok {
		log.Error().Msgf("[%s] Responder for TypeUrl %s is not implemented", packageName, request.TypeUrl)
		return nil, errUnknownTypeURL
	}

	log.Trace().Msgf("[%s] Invoking handler for %s with request: %+v", packageName, typeURL, request)
	response, err := handler(s.ctx, s.catalog, s.meshSpec, proxy, request)
	if err != nil {
		log.Error().Msgf("[%s] Responder for TypeUrl %s is not implemented", packageName, request.TypeUrl)
		return nil, errCreatingResponse
	}

	response.Nonce = proxy.SetNewNonce(typeURL)
	response.VersionInfo = strconv.FormatUint(proxy.IncrementLastSentVersion(typeURL), 10)

	log.Trace().Msgf("[%s] Constructed %s response: %+v", packageName, request.TypeUrl, response)

	return response, nil
}
