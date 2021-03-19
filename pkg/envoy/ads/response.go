package ads

import (
	"strconv"
	"time"

	mapset "github.com/deckarep/golang-set"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
)

const (
	// ADSUpdateStr is a constant string value to identify full XDS update times on metric labels
	ADSUpdateStr = "ADS"
)

// Wrapper to create and send a discovery response to an envoy server
func (s *Server) sendTypeResponse(tURI envoy.TypeURI,
	proxy *envoy.Proxy, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer,
	req *xds_discovery.DiscoveryRequest, cfg configurator.Configurator) error {
	// Tracks the success of this TypeURI response operation; accounts also for receipt on envoy server side
	success := false
	xdsShortName := envoy.XDSShortURINames[tURI]
	defer xdsPathTimeTrack(time.Now(), log.Debug(), xdsShortName, proxy.GetCertificateSerialNumber().String(), &success)

	log.Trace().Msgf("[%s] Creating response for proxy with SerialNumber=%s on Pod with UID=%s", xdsShortName, proxy.GetCertificateSerialNumber(), proxy.GetPodUID())

	discoveryResponse, err := s.newAggregatedDiscoveryResponse(proxy, req, cfg)
	if err != nil {
		log.Error().Err(err).Msgf("[%s] Failed to create response for proxy with SerialNumber=%s on Pod with UID=%s", xdsShortName, proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return err
	}

	if err := (*server).Send(discoveryResponse); err != nil {
		log.Error().Err(err).Msgf("[%s] Error sending to proxy with SerialNumber=%s on Pod with UID=%s", xdsShortName, proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return err
	}

	success = true // read by deferred function
	return nil
}

// sendResponse takes a set of TypeURIs which will be called to generate the xDS resources
// for, and will have them sent to the proxy server.
// If no DiscoveryRequest is passed, an empty one for the TypeURI is created
func (s *Server) sendResponse(typeURIsToSend mapset.Set,
	proxy *envoy.Proxy,
	server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer,
	request *xds_discovery.DiscoveryRequest,
	cfg configurator.Configurator) error {
	success := true
	if typeURIsToSend.Cardinality() == len(envoy.XDSResponseOrder) {
		defer xdsPathTimeTrack(time.Now(), log.Info(), ADSUpdateStr, proxy.GetCertificateSerialNumber().String(), &success)
	}

	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for _, typeURI := range envoy.XDSResponseOrder {
		if !typeURIsToSend.Contains(typeURI) {
			continue
		}

		// Handle request when is not provided, and the SDS case
		var finalReq *xds_discovery.DiscoveryRequest
		if request == nil || request.TypeUrl == envoy.TypeWildcard.String() {
			if typeURI == envoy.TypeSDS {
				finalReq = makeRequestForAllSecrets(proxy, s.catalog)
				if finalReq == nil {
					continue
				}
			} else {
				finalReq = &xds_discovery.DiscoveryRequest{TypeUrl: string(typeURI)}
			}
		} else {
			finalReq = request
		}

		err := s.sendTypeResponse(typeURI, proxy, server, finalReq, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create %s update for Proxy %s",
				envoy.XDSShortURINames[typeURI], proxy.GetCertificateCommonName())
			success = false
		}
	}

	return nil
}

func (s *Server) newAggregatedDiscoveryResponse(proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, cfg configurator.Configurator) (*xds_discovery.DiscoveryResponse, error) {
	typeURL := envoy.TypeURI(request.TypeUrl)
	handler, ok := s.xdsHandlers[typeURL]
	if !ok {
		log.Error().Msgf("Responder for TypeUrl %s is not implemented", request.TypeUrl)
		return nil, errUnknownTypeURL
	}

	if s.cfg.IsDebugServerEnabled() {
		s.trackXDSLog(proxy.GetCertificateCommonName(), typeURL)
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

	// NOTE: Never log entire 'response' - will contain secrets!
	log.Trace().Msgf("Constructed %s response: VersionInfo=%s", response.TypeUrl, response.VersionInfo)

	return response, nil
}
