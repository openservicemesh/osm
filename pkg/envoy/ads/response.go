package ads

import (
	"strconv"
	"time"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
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
	defer xdsPathTimeTrack(time.Now(), xdsShortName, proxy.GetCertificateCommonName().String(), &success)

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

func (s *Server) sendAllResponses(proxy *envoy.Proxy, server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, cfg configurator.Configurator) {
	log.Trace().Msgf("A change announcement triggered *DS update for proxy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())

	// Tracks the success of this full update of all its XDS paths. If a single XDS response path fails for this full update,
	// the full updated will be considered as failed for metric purposes (success = false)
	success := true
	defer xdsPathTimeTrack(time.Now(), ADSUpdateStr, proxy.GetCertificateCommonName().String(), &success)

	// Order is important: CDS, EDS, LDS, RDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for _, typeURI := range envoy.XDSResponseOrder {
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

		err := s.sendTypeResponse(typeURI, proxy, server, request, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create and send %s update to Proxy %s",
				envoy.XDSShortURINames[typeURI], proxy.GetCertificateCommonName())
			success = false
		}
	}
}

// makeRequestForAllSecrets constructs an SDS request AS IF an Envoy proxy sent it.
// This request will result in the rest of the system creating an SDS response with the certificates
// required by this proxy. The proxy itself did not ask for these. We know it needs them - so we send them.
func makeRequestForAllSecrets(proxy *envoy.Proxy, meshCatalog catalog.MeshCataloger) *xds_discovery.DiscoveryRequest {
	svcList, err := meshCatalog.GetServicesFromEnvoyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil
	}
	// Github Issue #1575
	serviceForProxy := svcList[0]

	proxyIdentity, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
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
		if _, ok := s.xdsLog[proxy.GetCertificateCommonName()]; !ok {
			s.xdsLog[proxy.GetCertificateCommonName()] = make(map[envoy.TypeURI][]time.Time)
		}
		s.xdsLog[proxy.GetCertificateCommonName()][typeURL] = append(s.xdsLog[proxy.GetCertificateCommonName()][typeURL], time.Now())
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
