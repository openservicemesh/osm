package osm

import (
	"context"
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
	minProxyUpdateTime = time.Minute * 5
)

type updateScheduler[T any] struct {
	proxy *models.Proxy

	configServer    ProxyUpdater[T]
	configGenerator ProxyConfigGenerator[T]

	certManager *certificate.Manager
	workqueues  *workerpool.WorkerPool
	msgBroker   *messaging.Broker

	lastUpdate  time.Time
	timer       time.Timer
	mu          sync.Mutex
	needsUpdate atomic.Bool

	retryChan chan struct{}
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
		timer:           *time.NewTimer(minProxyUpdateTime),
	}
}

func (s *updateScheduler[T]) start(ctx context.Context) {
	proxyUpdateChan, unsubUpdates := s.msgBroker.SubscribeProxyUpdates(messaging.ProxyUpdateTopic, messaging.GetPubSubTopicForProxyUUID(s.proxy.UUID.String()))
	defer unsubUpdates()

	certRotations, unsubRotations := s.certManager.SubscribeRotations(s.proxy.Identity.String())
	defer unsubRotations()

	s.tryUpdate(ctx)
	for {
		select {
		case <-s.timer.C:
			log.Debug().Str("proxy", s.proxy.String()).Msgf("haven't updated the proxy in over %s, sending a new update.", minProxyUpdateTime)
			s.tryUpdate(ctx)
		case <-proxyUpdateChan:
			log.Debug().Str("proxy", s.proxy.String()).Msg("Broadcast update received")
			s.tryUpdate(ctx)
		case <-certRotations:
			log.Debug().Str("proxy", s.proxy.String()).Msg("Certificate has been updated for proxy")
			s.tryUpdate(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *updateScheduler[T]) tryUpdate(ctx context.Context) {
	s.needsUpdate.Store(true)
	if s.mu.TryLock() {
		// Continue to push out updates while needsUpdate is true. We set it to false right before scheduling an update.
		// If an event comes in that should trigger a new update, while we're holding the lock we then set needsUpdate
		// to true, which will force a retry as soon as the update finishes.
		if !s.lastUpdate.IsZero() {
			metricsstore.DefaultMetricsStore.ProxyTimeSinceLastUpdate.WithLabelValues(s.proxy.UUID.String()).Observe(time.Since(s.lastUpdate).Seconds())
		}
		s.scheduleUpdate(ctx, s.proxy, func() {
			s.lastUpdate = time.Now()
			if !s.timer.Stop() {
				<-s.timer.C
			}
			s.timer.Reset(minProxyUpdateTime)
			time.Sleep(10 * time.Second)
			s.mu.Unlock()
			// if another update has come in since we started the last, schedule another.
			if s.needsUpdate.Load() {
				s.tryUpdate(ctx)
			}
		})
	} else {
		log.Debug().Str("proxy", s.proxy.String()).Msg("skipping update due to in process update")
	}
}

func (s *updateScheduler[T]) scheduleUpdate(ctx context.Context, proxy *models.Proxy, done func()) {
	// AddJob can block, so we call it on a goroutine
	go s.workqueues.AddJob(
		func() {
			t := time.Now()
			log.Debug().Str("proxy", proxy.String()).Msg("Starting update for proxy")

			if err := s.update(ctx, proxy); err != nil {
				log.Error().Err(err).Str("proxy", proxy.String()).Msg("Error generating resources for proxy")
			}
			log.Debug().Msgf("Update for proxy %s took took %v", proxy.String(), time.Since(t))
			done()
		})
}

func (s *updateScheduler[T]) update(ctx context.Context, proxy *models.Proxy) error {
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
	log.Debug().Str("proxy", proxy.String()).Msg("successfully updated resources for proxy")
	return nil
}
