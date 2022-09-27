package osm

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/utils"
)

// ProxyConnected is called on stream open
func (cp *ControlPlane[T]) ProxyConnected(ctx context.Context, connectionID int64) error {
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	// If the current issuer is singing then we should use the SPIFFE ID instead of common name
	kind, uuid, si, certSerialNumber, err := ValidateClient(ctx, cp.certManager.GetIssuers())
	if err != nil {
		return fmt.Errorf("Could not start cannot connect proxy for stream id %d: %w", connectionID, err)
	}

	// If maxDataPlaneConnections is enabled i.e. not 0, then check that the number of Envoy connections is less than maxDataPlaneConnections
	if cp.catalog.GetMeshConfig().Spec.Sidecar.MaxDataPlaneConnections > 0 && cp.proxyRegistry.GetConnectedProxyCount() >= cp.catalog.GetMeshConfig().Spec.Sidecar.MaxDataPlaneConnections {
		metricsstore.DefaultMetricsStore.ProxyMaxConnectionsRejected.Inc()
		return errTooManyConnections
	}

	log.Trace().Msgf("Envoy with certificate SerialNumber=%s connected", certSerialNumber)
	metricsstore.DefaultMetricsStore.ProxyConnectCount.Inc()

	proxy := models.NewProxy(kind, uuid, si, utils.GetIPFromContext(ctx), connectionID)

	if err := cp.catalog.VerifyProxy(proxy); err != nil {
		return err
	}

	cp.proxyRegistry.RegisterProxy(proxy)
	go func() {
		// Register for proxy config updates broadcasted by the message broker
		proxyUpdatePubSub := cp.msgBroker.GetProxyUpdatePubSub()
		proxyUpdateChan := proxyUpdatePubSub.Sub(messaging.ProxyUpdateTopic, messaging.GetPubSubTopicForProxyUUID(proxy.UUID.String()))
		defer cp.msgBroker.Unsub(proxyUpdatePubSub, proxyUpdateChan)

		certRotations, unsubRotations := cp.certManager.SubscribeRotations(proxy.Identity.String())
		defer unsubRotations()

		// schedule one update for this proxy initially.
		cp.scheduleUpdate(ctx, proxy)
		for {
			select {
			case <-proxyUpdateChan:
				log.Debug().Str("proxy", proxy.String()).Msg("Broadcast update received")
				cp.scheduleUpdate(ctx, proxy)
			case <-certRotations:
				log.Debug().Str("proxy", proxy.String()).Msg("Certificate has been updated for proxy")
				cp.scheduleUpdate(ctx, proxy)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (cp *ControlPlane[T]) scheduleUpdate(ctx context.Context, proxy *models.Proxy) {
	var wg sync.WaitGroup
	wg.Add(1)
	cp.workqueues.AddJob(
		func() {
			t := time.Now()
			log.Debug().Str("proxy", proxy.String()).Msg("Starting update for proxy")

			if err := cp.update(ctx, proxy); err != nil {
				log.Error().Err(err).Str("proxy", proxy.String()).Msg("Error generating resources for proxy")
			}
			log.Debug().Msgf("Update for proxy %s took took %v", proxy.String(), time.Since(t))
			wg.Done()
		})
	wg.Wait()
}

func (cp *ControlPlane[T]) update(ctx context.Context, proxy *models.Proxy) error {
	resources, err := cp.configGenerator.GenerateConfig(ctx, proxy)
	if err != nil {
		return err
	}
	if err := cp.configServer.UpdateProxy(ctx, proxy, resources); err != nil {
		return err
	}
	log.Debug().Str("proxy", proxy.String()).Msg("successfully updated resources for proxy")
	return nil
}

// ProxyDisconnected is called on stream closed
func (cp *ControlPlane[T]) ProxyDisconnected(connectionID int64) {
	log.Debug().Msgf("OnStreamClosed id: %d", connectionID)
	cp.proxyRegistry.UnregisterProxy(connectionID)

	metricsstore.DefaultMetricsStore.ProxyConnectCount.Dec()
}

// ValidateClient ensures that the connected client is authorized to connect to the gRPC server.
func ValidateClient(ctx context.Context, issuers certificate.IssuerInfo) (models.ProxyKind, uuid.UUID, identity.ServiceIdentity, certificate.SerialNumber, error) {
	mtlsPeer, ok := peer.FromContext(ctx)
	if !ok {
		log.Error().Msg("[grpc][mTLS] No peer found")
		return "", uuid.UUID{}, "", "", status.Error(codes.Unauthenticated, "no peer found")
	}

	tlsAuth, ok := mtlsPeer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		log.Error().Msg("[grpc][mTLS] Unexpected peer transport credentials")
		return "", uuid.UUID{}, "", "", status.Error(codes.Unauthenticated, "unexpected peer transport credentials")
	}

	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
		log.Error().Msgf("[grpc][mTLS] Could not verify peer certificate")
		return "", uuid.UUID{}, "", "", status.Error(codes.Unauthenticated, "could not verify peer certificate")
	}

	// Check whether the subject common name is one that is allowed to connect.
	cn := tlsAuth.State.VerifiedChains[0][0].Subject.CommonName

	kind, certuuid, si, err := getCertificateCommonNameMeta(cn)
	if err != nil {
		return "", uuid.UUID{}, "", "", fmt.Errorf("error parsing certificate common name %s: %w", cn, err)
	}

	// Can only extract and use SPIFFE ID as Primary Identification of the proxy if both issuers are using it
	// otherwise we run the risk of some certs not having the URI specified.
	if issuers.Signing.SpiffeEnabled && issuers.Validating.SpiffeEnabled {
		si, err = extractSpiffeID(tlsAuth.State.VerifiedChains[0][0].URIs)
		if err != nil {
			return "", uuid.UUID{}, "", "", err
		}
	}

	certificateSerialNumber := tlsAuth.State.VerifiedChains[0][0].SerialNumber.String()
	return kind, certuuid, si, certificate.SerialNumber(certificateSerialNumber), nil
}

// extractSpiffeID parses urls and extracts the XDS proxy identity from the SPIFFE ID in the form of spiffe://trustdomain/<proxy-UUID>/<kind>/<ServiceAccount>/<Namespace>
func extractSpiffeID(uris []*url.URL) (identity.ServiceIdentity, error) {
	if len(uris) != 1 {
		return "", fmt.Errorf("error parsing SPIFFE ID, expecting exactly one SPIFFE ID got %d", len(uris))
	}
	spiffeURL := uris[0]

	// The PATH sometimes has a slash prefixed which gives wrong number when parsing
	path := strings.TrimPrefix(spiffeURL.Path, "/")
	idParts := strings.Split(path, "/")
	if len(idParts) != 4 {
		return "", fmt.Errorf("error parsing SPIFFE ID, expecting 4 parts to SPIFFE ID got %d", len(uris))
	}

	si := identity.ServiceIdentity(fmt.Sprintf("%s.%s", idParts[2], idParts[3]))
	log.Info().Str("spiffeid", spiffeURL.String()).Msg("extracted proxy identity from SPIFFE ID")
	return si, nil
}

func getCertificateCommonNameMeta(cn string) (models.ProxyKind, uuid.UUID, identity.ServiceIdentity, error) {
	// XDS cert CN is of the form <proxy-UUID>.<kind>.<proxy-identity>.<trust-domain>
	chunks := strings.SplitN(cn, constants.DomainDelimiter, 5)
	if len(chunks) < 4 {
		return "", uuid.UUID{}, "", errInvalidCertificateCN
	}
	proxyUUID, err := uuid.Parse(chunks[0])
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrParsingXDSCertCN)).
			Msgf("Error parsing %s into uuid.UUID", chunks[0])
		return "", uuid.UUID{}, "", err
	}

	switch {
	case chunks[1] == "":
		return "", uuid.UUID{}, "", errInvalidCertificateCN
	case chunks[2] == "":
		return "", uuid.UUID{}, "", errInvalidCertificateCN
	case chunks[3] == "":
		return "", uuid.UUID{}, "", errInvalidCertificateCN
	}

	return models.ProxyKind(chunks[1]), proxyUUID, identity.New(chunks[2], chunks[3]), nil
}
