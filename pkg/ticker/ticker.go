package ticker

import (
	"sync/atomic"
	"time"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// ResyncTicker is the type that implements a ticker to trigger internal system resyncs
// periodicially based on the configuration specified in the MeshConfig resource.
type ResyncTicker struct {
	stopTickerCh chan struct{}
	msgBroker    *messaging.Broker
	running      bool
	// minTickerDuration is the minimum duration that can be used to configure
	// the ticker. Any value under minTickerDuration will be ignored with a warn
	// log, while a value of 0 indicates that the ticker must be stopped.
	minTickInterval time.Duration

	invalidIntervalCounter uint64
}

var (
	log = logger.New("ticker")
)

// NewResyncTicker returns a ResyncTicker instance that is used to periodically
// trigger proxy config resyncs.
func NewResyncTicker(msgBroker *messaging.Broker, minTickInterval time.Duration) *ResyncTicker {
	return &ResyncTicker{
		stopTickerCh:    make(chan struct{}),
		msgBroker:       msgBroker,
		minTickInterval: minTickInterval,
	}
}

// Start starts the ResyncTicker's configuration watcher in a goroutine which runs
// until the given channel is closed.
func (r *ResyncTicker) Start(quit <-chan struct{}) {
	go r.watchConfig(quit)
}

// watchConfig watches for new ticker configuration and starts/stops/resets ticker
// based on the configuration.
func (r *ResyncTicker) watchConfig(quit <-chan struct{}) {
	// Subscribe to MeshConfig updates through which Ticker can be turned on/off
	kubePubSub := r.msgBroker.GetKubeEventPubSub()
	meshConfigUpdateChan := kubePubSub.Sub(announcements.MeshConfigUpdated.String())
	defer r.msgBroker.Unsub(kubePubSub, meshConfigUpdateChan)

	for {
		select {
		case msg, ok := <-meshConfigUpdateChan:
			if !ok {
				log.Warn().Msgf("Notification channel closed for MeshConfig")
				continue
			}

			event, ok := msg.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Received unexpected message %T on channel, expected PubSubMessage", event)
				continue
			}

			oldMeshSpec, oldOk := event.OldObj.(*configv1alpha3.MeshConfig)
			newMeshSpec, newOk := event.NewObj.(*configv1alpha3.MeshConfig)
			if !oldOk || !newOk {
				log.Error().Msgf("Received unexpected message old=%T new=%T on channel, expected *MeshConfig", oldMeshSpec, newMeshSpec)
				continue
			}

			if oldMeshSpec.Spec.Sidecar.ConfigResyncInterval == newMeshSpec.Spec.Sidecar.ConfigResyncInterval {
				// No change in Ticker configuration
				continue
			}

			if newMeshSpec.Spec.Sidecar.ConfigResyncInterval == "" {
				// No resync configured
				continue
			}

			newResyncInterval, err := time.ParseDuration(newMeshSpec.Spec.Sidecar.ConfigResyncInterval)
			if err != nil {
				log.Error().Err(err).Msg("Error parsing config new resync interval")
				continue
			}

			// We have a change
			if newResyncInterval >= r.minTickInterval {
				log.Debug().Msgf("Updating resync ticker to tick every %v", newResyncInterval)
				// The check to only stop ticker when it is running is required to avoid
				// writing to a blocked channel when the ticker routine is not running, which
				// would happen when 'stopTicker()' is invoked.
				// For e.g., when the ticker routine was not started previously, we must not try to
				// stop it.
				if r.running {
					r.stopTicker()
				}
				go r.startTicker(newResyncInterval)
			} else if newResyncInterval == 0 {
				// Notify to ticker to stop
				log.Warn().Msg("Resync interval set to 0, stopping ticker")
				r.stopTicker()
			} else {
				log.Warn().Msgf("New resync interval is less than min allowed interval %v, ticker will not be updated", r.minTickInterval)
				atomic.AddUint64(&r.invalidIntervalCounter, 1)
			}

		case <-quit:
			r.stopTicker()
			return
		}
	}
}

// stopTicker stops the ticker routine
func (r *ResyncTicker) stopTicker() {
	r.stopTickerCh <- struct{}{}
	r.running = false
}

// startTicker runs the ticker routine and ticks periodically at the given interval.
// It stops when 'stopTicker()' is invoked.
func (r *ResyncTicker) startTicker(tickIterval time.Duration) {
	ticker := time.NewTicker(time.Duration(tickIterval))
	r.running = true

	for {
		select {
		case <-r.stopTickerCh:
			log.Info().Msgf("Received signal to stop ticker, exiting ticker routine")
			return

		case <-ticker.C:
			r.msgBroker.GetQueue().AddRateLimited(events.PubSubMessage{
				Kind: announcements.ProxyUpdate,
			})
			log.Trace().Msg("Ticking, queued internal event")
		}
	}
}
