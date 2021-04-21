package ads

import (
	"context"
	"strconv"
	"strings"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/utils"
)

// StreamAggregatedResources handles streaming of the clusters to the connected Envoy proxies
// This is evaluated once per new Envoy proxy connecting and remains running for the duration of the gRPC socket.
func (s *Server) StreamAggregatedResources(server xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	certCommonName, certSerialNumber, err := utils.ValidateClient(server.Context(), nil)
	if err != nil {
		return errors.Wrap(err, "Could not start Aggregated Discovery Service gRPC stream for newly connected Envoy proxy")
	}

	// If maxDataPlaneConnections is enabled i.e. not 0, then check that the number of Envoy connections is less than maxDataPlaneConnections
	if s.cfg.GetMaxDataPlaneConnections() != 0 && s.proxyRegistry.GetConnectedProxyCount() >= s.cfg.GetMaxDataPlaneConnections() {
		return errTooManyConnections
	}

	log.Trace().Msgf("Envoy with certificate SerialNumber=%s connected", certSerialNumber)
	metricsstore.DefaultMetricsStore.ProxyConnectCount.Inc()

	// This is the Envoy proxy that just connected to the control plane.
	// NOTE: This is step 1 of the registration. At this point we do not yet have context on the Pod.
	//       Details on which Pod this Envoy is fronting will arrive via xDS in the NODE_ID string.
	//       When this arrives we will call RegisterProxy() a second time - this time with Pod context!
	proxy := envoy.NewProxy(certCommonName, certSerialNumber, utils.GetIPFromContext(server.Context()))
	s.proxyRegistry.RegisterProxy(proxy) // First of Two invocations.  Second one will be during xDS hand-shake!

	defer s.proxyRegistry.UnregisterProxy(proxy)

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	quit := make(chan struct{})
	requests := make(chan xds_discovery.DiscoveryRequest)

	// This helper handles receiving messages from the connected Envoys
	// and any gRPC error states.
	go receive(requests, &server, proxy, quit, s.proxyRegistry)

	// Register to Envoy global broadcast updates
	broadcastUpdate := events.GetPubSubInstance().Subscribe(announcements.ProxyBroadcast)

	// Register for certificate rotation updates
	certAnnouncement := events.GetPubSubInstance().Subscribe(announcements.CertificateRotated)

	// Issues a send all response on a connecting envoy
	// If this were to fail, it most likely just means we still have configuration being applied on flight,
	// which will get triggered by the dispatcher anyway
	if err = s.sendResponse(proxy, &server, nil, s.cfg, envoy.XDSResponseOrder...); err != nil {
		log.Error().Err(err).Msgf("Initial sendResponse for proxy %s returned error", proxy.GetCertificateSerialNumber())
	}

	newJob := func(typeURIs []envoy.TypeURI, discoveryRequest *xds_discovery.DiscoveryRequest) *proxyResponseJob {
		return &proxyResponseJob{
			typeURIs:  typeURIs,
			proxy:     proxy,
			adsStream: &server,
			request:   discoveryRequest,
			xdsServer: s,
			done:      make(chan struct{}),
		}
	}

	for {
		select {
		case <-ctx.Done():
			metricsstore.DefaultMetricsStore.ProxyConnectCount.Dec()
			return nil

		case <-quit:
			log.Debug().Msgf("gRPC stream with Envoy on Pod with UID=%s closed!", proxy.GetPodUID())
			metricsstore.DefaultMetricsStore.ProxyConnectCount.Dec()
			return nil

		case discoveryRequest, ok := <-requests:
			if !ok {
				log.Error().Msgf("Envoy with xDS Certificate SerialNumber=%s on Pod with UID=%s closed gRPC!", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
				metricsstore.DefaultMetricsStore.ProxyConnectCount.Dec()
				return errGrpcClosed
			}

			// This function call runs xDS proto state machine given DiscoveryRequest as input.
			// It's output is the decision to reply or not to this request.
			if !respondToRequest(proxy, &discoveryRequest) {
				continue
			}

			typeURL := envoy.TypeURI(discoveryRequest.TypeUrl)
			var typesRequest []envoy.TypeURI
			if typeURL == envoy.TypeWildcard {
				typesRequest = envoy.XDSResponseOrder
			} else {
				typesRequest = []envoy.TypeURI{typeURL}
			}

			<-s.workqueues.AddJob(newJob(typesRequest, &discoveryRequest))

		case <-broadcastUpdate:
			log.Info().Msgf("Broadcast wake for Proxy SerialNumber=%s UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())

			// Queue a full configuration update
			<-s.workqueues.AddJob(newJob(envoy.XDSResponseOrder, nil))

		case certUpdateMsg := <-certAnnouncement:
			cert := certUpdateMsg.(events.PubSubMessage).NewObj.(certificate.Certificater)
			if isCNforProxy(proxy, cert.GetCommonName()) {
				// The CN whose corresponding certificate was updated (rotated) by the certificate provider is associated
				// with this proxy, so update the secrets corresponding to this certificate via SDS.
				log.Debug().Msgf("Certificate has been updated for proxy with SerialNumber=%s, UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())

				// Empty DiscoveryRequest should create the SDS specific request
				// Prepare to queue the SDS proxy response job on the worker pool
				<-s.workqueues.AddJob(newJob([]envoy.TypeURI{envoy.TypeSDS}, nil))
			}
		}
	}
}

// respondToRequest assesses if a given DiscoveryRequest for a given proxy should be responded with
// an xDS DiscoveryResponse.
func respondToRequest(proxy *envoy.Proxy, discoveryRequest *xds_discovery.DiscoveryRequest) bool {
	var err error
	var ackVersion uint64

	log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: Request %s [nonce=%s; version=%s; resources=%v] last sent [nonce=%s; version=%d]",
		proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), discoveryRequest.TypeUrl,
		discoveryRequest.ResponseNonce, discoveryRequest.VersionInfo, discoveryRequest.ResourceNames,
		proxy.GetLastSentNonce(envoy.TypeURI(discoveryRequest.TypeUrl)), proxy.GetLastSentVersion(envoy.TypeURI(discoveryRequest.TypeUrl)))

	if discoveryRequest.ErrorDetail != nil {
		log.Error().Msgf("Proxy SerialNumber=%s PodUID=%s: [NACK] err: \"%s\" for nonce %s, last version applied on request %s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), discoveryRequest.ErrorDetail, discoveryRequest.ResponseNonce, discoveryRequest.VersionInfo)
		return false
	}

	typeURL, ok := envoy.ValidURI[discoveryRequest.TypeUrl]
	if !ok {
		log.Error().Msgf("Proxy SerialNumber=%s PodUID=%s: Unknown/Unsupported URI: %s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), discoveryRequest.TypeUrl)
		return false
	}

	// It is possible for Envoy to return an empty VersionInfo.
	// When that's the case - start with 0
	if discoveryRequest.VersionInfo != "" {
		if ackVersion, err = strconv.ParseUint(discoveryRequest.VersionInfo, 10, 64); err != nil {
			// It is probable that Envoy responded with a VersionInfo we did not understand
			log.Error().Err(err).Msgf("Proxy SerialNumber=%s PodUID=%s: Error parsing DiscoveryRequest with TypeURL=%s VersionInfo=%s (%v)",
				proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), typeURL.Short(), discoveryRequest.VersionInfo, err)
			return false
		}
	}

	// Set last version applied
	proxy.SetLastAppliedVersion(typeURL, ackVersion)

	// In the DiscoveryRequest we have a VersionInfo field.
	// When this is smaller or equal to what we last sent to this proxy - it is
	// interpreted as an acknowledgement of a previously sent request.
	// Such DiscoveryRequest requires no further action.
	if ackVersion > 0 && ackVersion <= proxy.GetLastSentVersion(typeURL) {
		log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: Skipping request of type %s for resources [%v],  VersionInfo (%d) <= last sent VersionInfo (%d); ACK",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), typeURL, discoveryRequest.ResourceNames, ackVersion, proxy.GetLastSentVersion(typeURL))
		return false
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
		log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: Nothing changed for Envoy on Pod since Nonce=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), discoveryRequest.ResponseNonce)
		return false
	}

	if discoveryRequest.ResponseNonce != "" {
		log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: Received discovery request with Nonce=%s matches=%t; proxy last Nonce=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), discoveryRequest.ResponseNonce, discoveryRequest.ResponseNonce == lastNonce, lastNonce)
	}

	return true
}

// isCNforProxy returns true if the given CN for the workload certificate matches the given proxy's identity.
// Proxy identity corresponds to the k8s service account, while the workload certificate is of the form
// <svc-account>.<namespace>.<trust-domain>.
func isCNforProxy(proxy *envoy.Proxy, cn certificate.CommonName) bool {
	proxyIdentity, err := catalog.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return false
	}

	// Workload certificate CN is of the form <svc-account>.<namespace>.<trust-domain>
	chunks := strings.Split(cn.String(), constants.DomainDelimiter)
	if len(chunks) < 3 {
		return false
	}

	identityForCN := identity.K8sServiceAccount{Name: chunks[0], Namespace: chunks[1]}
	return identityForCN == proxyIdentity
}
