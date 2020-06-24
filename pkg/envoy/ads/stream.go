package ads

import (
	"context"
	"strconv"

	"github.com/open-service-mesh/osm/pkg/configurator"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/utils"
)

// StreamAggregatedResources handles streaming of the clusters to the connected Envoy proxies
func (s *Server) StreamAggregatedResources(server discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil)
	if err != nil {
		return errors.Wrap(err, "[%s] Could not start stream")
	}

	// TODO(draychev): check for envoy.ErrTooManyConnections

	ip := utils.GetIPFromContext(server.Context())

	namespacedService, err := s.catalog.GetServiceFromEnvoyCertificate(cn)
	if err != nil {
		log.Error().Err(err).Msgf("Error fetching service for Envoy %s with CN %s", ip, cn)
		return err
	}
	log.Info().Msgf("Client %s connected: Subject CN=%s; Service=%s", ip, cn, namespacedService)

	proxy := envoy.NewProxy(cn, ip)
	s.catalog.RegisterProxy(proxy)
	defer s.catalog.UnregisterProxy(proxy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requests := make(chan v2.DiscoveryRequest)
	go receive(requests, &server, proxy)

	// TODO(draychev): create this struct with data from the OSM Config CRD when that is ready.
	config := &configurator.Config{
		OSMNamespace:     s.configurator.GetOSMNamespace(),
		EnablePrometheus: true,
		EnableTracing:    true,
	}

	for {

		select {
		case <-ctx.Done():
			return nil

		case discoveryRequest, ok := <-requests:
			log.Info().Msgf("Received %s (nonce=%s; version=%s) from Envoy %s", discoveryRequest.TypeUrl, discoveryRequest.ResponseNonce, discoveryRequest.VersionInfo, proxy.GetCommonName())
			log.Info().Msgf("Last sent for %s nonce=%s; last sent version=%s for Envoy %s", discoveryRequest.TypeUrl, discoveryRequest.ResponseNonce, discoveryRequest.VersionInfo, proxy.GetCommonName())
			if !ok {
				log.Error().Msgf("Proxy %s closed GRPC", proxy)
				return errGrpcClosed
			}

			if discoveryRequest.ErrorDetail != nil {
				log.Error().Msgf("[NACK] Discovery request error from proxy %s: %s", proxy, discoveryRequest.ErrorDetail)
				return errEnvoyError
			}

			typeURL := envoy.TypeURI(discoveryRequest.TypeUrl)

			ackVersion, err := strconv.ParseUint(discoveryRequest.VersionInfo, 10, 64)
			if err != nil && discoveryRequest.VersionInfo != "" {
				log.Error().Err(err).Msgf("Error parsing %s discovery request VersionInfo (%s) from proxy %s", typeURL, discoveryRequest.VersionInfo, proxy.GetCommonName())
				ackVersion = 0
			}

			log.Debug().Msgf("Incoming Discovery Request %s (nonce=%s; version=%d) from Envoy %s; last applied version: %d",
				discoveryRequest.TypeUrl, discoveryRequest.ResponseNonce, ackVersion, proxy.GetCommonName(), proxy.GetLastAppliedVersion(typeURL))
			log.Debug().Msgf("Last sent nonce=%s; last sent version=%d for Envoy %s",
				proxy.GetLastSentNonce(typeURL), proxy.GetLastSentVersion(typeURL), proxy.GetCommonName())

			proxy.SetLastAppliedVersion(typeURL, ackVersion)

			// In the DiscoveryRequest we have a VersionInfo field.
			// When this is smaller or equal to what we last sent to this proxy - it is
			// interpreted as an acknoweldgement of a previously sent request.
			// Such DiscoveryRequest requires no further action.
			if ackVersion > 0 && ackVersion <= proxy.GetLastSentVersion(typeURL) {
				log.Debug().Msgf("Request %s VersionInfo (%d) <= last sent VersionInfo (%d); ACK", typeURL, ackVersion, proxy.GetLastSentVersion(typeURL))
				continue
			}

			// The version of the config received along with the DiscoveryRequest (ackVersion)
			// is what the Envoy proxy may be acknowledging. It is acknowledging
			// and not requesting when the ackVersion is <= what we last sent.
			// It is possible however for a proxy to have a version that is higher
			// than what we last sent. (Perhaps the control plane restarted.)
			// In that case we want to make sure that we send new responses with
			// VersionInfo incremented starting with the version which the proxy last had.
			if ackVersion > proxy.GetLastSentVersion(typeURL) {
				proxy.SetLastSentVersion(typeURL, ackVersion)
			}

			lastNonce := proxy.GetLastSentNonce(typeURL)
			if lastNonce != "" && discoveryRequest.ResponseNonce == lastNonce {
				log.Debug().Msgf("Nothing changed since Nonce=%s", discoveryRequest.ResponseNonce)
				continue
			}

			if discoveryRequest.ResponseNonce != "" {
				log.Debug().Msgf("Received discovery request with Nonce=%s; matches=%t; proxy last Nonce=%s", discoveryRequest.ResponseNonce, discoveryRequest.ResponseNonce == lastNonce, lastNonce)
			}
			log.Info().Msgf("Received discovery request <%s> from Envoy <%s> with Nonce=%s", discoveryRequest.TypeUrl, proxy, discoveryRequest.ResponseNonce)

			resp, err := s.newAggregatedDiscoveryResponse(proxy, &discoveryRequest, config)
			if err != nil {
				log.Error().Err(err).Msgf("Error composing a DiscoveryResponse")
				continue
			}

			if err := server.Send(resp); err != nil {
				log.Error().Err(err).Msgf("Error sending DiscoveryResponse")
			}

		case <-proxy.GetAnnouncementsChannel():
			log.Info().Msgf("Change detected - update all Envoys.")
			s.sendAllResponses(proxy, &server, config)

		}
	}
}
