package ads

import (
	"strconv"
	"time"

	mapset "github.com/deckarep/golang-set"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
)

const (
	// ADSUpdateStr is a constant string value to identify full XDS update times on metric labels
	ADSUpdateStr = "ADS"
)

// Wrapper to create and send a discovery response to an envoy server
func (s *Server) sendTypeResponse(typeURI envoy.TypeURI, proxy *envoy.Proxy, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, req *xds_discovery.DiscoveryRequest, cfg configurator.Configurator) error {
	// Tracks the success of this TypeURI response operation; accounts also for receipt on envoy server side
	startedAt := time.Now()
	log.Trace().Msgf("[%s] Creating response for proxy with SerialNumber=%s on Pod with UID=%s", typeURI.Short(), proxy.GetCertificateSerialNumber(), proxy.GetPodUID())

	if discoveryResponse, err := s.newAggregatedDiscoveryResponse(proxy, req, cfg); err != nil {
		log.Error().Err(err).Msgf("[%s] Failed to create response for proxy with SerialNumber=%s on Pod with UID=%s", typeURI.Short(), proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		xdsPathTimeTrack(startedAt, log.Debug(), typeURI, proxy, false)
		return err
	} else if err := (*server).Send(discoveryResponse); err != nil {
		log.Error().Err(err).Msgf("[%s] Error sending to proxy with SerialNumber=%s on Pod with UID=%s", typeURI.Short(), proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		xdsPathTimeTrack(startedAt, log.Debug(), typeURI, proxy, false)
		return err
	}

	xdsPathTimeTrack(startedAt, log.Debug(), typeURI, proxy, true)
	return nil
}

// sendResponse takes a set of TypeURIs which will be called to generate the xDS resources
// for, and will have them sent to the proxy server.
// If no DiscoveryRequest is passed, an empty one for the TypeURI is created
// TODO(draychev): Convert to variadic function: https://github.com/openservicemesh/osm/issues/3127
func (s *Server) sendResponse(typeURIsToSend mapset.Set,
	proxy *envoy.Proxy,
	server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer,
	request *xds_discovery.DiscoveryRequest,
	cfg configurator.Configurator) error {
	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	var typesToSend []envoy.TypeURI
	for _, typeURI := range envoy.XDSResponseOrder {
		if typeURIsToSend.Contains(typeURI) {
			typesToSend = append(typesToSend, typeURI)
		}
	}

	if len(typesToSend) < 1 {
		// This should never happen.
		log.Error().Msg("The list of discovery responses passed to sendResponse() is empty. This is an invalid invocation of sendResponse().")
		return errors.New("list of discovery responses is empty")
	}

	thereWereErrors := false

	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for _, typeURI := range typesToSend {
		// A nil request indicates a request for all SDS responses
		fullUpdateRequested := request == nil || envoy.TypeURI(request.TypeUrl).IsWildcard()

		// Handle request when is not provided, and the SDS case
		var finalReq *xds_discovery.DiscoveryRequest
		if fullUpdateRequested {
			if typeURI == envoy.TypeSDS {
				finalReq = makeRequestForAllSecrets(proxy, s.catalog)
				if finalReq == nil {
					continue
				}
			} else {
				finalReq = &xds_discovery.DiscoveryRequest{TypeUrl: typeURI.String()}
			}
		} else {
			finalReq = request
		}

		if err := s.sendTypeResponse(typeURI, proxy, server, finalReq, cfg); err != nil {
			log.Error().Err(err).Msgf("Creating %s update for Proxy %s", typeURI.Short(), proxy.GetCertificateCommonName())
			thereWereErrors = true
		}
	}

	isFullUpdate := len(typesToSend) == len(envoy.XDSResponseOrder)
	if isFullUpdate {
		success := !thereWereErrors
		xdsPathTimeTrack(time.Now(), log.Info(), envoy.TypeADS, proxy, success)
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
	resources, err := handler(s.catalog, proxy, request, cfg, s.certManager)
	if err != nil {
		log.Error().Err(err).Msgf("Handler errored TypeURL: %s, proxy: %s", request.TypeUrl, proxy.GetCertificateSerialNumber())
		return nil, errCreatingResponse
	}

	response := &xds_discovery.DiscoveryResponse{
		TypeUrl:     request.TypeUrl, // Request TypeURL
		VersionInfo: strconv.FormatUint(proxy.IncrementLastSentVersion(typeURL), 10),
		Nonce:       proxy.SetNewNonce(typeURL),
	}

	for _, res := range resources {
		proto, err := ptypes.MarshalAny(res)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling resource %s for proxy %s", typeURL.String(), proxy.GetCertificateSerialNumber())
			continue
		}
		response.Resources = append(response.Resources, proto)
	}

	// NOTE: Never log entire 'response' - will contain secrets!
	log.Trace().Msgf("Constructed %s response: VersionInfo=%s", response.TypeUrl, response.VersionInfo)

	return response, nil
}
