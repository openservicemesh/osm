package osm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/utils"
)

const (
	// Schedule a proxy every 5 minutes at minimum.
	minProxyUpdateTime = time.Minute * 5
)

// ProxyConnected is called on stream open
func (cp *ControlPlane[T]) ProxyConnected(ctx context.Context, connectionID int64) error {
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	certCommonName, certSerialNumber, err := utils.ValidateClient(ctx)
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

	kind, uuid, si, err := getCertificateCommonNameMeta(certCommonName)
	if err != nil {
		return fmt.Errorf("error parsing certificate common name %s: %w", certCommonName, err)
	}

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

		var mu sync.Mutex
		mu.Lock()
		// schedule one update for this proxy initially.
		cp.scheduleUpdate(ctx, proxy, mu.Unlock)
		// Needs to be of size one since we add to it on the same routine we listen on.
		updateChan := make(chan any, 1)
		timer := time.NewTimer(minProxyUpdateTime)
		var needsUpdate atomic.Bool
		for {
			select {
			case <-timer.C:
				log.Debug().Str("proxy", proxy.String()).Msgf("haven't updated the proxy in over %s, sending a new update.", minProxyUpdateTime)
				updateChan <- struct{}{}
			case <-proxyUpdateChan:
				log.Debug().Str("proxy", proxy.String()).Msg("Broadcast update received")
				updateChan <- struct{}{}
			case <-certRotations:
				log.Debug().Str("proxy", proxy.String()).Msg("Certificate has been updated for proxy")
				updateChan <- struct{}{}
			case <-updateChan:
				// This attempts to grab the lock. If it can't, that means that another update is processing, but that prior
				// update may have missed some new resources that determine that proxy config state. So we set needsUpdate to
				// true. It's possible that it didn't, but and the next update is redundant, but it's the safe choice.
				// If it does grab the lock, it schedules an update with a close func, that resets the timer, schedules a new
				// update if any new data may have come in during the prior update, and sets shouldUpdate to false.
				if mu.TryLock() {
					cp.scheduleUpdate(ctx, proxy, func() {
						// Update to false while holding this lock, to not erase a reset to true.
						// It could later incorrectly get over written to true before we release the lock, which would schedule a
						// second, and potentially redundant update, but this is better than getting incorrectly overwritten to false.
						shouldUpdate := needsUpdate.Swap(false)
						if !timer.Stop() {
							<-timer.C
						}
						timer.Reset(minProxyUpdateTime)
						mu.Unlock()
						if shouldUpdate {
							updateChan <- struct{}{}
						}
					})
				} else {
					needsUpdate.Store(true)
					log.Debug().Str("proxy", proxy.String()).Msg("skipping update due to in process update")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (cp *ControlPlane[T]) scheduleUpdate(ctx context.Context, proxy *models.Proxy, done func()) {
	// AddJob can block, so we call it on a goroutine
	go cp.workqueues.AddJob(
		func() {
			t := time.Now()
			log.Debug().Str("proxy", proxy.String()).Msg("Starting update for proxy")

			if err := cp.update(ctx, proxy); err != nil {
				log.Error().Err(err).Str("proxy", proxy.String()).Msg("Error generating resources for proxy")
			}
			log.Debug().Msgf("Update for proxy %s took took %v", proxy.String(), time.Since(t))
			done()
		})
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

func getCertificateCommonNameMeta(cn certificate.CommonName) (models.ProxyKind, uuid.UUID, identity.ServiceIdentity, error) {
	// XDS cert CN is of the form <proxy-UUID>.<kind>.<proxy-identity>.<trust-domain>
	chunks := strings.SplitN(cn.String(), constants.DomainDelimiter, 5)
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
