package ads

import (
	"strconv"
	"time"

	mapset "github.com/deckarep/golang-set"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
)

const (
	// ADSUpdateStr is a constant string value to identify full XDS update times on metric labels
	ADSUpdateStr = "ADS"
)

// getXDSResource function will invoke the right TypeURI handler for the resources associated with a proxy
// Returns the list of resources to be sent associated to TypeURI and the proxy
func (s *Server) getXDSResource(tURI envoy.TypeURI, proxy *envoy.Proxy,
	req *xds_discovery.DiscoveryRequest, cfg configurator.Configurator) ([]types.Resource, error) {
	// Tracks the success of this TypeURI response operation; accounts also for receipt on envoy server side
	success := false
	xdsShortName := envoy.XDSShortURINames[tURI]
	defer xdsPathTimeTrack(time.Now(), log.Debug(), xdsShortName, proxy.GetCertificateSerialNumber().String(), &success)

	log.Trace().Msgf("[%s] Creating response for proxy with SerialNumber=%s on Pod with UID=%s", xdsShortName, proxy.GetCertificateSerialNumber(), proxy.GetPodUID())

	handler, ok := s.xdsHandlers[tURI]
	if !ok {
		log.Error().Msgf("Responder for TypeUrl %s is not implemented", tURI)
		return nil, errUnknownTypeURL
	}

	if s.cfg.IsDebugServerEnabled() {
		s.trackXDSLog(proxy.GetCertificateCommonName(), tURI)
	}

	log.Trace().Msgf("Invoking handler for type %s, proxy Serial %s", tURI, proxy.GetCertificateSerialNumber())
	resources, err := handler(s.catalog, proxy, req, cfg, s.certManager)
	if err != nil {
		log.Error().Err(err).Msgf("Responder executing handler for %s", tURI)
		return nil, err
	}

	success = true // read by deferred function
	return resources, nil
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

	// Temporary store for XDS resources
	resourceMap := map[envoy.TypeURI][]types.Resource{}

	// Order is important: CDS, EDS, LDS, RDS, SDS
	// See: https://github.com/envoyproxy/go-control-plane/issues/59
	for _, typeURI := range envoy.XDSResponseOrder {
		if !typeURIsToSend.Contains(typeURI) {
			continue
		}

		// Handle request when is not provided, and the SDS case
		var finalReq *xds_discovery.DiscoveryRequest
		if request == nil {
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

		resources, err := s.getXDSResource(typeURI, proxy, finalReq, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create %s update for Proxy %s",
				envoy.XDSShortURINames[typeURI], proxy.GetCertificateCommonName())
			success = false
			return err
		}
		// Store resources for this TypeURI
		resourceMap[typeURI] = resources
	}

	// Send these to the proxy server
	err := s.sendToServer(proxy, server, resourceMap)

	return err
}

// makeRequestForAllSecrets constructs an SDS request AS IF an Envoy proxy sent it.
// This request will result in the rest of the system creating an SDS response with the certificates
// required by this proxy. The proxy itself did not ask for these. We know it needs them - so we send them.
func makeRequestForAllSecrets(proxy *envoy.Proxy, meshCatalog catalog.MeshCataloger) *xds_discovery.DiscoveryRequest {
	proxyIdentity, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil
	}

	discoveryRequest := &xds_discovery.DiscoveryRequest{
		ResourceNames: []string{
			envoy.SDSCert{
				Name:     proxyIdentity.String(),
				CertType: envoy.ServiceCertType,
			}.String(),
			envoy.SDSCert{
				Name:     proxyIdentity.String(),
				CertType: envoy.RootCertTypeForMTLSInbound,
			}.String(),
			envoy.SDSCert{
				Name:     proxyIdentity.String(),
				CertType: envoy.RootCertTypeForHTTPS,
			}.String(),
		},
		TypeUrl: string(envoy.TypeSDS),
	}

	// There is an SDS validation cert corresponding to each upstream service.
	// Each cert is used to validate the certificate presented by the corresponding upstream service.
	upstreamServices := meshCatalog.ListAllowedOutboundServicesForIdentity(proxyIdentity)
	for _, upstream := range upstreamServices {
		upstreamRootCertResource := envoy.SDSCert{
			Name:     upstream.String(),
			CertType: envoy.RootCertTypeForMTLSOutbound,
		}.String()
		discoveryRequest.ResourceNames = append(discoveryRequest.ResourceNames, upstreamRootCertResource)
	}

	return discoveryRequest
}

// sendToServer is a generic function that will generate and send
// DiscoveryResponses for a given map of <TypeURI> -> <Resources> passed
// by parametre.
// Works for one or more TypeURIs and/or resources to be sent.
// Proper XDS order is maintained when iterating (and sending) if multiple are sent.
func (s *Server) sendToServer(proxy *envoy.Proxy,
	server *xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer,
	xdsUpdates map[envoy.TypeURI][]types.Resource) error {
	// Walk the typeURIs to be sent in right order
	for _, typeURI := range envoy.XDSResponseOrder {
		resources, xdsUpdatePresent := xdsUpdates[typeURI]
		if !xdsUpdatePresent {
			// Skip non-present URIs
			continue
		}

		// Create the response
		response := xds_discovery.DiscoveryResponse{
			TypeUrl:     typeURI.String(),
			Resources:   []*anypb.Any{},
			VersionInfo: strconv.FormatUint(proxy.IncrementLastSentVersion(typeURI), 10),
			Nonce:       proxy.SetNewNonce(typeURI),
		}
		// Walk and marshal each resource for this TypeURI
		for _, res := range resources {
			proto, err := ptypes.MarshalAny(res)
			if err != nil {
				log.Error().Err(err).Msgf("Error marshalling resource %s", typeURI)
				continue
			}
			response.Resources = append(response.Resources, proto)
		}

		// Send to proxy server
		if err := (*server).Send(&response); err != nil {
			log.Error().Err(err).Msgf("Failed to send %s resources to proxi serial %s", typeURI, proxy.GetCertificateSerialNumber())
			return err
		}
	}

	return nil
}
