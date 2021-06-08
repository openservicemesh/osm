package ads

import (
	"context"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/announcements"
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
	proxy, err := envoy.NewProxy(certCommonName, certSerialNumber, utils.GetIPFromContext(server.Context()))
	if err != nil {
		log.Error().Err(err).Msgf("Error initializing proxy with certificate SerialNumber=%s", certSerialNumber)
		return err
	}

	if err := s.recordPodMetadata(proxy); err == errServiceAccountMismatch {
		// Service Account mismatch
		log.Error().Err(err).Msgf("Mismatched service account for proxy with certificate SerialNumber=%s", certSerialNumber)
		return err
	}

	s.proxyRegistry.RegisterProxy(proxy)

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

			typesRequest := []envoy.TypeURI{envoy.TypeURI(discoveryRequest.TypeUrl)}

			<-s.workqueues.AddJob(newJob(typesRequest, &discoveryRequest))

		case <-broadcastUpdate:
			log.Info().Msgf("Proxy SerialNumber=%s PodUID=%s: Broadcast wake", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())

			// Per protocol, we have to wait for the proxy to go through init phase (initial no-nonce request),
			// otherwise we will be generating versions that will be ignored as empty nonce will generate a new version anyway.
			// We only have to push an update from control plane if we have provided already something before.
			if !shouldPushUpdate(proxy) {
				log.Error().Msgf("Proxy SerialNumber=%s PodUID=%s: Proxy has still not gone through init phase, not force-pushing new version",
					proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
				continue
			}

			// Queue a full configuration update
			// Do not send SDS, let envoy figure out what certs does it want.
			<-s.workqueues.AddJob(newJob([]envoy.TypeURI{envoy.TypeCDS, envoy.TypeEDS, envoy.TypeLDS, envoy.TypeRDS}, nil))

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

// shouldPushUpdate handles allowing new updates to envoy from control-plane driven config changes.
// Its use is to make sure we don't unintentintionally push new versions if at least a first request has not arrived yet.
func shouldPushUpdate(proxy *envoy.Proxy) bool {
	// In ADS, CDS and LDS will come first in all cases. Only allow an control-plane-push update push if
	// we have sent either to the proxy already.
	if proxy.GetLastSentNonce(envoy.TypeLDS) == "" && proxy.GetLastSentNonce(envoy.TypeCDS) == "" {
		log.Error().Msgf("Proxy SerialNumber=%s PodUID=%s: LDS and CDS unrequested yet, waiting for first request for this proxy to be responded to",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return false
	}
	return true
}

// parseRequestVersion parses version from a DiscoveryRequest
func parseRequestVersion(discoveryRequest *xds_discovery.DiscoveryRequest) (uint64, error) {
	// Empty VersionInfo implies no configuration applied on the proxy. 0 is used to identify such case
	if discoveryRequest.VersionInfo == "" {
		return 0, nil
	}
	// Parse version otherwise
	return strconv.ParseUint(discoveryRequest.VersionInfo, 10, 64)
}

// respondToRequest assesses if a given DiscoveryRequest for a given proxy should be responded with
// an xDS DiscoveryResponse.
func respondToRequest(proxy *envoy.Proxy, discoveryRequest *xds_discovery.DiscoveryRequest) bool {
	var err error
	var requestVersion uint64
	var requestNonce string
	var lastNonce string

	log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: Request %s [nonce=%s; version=%s; resources=%v] last sent [nonce=%s; version=%d]",
		proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), discoveryRequest.TypeUrl,
		discoveryRequest.ResponseNonce, discoveryRequest.VersionInfo, discoveryRequest.ResourceNames,
		proxy.GetLastSentNonce(envoy.TypeURI(discoveryRequest.TypeUrl)), proxy.GetLastSentVersion(envoy.TypeURI(discoveryRequest.TypeUrl)))

	// Parse TypeURL of the request
	typeURL, ok := envoy.ValidURI[discoveryRequest.TypeUrl]
	if !ok {
		log.Error().Msgf("Proxy SerialNumber=%s PodUID=%s: Unknown/Unsupported URI: %s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), discoveryRequest.TypeUrl)
		return false
	}

	if typeURL == envoy.TypeEmptyURI {
		// Skip handling TypeEmptyURI for now, context #3258
		log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: Ignoring EmptyURI Type", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return false
	}

	// Parse ACK'd verion on the proxy for this given resource
	requestVersion, err = parseRequestVersion(discoveryRequest)
	if err != nil {
		log.Error().Err(err).Msgf("Proxy SerialNumber=%s PodUID=%s: Error parsing version %s for type %s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), discoveryRequest.VersionInfo, typeURL)
		return false
	}

	// Handle NACK case
	if discoveryRequest.ErrorDetail != nil {
		log.Error().Msgf("Proxy SerialNumber=%s PodUID=%s: [NACK] err: \"%s\" for nonce %s, last version applied on request %s",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), discoveryRequest.ErrorDetail, discoveryRequest.ResponseNonce, discoveryRequest.VersionInfo)
		// TODO: if NACK's on our latest nonce, we can also update lastAppliedVersion
		// TODO: if the NACK's nonce is our latest nonce, we should retry to avoid leaving the envoy in a wrong config state and update
		// last applied version to this requests one's, as it tells us what version is the proxy using.
		return false
	}

	// Handle first request on stream case, should always reply to empty nonce
	requestNonce = discoveryRequest.ResponseNonce
	if requestNonce == "" {
		log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: Empty nonce for %s, should be first message on stream (req resources: %v)",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), typeURL.Short(), discoveryRequest.ResourceNames)
		return true
	}

	// Handle reconnection case for a proxy that already has some configuration applied
	lastNonce = proxy.GetLastSentNonce(typeURL)
	if lastNonce == "" {
		// This is a new connection case, where the incoming nonce is not empty. This is the case of a proxy that lost connection to its control plane
		// and connected back to a control plane.
		// Version applied is going to be X, we will set our version to be also X, and trigger a response. This will make
		// this control plane to generate version X+1 for this proxy, thus keeping linearity between versions even if the proxy
		// moves around different control planes, and updating the resources to the SotW of this control plane.
		log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: Request type %s nonce %s for a proxy we didn't yet issue a nonce for. Updating version to %d.",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), typeURL.Short(), requestNonce, requestVersion)
		proxy.SetLastSentVersion(typeURL, requestVersion)
		proxy.SetLastAppliedVersion(typeURL, requestVersion)
		return true
	}

	// Handle regular proto (nonce based) from now on
	// As per protocol, we can ignore any request on the TypeURL stream that has not caught up with last sent nonce.
	if requestNonce != lastNonce {
		log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: Ignoring request for %s non-latest nonce (request: %s, current: %s)",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), typeURL.Short(), requestNonce, lastNonce)
		return false
	}

	// Nonces match
	// At this point, there is no error and nonces match, it is guaranteed an ACK with last sent version.
	proxy.SetLastAppliedVersion(typeURL, requestVersion)

	// ----
	// What's left is to check if the resources listed are the same. If they are not, we must respond
	// with the new resources requested.
	//
	// In case of LDS and CDS, "Envoy will always use wildcard mode for Listener and Cluster resources".
	// The following logic is not needed (though correct) for LDS and CDS as request resources are also empty in ACK case.
	//
	// This part of the code was inspired by Istio's `shouldRespond` handling of request resource difference
	// https://github.com/istio/istio/blob/da6178604559bdf2c707a57f452d16bee0de90c8/pilot/pkg/xds/ads.go#L347
	// ----
	resourcesLastSent := proxy.GetLastResourcesSent(typeURL)
	resourcesRequested := getRequestedResourceNamesSet(discoveryRequest)

	// If what we last sent is a superset of what the
	// requests resources subscribes to, it's ACK and nothing needs to be done.
	// Otherwise, envoy might be asking us for additional resources that have to be sent along last time.
	// Difference returns elemenets of <requested> that are not part of elements of <last sent>

	requestedResourcesDifference := resourcesRequested.Difference(resourcesLastSent)
	if requestedResourcesDifference.Cardinality() != 0 {
		log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: request difference in v:%d - requested: %v lastSent: %v (diff: %v), triggering update",
			proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), requestVersion, resourcesRequested, resourcesLastSent, requestedResourcesDifference)
		return true
	}

	log.Debug().Msgf("Proxy SerialNumber=%s PodUID=%s: ACK received for %s, version: %d nonce: %s resources ACKd: %v",
		proxy.GetCertificateSerialNumber(), proxy.GetPodUID(), typeURL.Short(), requestVersion, requestNonce, resourcesRequested)
	return false
}

// Helper to turn the resource names on a discovery request to a Set for later efficient intersection
func getRequestedResourceNamesSet(discoveryRequest *xds_discovery.DiscoveryRequest) mapset.Set {
	resourcesRequested := mapset.NewSet()
	for idx := range discoveryRequest.ResourceNames {
		resourcesRequested.Add(discoveryRequest.ResourceNames[idx])
	}
	return resourcesRequested
}

// isCNforProxy returns true if the given CN for the workload certificate matches the given proxy's identity.
// Proxy identity corresponds to the k8s service account, while the workload certificate is of the form
// <svc-account>.<namespace>.<trust-domain>.
func isCNforProxy(proxy *envoy.Proxy, cn certificate.CommonName) bool {
	proxyIdentity, err := envoy.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
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

// recordPodMetadata records pod metadata and verifies the certificate issued for this pod
// is for the same service account as seen on the pod's service account
func (s *Server) recordPodMetadata(p *envoy.Proxy) error {
	if p.Kind() == envoy.KindGateway {
		log.Debug().Msgf("Proxy with serial no %s is a gateway, skipping recording pod metadata", p.GetCertificateSerialNumber())
		return nil
	}

	pod, err := envoy.GetPodFromCertificate(p.GetCertificateCommonName(), s.kubecontroller)
	if err != nil {
		log.Warn().Msgf("Could not find pod for connecting proxy %s. No metadata was recorded.", p.GetCertificateSerialNumber())
		return nil
	}

	workloadKind := ""
	workloadName := ""
	for _, ref := range pod.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			workloadKind = ref.Kind
			workloadName = ref.Name
			break
		}
	}

	p.PodMetadata = &envoy.PodMetadata{
		UID:       string(pod.UID),
		Name:      pod.Name,
		Namespace: pod.Namespace,
		ServiceAccount: identity.K8sServiceAccount{
			Namespace: pod.Namespace,
			Name:      pod.Spec.ServiceAccountName,
		},
		WorkloadKind: workloadKind,
		WorkloadName: workloadName,
	}

	// Verify Service account matches (cert to pod Service Account)
	cn := p.GetCertificateCommonName()
	certSA, err := envoy.GetServiceAccountFromProxyCertificate(cn)
	if err != nil {
		log.Err(err).Msgf("Error getting service account from XDS certificate with CommonName=%s", cn)
		return err
	}

	if certSA != p.PodMetadata.ServiceAccount {
		log.Error().Msgf("Service Account referenced in NodeID (%s) does not match Service Account in Certificate (%s). This proxy is not allowed to join the mesh.", p.PodMetadata.ServiceAccount, certSA)
		return errServiceAccountMismatch
	}

	return nil
}
