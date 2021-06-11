package ads

import (
	"strconv"
	"time"

	mapset "github.com/deckarep/golang-set"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/golang/protobuf/ptypes"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// getTypeResource invokes the XDS handler (LDS, CDS etc.) to respond to the XDS request containing the requests' type and associated resources
func (s *Server) getTypeResources(proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest) ([]types.Resource, error) {
	// Tracks the success of this TypeURI response operation; accounts also for receipt on envoy server side
	startedAt := time.Now()
	typeURI := envoy.TypeURI(request.TypeUrl)
	log.Trace().Msgf("Proxy %s: getting resources for type %s", proxy.String(), typeURI.Short())

	handler, ok := s.xdsHandlers[typeURI]
	if !ok {
		return nil, errUnknownTypeURL
	}

	if s.cfg.IsDebugServerEnabled() {
		s.trackXDSLog(proxy.GetCertificateCommonName(), typeURI)
	}

	// Invoke XDS handler
	resources, err := handler(s.catalog, proxy, request, s.cfg, s.certManager, s.proxyRegistry)
	if err != nil {
		xdsPathTimeTrack(startedAt, log.Debug(), typeURI, proxy, false)
		return nil, errCreatingResponse
	}

	xdsPathTimeTrack(startedAt, log.Debug(), typeURI, proxy, true)
	return resources, nil
}

// sendResponse takes a set of TypeURIs which will be called to generate the xDS resources
// for, and will have them sent to the proxy server.
// If no DiscoveryRequest is passed, an empty one for the TypeURI is created
// TODO(draychev): Convert to variadic function: https://github.com/openservicemesh/osm/issues/3127
func (s *Server) sendResponse(proxy *envoy.Proxy, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, request *xds_discovery.DiscoveryRequest, cfg configurator.Configurator, typeURIsToSend ...envoy.TypeURI) error {
	thereWereErrors := false

	// A nil request indicates a request for all SDS responses
	fullUpdateRequested := request == nil
	cacheResourceMap := map[envoy.TypeURI][]types.Resource{}

	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for _, typeURI := range typeURIsToSend {
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

		// Generate the resources for this request
		resources, err := s.getTypeResources(proxy, finalReq)
		if err != nil {
			log.Error().Err(err).Msgf("Creating %s update for Proxy %s", typeURI.Short(), proxy.GetCertificateCommonName())
			thereWereErrors = true
			continue
		}

		if s.cacheEnabled {
			// Keep a reference to later set the full snapshot in the cache
			cacheResourceMap[typeURI] = resources
		} else {
			// If cache disabled, craft and send a reply to the proxy on the stream
			if err := s.SendDiscoveryResponse(proxy, finalReq, server, resources); err != nil {
				log.Error().Err(err).Msgf("Creating %s update for Proxy %s", typeURI.Short(), proxy.GetCertificateCommonName())
				thereWereErrors = true
			}
		}
	}

	if s.cacheEnabled {
		// Store the aggregated resources as a full snapshot
		if err := s.RecordFullSnapshot(proxy, cacheResourceMap); err != nil {
			log.Error().Err(err).Msgf("Failed to record snapshot for proxy %s: %v", proxy.GetCertificateCommonName(), err)
			thereWereErrors = true
		}
	}

	isFullUpdate := len(typeURIsToSend) == len(envoy.XDSResponseOrder)
	if isFullUpdate {
		success := !thereWereErrors
		xdsPathTimeTrack(time.Now(), log.Info(), envoy.TypeADS, proxy, success)
	}

	return nil
}

// SendDiscoveryResponse creates a new response for <proxy> given <resourcesToSend> and <request.TypeURI> and sends it
func (s *Server) SendDiscoveryResponse(proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, resourcesToSend []types.Resource) error {
	// request.Node is only available on the first Discovery Request; will be nil on the following
	typeURI := envoy.TypeURI(request.TypeUrl)

	response := &xds_discovery.DiscoveryResponse{
		TypeUrl:     request.TypeUrl,
		VersionInfo: strconv.FormatUint(proxy.IncrementLastSentVersion(typeURI), 10),
		Nonce:       proxy.SetNewNonce(typeURI),
	}

	resourcesSent := mapset.NewSet()
	for _, res := range resourcesToSend {
		proto, err := ptypes.MarshalAny(res)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling resource %s for proxy %s", typeURI, proxy.GetCertificateSerialNumber())
			continue
		}
		response.Resources = append(response.Resources, proto)
		resourcesSent.Add(cache.GetResourceName(res))
	}

	// NOTE: Never log entire 'response' - will contain secrets!
	log.Trace().Msgf("Constructed %s response: VersionInfo=%s", response.TypeUrl, response.VersionInfo)

	// Validate the generated resources given the request
	validateRequestResponse(proxy, request, resourcesToSend)

	// Send the response
	if err := (*server).Send(response); err != nil {
		log.Error().Err(err).Msgf("Error sending response for type %s to proxy %s", typeURI.Short(), proxy.String())
		return err
	}

	// Sending discovery response succeeded, record last resources sent
	// TODO: increase version and nonce only if Send succeeded
	proxy.SetLastResourcesSent(typeURI, resourcesSent)

	return nil
}
