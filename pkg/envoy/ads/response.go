package ads

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
)

func (s *Server) sendAllResponses(proxy *envoy.Proxy, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, cfg configurator.Configurator) {
	log.Trace().Msgf("A change announcement triggered *DS update for proxy with CN=%s", proxy.GetCommonName())
	fullUpdateStartTime := time.Now()

	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for idx, typeURI := range envoy.XDSResponseOrder {
		prefix := fmt.Sprintf("[*DS %d/%d]", idx+1, len(envoy.XDSResponseOrder))
		log.Trace().Msgf("%s Creating %s response for proxy with CN=%s", prefix, typeURI, proxy.GetCommonName())
		updateStartTime := time.Now()

		// For SDS we need to add ResourceNames
		var request *xds_discovery.DiscoveryRequest
		if typeURI == envoy.TypeSDS {
			request = makeRequestForAllSecrets(proxy, s.catalog)
			if request == nil {
				continue
			}
		} else {
			request = &xds_discovery.DiscoveryRequest{TypeUrl: string(typeURI)}
		}

		discoveryResponse, err := s.newAggregatedDiscoveryResponse(proxy, request, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("%s Failed to create %s discovery response for proxy with CN=%s", prefix, typeURI, proxy.GetCommonName())
		} else {
			if err := (*server).Send(discoveryResponse); err != nil {
				log.Error().Err(err).Msgf("%s Error sending %s to proxy with CN=%s", prefix, typeURI, proxy.GetCommonName())
			}
		}
		log.Debug().Msgf("%s (%s) proxy %s took %s",
			prefix,
			typeURI.String()[strings.LastIndex(typeURI.String(), ".")+1:], // Last word of typeUri
			proxy.GetCommonName(),
			time.Since(updateStartTime))
	}

	log.Info().Msgf("Full update for %s took %s", proxy.GetCommonName(), time.Since(fullUpdateStartTime))
}

// makeRequestForAllSecrets constructs an SDS request AS IF an Envoy proxy sent it.
// This request will result in the rest of the system creating an SDS response with the certificates
// required by this proxy. The proxy itself did not ask for these. We know it needs them - so we send them.
func makeRequestForAllSecrets(proxy *envoy.Proxy, meshCatalog catalog.MeshCataloger) *xds_discovery.DiscoveryRequest {
	svcList, err := meshCatalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with CN=%q", proxy.GetCommonName())
		return nil
	}
	// Github Issue #1575
	serviceForProxy := svcList[0]

	proxyIdentity, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with CN=%q", proxy.GetCommonName())
		return nil
	}

	discoveryRequest := &xds_discovery.DiscoveryRequest{
		ResourceNames: []string{
			envoy.SDSCert{
				MeshService: serviceForProxy,
				CertType:    envoy.ServiceCertType,
			}.String(),
			envoy.SDSCert{
				MeshService: serviceForProxy,
				CertType:    envoy.RootCertTypeForMTLSInbound,
			}.String(),
			envoy.SDSCert{
				MeshService: serviceForProxy,
				CertType:    envoy.RootCertTypeForHTTPS,
			}.String(),
		},
		TypeUrl: string(envoy.TypeSDS),
	}

	// There is an SDS validation cert corresponding to each upstream service
	upstreamServices := meshCatalog.ListAllowedOutboundServicesForIdentity(proxyIdentity)
	for _, upstream := range upstreamServices {
		upstreamRootCertResource := envoy.SDSCert{
			MeshService: upstream,
			CertType:    envoy.RootCertTypeForMTLSOutbound,
		}.String()
		discoveryRequest.ResourceNames = append(discoveryRequest.ResourceNames, upstreamRootCertResource)
	}

	return discoveryRequest
}

func (s *Server) newAggregatedDiscoveryResponse(proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, cfg configurator.Configurator) (*xds_discovery.DiscoveryResponse, error) {
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

	// request.Node is only available on the first Discovery Request; will be nil on the following
	nodeID := ""
	if request.Node != nil {
		nodeID = request.Node.Id
	}

	log.Trace().Msgf("Invoking handler for type %s; request from Envoy with Node ID %s", typeURL, nodeID)
	response, err := handler(s.catalog, proxy, request, cfg, s.certManager)
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
