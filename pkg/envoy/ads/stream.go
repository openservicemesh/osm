package ads

import (
	"context"
	"fmt"
	"strconv"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"

	"github.com/open-service-mesh/osm/pkg/utils"
)

// StreamAggregatedResources handles streaming of the clusters to the connected Envoy proxies
func (s *Server) StreamAggregatedResources(server discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(server.Context(), nil, packageName)
	if err != nil {
		return errors.Wrap(err, "[%s] Could not start stream")
	}

	// TODO(draychev): check for envoy.ErrTooManyConnections

	log.Info().Msgf("[%s] Client connected: Subject CN=%s", packageName, cn)

	// Register the newly connected proxy w/ the catalog.
	// TODO(draychev): this does not produce the correct IP address
	ip := utils.GetIPFromContext(server.Context())

	// TODO: Need a better way to map a proxy to a service. This
	// is primarly required because envoy configurations are programmed
	// per service.
	cnMeta := utils.GetCertificateCommonNameMeta(cn.String())
	namespacedSvcAcc := endpoint.NamespacedServiceAccount{
		Namespace:      cnMeta.Namespace,
		ServiceAccount: cnMeta.ServiceAccountName,
	}
	services := s.catalog.GetServicesByServiceAccountName(namespacedSvcAcc, true)
	if len(services) == 0 {
		// No services found for this service account, don't patch
		return fmt.Errorf("No service found for service account %q", namespacedSvcAcc)
	}
	// TODO: Don't assume a service account maps to a single service
	namespacedService := services[0]
	log.Info().Msgf("cert: cn=%s, service=%s", cn, namespacedService)

	proxy := envoy.NewProxy(cn, namespacedService, ip)
	s.catalog.RegisterProxy(proxy)
	defer s.catalog.UnregisterProxy(proxy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requests := make(chan v2.DiscoveryRequest)
	go receive(requests, &server)

	for {
		select {
		case <-ctx.Done():
			return nil

		case discoveryRequest, ok := <-requests:
			log.Info().Msgf("[%s] Discovery Request %s (nonce=%s; version=%s) from Envoy %s", packageName, discoveryRequest.TypeUrl, discoveryRequest.ResponseNonce, discoveryRequest.VersionInfo, proxy.GetCommonName())
			log.Info().Msgf("[%s] Last sent for %s nonce=%s; last sent version=%s for Envoy %s", packageName, discoveryRequest.TypeUrl, discoveryRequest.ResponseNonce, discoveryRequest.VersionInfo, proxy.GetCommonName())
			if !ok {
				log.Error().Msgf("[%s] Proxy %s closed GRPC", packageName, proxy)
				return errGrpcClosed
			}

			if discoveryRequest.ErrorDetail != nil {
				log.Error().Msgf("[%s] Discovery request error from proxy %s: %s", packageName, proxy, discoveryRequest.ErrorDetail)
				return errEnvoyError
			}

			typeURL := envoy.TypeURI(discoveryRequest.TypeUrl)

			ackVersion, err := strconv.ParseUint(discoveryRequest.VersionInfo, 10, 64)
			if err != nil && discoveryRequest.VersionInfo != "" {
				log.Error().Err(err).Msgf("[%s] Error parsing %s discovery request VersionInfo (%s) from proxy %s", packageName, typeURL, discoveryRequest.VersionInfo, proxy.GetCommonName())
				ackVersion = 0
			}

			log.Debug().Msgf("[%s] Incoming Discovery Request %s (nonce=%s; version=%d) from Envoy %s; last applied version: %d",
				packageName, discoveryRequest.TypeUrl, discoveryRequest.ResponseNonce, ackVersion, proxy.GetCommonName(), proxy.GetLastAppliedVersion(typeURL))
			log.Debug().Msgf("[%s] Last sent nonce=%s; last sent version=%d for Envoy %s",
				packageName, proxy.GetLastSentNonce(typeURL), proxy.GetLastSentVersion(typeURL), proxy.GetCommonName())

			proxy.SetLastAppliedVersion(typeURL, ackVersion)

			if ackVersion > 0 && ackVersion <= proxy.GetLastSentVersion(typeURL) {
				log.Debug().Msgf("[%s] %s Discovery Request VersionInfo (%d) <= last sent VersionInfo (%d); ACK", packageName, typeURL, ackVersion, proxy.GetLastSentVersion(typeURL))
				continue
			}

			lastNonce := proxy.GetLastSentNonce(typeURL)
			if lastNonce != "" && discoveryRequest.ResponseNonce == lastNonce {
				log.Debug().Msgf("[%s] Nothing changed since Nonce=%s", packageName, discoveryRequest.ResponseNonce)
				continue
			}

			if discoveryRequest.ResponseNonce != "" {
				log.Debug().Msgf("[%s] Received discovery request with Nonce=%s; matches=%t; proxy last Nonce=%s", packageName, discoveryRequest.ResponseNonce, discoveryRequest.ResponseNonce == lastNonce, lastNonce)
			}
			log.Info().Msgf("[%s] Received discovery request <%s> from Envoy <%s> with Nonce=%s", packageName, discoveryRequest.TypeUrl, proxy, discoveryRequest.ResponseNonce)

			resp, err := s.newAggregatedDiscoveryResponse(proxy, &discoveryRequest)
			if err != nil {
				log.Error().Err(err).Msgf("[%s] Error composing a DiscoveryResponse", packageName)
				continue
			}

			if err := server.Send(resp); err != nil {
				log.Error().Err(err).Msgf("[%s] Error sending DiscoveryResponse", packageName)
			} else {
				log.Debug().Msgf("[%s] Sent Discovery Response %s to proxy %s: %s", packageName, resp.TypeUrl, proxy, resp)
			}

		case <-proxy.GetAnnouncementsChannel():
			log.Info().Msgf("[%s] Change detected - update all Envoys.", packageName)
			s.sendAllResponses(proxy, &server)
		}
	}
}
