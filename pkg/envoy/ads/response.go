package ads

import (
	"fmt"
	"strconv"
	"time"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

func (s *Server) sendAllResponses(proxy *envoy.Proxy, server *discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, cfg configurator.Configurator) {
	log.Trace().Msgf("A change announcement triggered *DS update for proxy with CN=%s", proxy.GetCommonName())
	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for idx, typeURI := range envoy.XDSResponseOrder {
		prefix := fmt.Sprintf("[*DS %d/%d]", idx+1, len(envoy.XDSResponseOrder))
		log.Trace().Msgf("%s Creating %s response for proxy with CN=%s", prefix, typeURI, proxy.GetCommonName())

		// For SDS we need to add ResourceNames
		var request *discovery.DiscoveryRequest
		if typeURI == envoy.TypeSDS {
			request = makeRequestForAllSecrets(proxy, s.catalog)
			if request == nil {
				continue
			}
		} else {
			request = &discovery.DiscoveryRequest{TypeUrl: string(typeURI)}
		}

		discoveryResponse, err := s.newAggregatedDiscoveryResponse(proxy, request, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("%s Failed to create %s discovery response for proxy with CN=%s", prefix, typeURI, proxy.GetCommonName())
			continue
		}
		if err := (*server).Send(discoveryResponse); err != nil {
			log.Error().Err(err).Msgf("%s Error sending %s to proxy with CN=%s", prefix, typeURI, proxy.GetCommonName())
		}
	}
}

// makeRequestForAllSecrets constructs an SDS request AS IF an Envoy proxy sent it.
// This request will result in the rest of the system creating an SDS response with the certificates
// required by this proxy. The proxy itself did not ask for these. We know it needs them - so we send them.
func makeRequestForAllSecrets(proxy *envoy.Proxy, catalog catalog.MeshCataloger) *discovery.DiscoveryRequest {
	serviceForProxy, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service for Envoy with CN=%q", proxy.GetCommonName())
		return nil
	}

	return &discovery.DiscoveryRequest{
		ResourceNames: []string{
			envoy.SDSCert{
				Service:  *serviceForProxy,
				CertType: envoy.ServiceCertType,
			}.String(),
			envoy.SDSCert{
				Service:  *serviceForProxy,
				CertType: envoy.RootCertTypeForMTLSOutbound,
			}.String(),
			envoy.SDSCert{
				Service:  *serviceForProxy,
				CertType: envoy.RootCertTypeForMTLSInbound,
			}.String(),
			envoy.SDSCert{
				Service:  *serviceForProxy,
				CertType: envoy.RootCertTypeForHTTPS,
			}.String(),
		},
		TypeUrl: string(envoy.TypeSDS),
	}
}

func (s *Server) newAggregatedDiscoveryResponse(proxy *envoy.Proxy, request *discovery.DiscoveryRequest, cfg configurator.Configurator) (*discovery.DiscoveryResponse, error) {
	typeURL := envoy.TypeURI(request.TypeUrl)
	handler, ok := s.xdsHandlers[typeURL]
	if !ok {
		log.Error().Msgf("Responder for TypeUrl %s is not implemented", request.TypeUrl)
		return nil, errUnknownTypeURL
	}

	if s.enableDebug {
		if _, ok := s.xdsLog[proxy.GetCommonName()]; !ok {
			s.xdsLog[proxy.GetCommonName()] = make(map[envoy.TypeURI][]time.Time)
		}
		s.xdsLog[proxy.GetCommonName()][typeURL] = append(s.xdsLog[proxy.GetCommonName()][typeURL], time.Now())
	}

	log.Trace().Msgf("Invoking handler for %s with request: %+v", typeURL, request)
	response, err := handler(s.ctx, s.catalog, s.meshSpec, proxy, request, cfg)
	if err != nil {
		log.Error().Msgf("Responder for TypeUrl %s is not implemented", request.TypeUrl)
		return nil, errCreatingResponse
	}

	response.Nonce = proxy.SetNewNonce(typeURL)
	response.VersionInfo = strconv.FormatUint(proxy.IncrementLastSentVersion(typeURL), 10)

	if envoy.TypeURI(request.TypeUrl) == envoy.TypeSDS {
		log.Trace().Msgf("Constructed %s response: VersionInfo=%s", response.TypeUrl, response.VersionInfo)
	} else {
		log.Trace().Msgf("Constructed %s response: VersionInfo=%s; %+v", response.TypeUrl, response.VersionInfo, response)
	}

	return response, nil
}
