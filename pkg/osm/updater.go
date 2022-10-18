package osm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/workerpool"
)

const (
	// Schedule a proxy every 5 minutes at minimum.
	minProxyUpdateFrequency = time.Minute * 5
	// the maximum frequency we will update a proxy
	maxProxyUpdateFrequency = time.Second * 10
)

type updateScheduler[T any] struct {
	proxy *models.Proxy

	certManager *certificate.Manager
	workqueues  *workerpool.WorkerPool
	msgBroker   *messaging.Broker

	lastUpdate      atomic.Time
	needsUpdate     atomic.Bool
	mu              sync.Mutex
	configServer    ProxyUpdater[T]
	configGenerator ProxyConfigGenerator[T]
}

func newUpdateScheduler[T any](proxy *models.Proxy, configServer ProxyUpdater[T],
	configGenerator ProxyConfigGenerator[T],
	certManager *certificate.Manager,
	workqueues *workerpool.WorkerPool,
	msgBroker *messaging.Broker) *updateScheduler[T] {
	return &updateScheduler[T]{
		proxy:           proxy,
		configServer:    configServer,
		configGenerator: configGenerator,
		certManager:     certManager,
		workqueues:      workqueues,
		msgBroker:       msgBroker,
	}
}

func (s *updateScheduler[T]) start(ctx context.Context) {
	proxyUpdateChan, unsubUpdates := s.msgBroker.SubscribeProxyUpdates(messaging.ProxyUpdateTopic, messaging.GetPubSubTopicForProxyUUID(s.proxy.UUID.String()))
	defer unsubUpdates()

	certRotations, unsubRotations := s.certManager.SubscribeRotations(s.proxy.Identity.String())
	defer unsubRotations()

	timer := time.NewTimer(minProxyUpdateFrequency)
	defer func() {
		if !timer.Stop() {
			<-timer.C
		}
	}()

	for {
		lastUpdate := s.lastUpdate.Load()
		if !lastUpdate.IsZero() && time.Since(lastUpdate) < maxProxyUpdateFrequency {
			s.needsUpdate.Store(true)
		} else {
			s.scheduleUpdate(ctx)
		}
		select {
		case <-timer.C:
			log.Trace().Str("proxy", s.proxy.String()).Msgf("haven't updated the proxy in over %s, sending a new update.", minProxyUpdateFrequency)
		case <-proxyUpdateChan:
			log.Trace().Str("proxy", s.proxy.String()).Msg("Broadcast update received")
		case <-certRotations:
			log.Trace().Str("proxy", s.proxy.String()).Msg("Certificate has been updated for proxy")
		case <-ctx.Done():
			log.Info().Str("proxy", s.proxy.String()).Msg("Stopping proxy update scheduler")
			return
		}
	}
}

func (s *updateScheduler[T]) scheduleUpdate(ctx context.Context) {
	// Try to grab the lock, if we can't another update is completing, but we set needsUpdate to true, which will ensure
	// another update gets processed when the next update finishes, once it signals on s.updated.
	if s.mu.TryLock() {
		// AddJob can block, so we call it on a goroutine
		go s.workqueues.AddJob(
			func() {
				log.Debug().Str("proxy", s.proxy.String()).Msg("Starting update for proxy")

				t := time.Now()
				if err := s.update(ctx, s.proxy); err != nil {
					log.Error().Err(err).Str("proxy", s.proxy.String()).Msg("Error generating resources for proxy")
				}
				log.Debug().Msgf("Update for proxy %s took took %v", s.proxy.String(), time.Since(t))

				s.mu.Unlock()
				// If another update has come in since we started the last, schedule another.
				// It's possible another update has already been scheduled, but it's better to schedule an extra than miss it.
				if s.needsUpdate.Load() {
					time.AfterFunc(maxProxyUpdateFrequency, func() { s.scheduleUpdate(ctx) })
				}
			})
	} else {
		log.Trace().Str("proxy", s.proxy.String()).Msg("skipping update due to in process update, or window too small")
	}
}

func (s *updateScheduler[T]) update(ctx context.Context, proxy *models.Proxy) error {
	// because an update can get scheduled while the proxy disconnects later, we check if the context has been cancelled.
	// This should also be done in the generate config/update proxy, but we add a check here for redundancy.
	if ctx.Err() != nil {
		return fmt.Errorf("not updating proxy %s due to context closed: %w", proxy, ctx.Err())
	}

	lastUpdate := s.lastUpdate.Load()
	if !lastUpdate.IsZero() {
		metricsstore.DefaultMetricsStore.ProxyTimeSinceLastUpdate.WithLabelValues(s.proxy.UUID.String()).Observe(time.Since(lastUpdate).Seconds())
	}
	// Set needsUpdate to false right before updating. If an event came in before this, we'll catch all the updates we
	// need now.
	s.needsUpdate.Store(false)
	resources, err := s.configGenerator.GenerateConfig(ctx, proxy)
	if err != nil {
		return err
	}
	if err := s.configServer.UpdateProxy(ctx, proxy, resources); err != nil {
		return err
	}
	s.lastUpdate.Store(time.Now())
	log.Debug().Str("proxy", proxy.String()).Msg("successfully updated resources for proxy")
	return nil
}
