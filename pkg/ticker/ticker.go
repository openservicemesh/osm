package ticker

import (
	"time"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// Any value under minimumTickerDuration will be understood as a ticker stop
	// Conversely, a value equals or above it will be understood as ticker start
	minimumTickerDuration = time.Duration(1 * time.Minute)
)

// ResyncTicker contains the stop configuration for the ticker routines
type ResyncTicker struct {
	stopTickerRoutine chan struct{}
	stopConfigRoutine chan struct{}
}

var (
	log = logger.New("ticker")
	// Local reference to global ticker
	rTicker *ResyncTicker = nil
)

// InitTicker initializes a global ticker that is configured via
// pubsub, and triggers global proxy updates also through pubsub.
// Upon this function return, the ticker is guaranteed to be started
// and ready to receive new events.
func InitTicker(c configurator.Configurator) *ResyncTicker {
	if rTicker != nil {
		return rTicker
	}

	// Start config resync ticker routine
	tickerIsReady := make(chan struct{})
	stopTicker := make(chan struct{})
	go ticker(tickerIsReady, stopTicker)
	<-tickerIsReady

	// Start config listener
	configIsReady := make(chan struct{})
	stopConfig := make(chan struct{})
	go tickerConfigListener(c, configIsReady, stopConfig)
	<-configIsReady

	rTicker = &ResyncTicker{
		stopTickerRoutine: stopTicker,
		stopConfigRoutine: stopConfig,
	}
	return rTicker
}

// Listens to configmap events and notifies ticker routine to start/stop
func tickerConfigListener(cfg configurator.Configurator, ready chan struct{}, stop <-chan struct{}) {
	// Subscribe to configuration updates
	configMapChannel := events.GetPubSubInstance().Subscribe(
		announcements.ConfigMapAdded,
		announcements.ConfigMapDeleted,
		announcements.ConfigMapUpdated)

	// Run config listener
	// Bootstrap after subscribing
	currentDuration := cfg.GetConfigResyncInterval()

	// Initial config
	if currentDuration >= minimumTickerDuration {
		events.GetPubSubInstance().Publish(events.PubSubMessage{
			AnnouncementType: announcements.TickerStart,
			NewObj:           currentDuration,
		})
	}
	close(ready)

	for {
		select {
		case <-configMapChannel:
			newResyncInterval := cfg.GetConfigResyncInterval()
			// Skip no changes from current applied conf
			if currentDuration == newResyncInterval {
				continue
			}

			// We have a change
			if newResyncInterval >= minimumTickerDuration {
				// Notify to re/start ticker
				log.Warn().Msgf("Interval %s >= %s, issuing start ticker.", newResyncInterval, minimumTickerDuration)
				events.GetPubSubInstance().Publish(events.PubSubMessage{
					AnnouncementType: announcements.TickerStart,
					NewObj:           newResyncInterval,
				})
			} else {
				// Notify to ticker to stop
				log.Warn().Msgf("Interval %s < %s, issuing ticker stop.", newResyncInterval, minimumTickerDuration)
				events.GetPubSubInstance().Publish(events.PubSubMessage{
					AnnouncementType: announcements.TickerStop,
					NewObj:           newResyncInterval,
				})
			}
			currentDuration = newResyncInterval
		case <-stop:
			return
		}
	}
}

func ticker(ready chan struct{}, stop <-chan struct{}) {
	ticker := make(<-chan time.Time)
	tickStart := events.GetPubSubInstance().Subscribe(
		announcements.TickerStart)
	tickStop := events.GetPubSubInstance().Subscribe(
		announcements.TickerStop)

	// Notify the calling function we are ready to receive events
	// Necessary as starting the ticker could loose events by the
	// caller if the caller intends to immedaitely start it
	close(ready)

	for {
		select {
		case msg := <-tickStart:
			psubMsg, ok := msg.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Could not cast to pubsub msg %v", msg)
				continue
			}

			// Cast new object to duration value
			tickerDuration, ok := psubMsg.NewObj.(time.Duration)
			if !ok {
				log.Error().Msgf("Failed to cast ticker duration %v", psubMsg)
				continue
			}

			log.Info().Msgf("Ticker Starting with duration of %s", tickerDuration)
			ticker = time.NewTicker(tickerDuration).C
		case <-tickStop:
			log.Info().Msgf("Ticker Stopping")
			ticker = make(<-chan time.Time)
		case <-ticker:
			log.Info().Msgf("Ticker requesting broadcast proxy update")
			events.GetPubSubInstance().Publish(
				events.PubSubMessage{
					AnnouncementType: announcements.ScheduleProxyBroadcast,
				},
			)
		case <-stop:
			return
		}
	}
}
