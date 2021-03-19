package ticker

import (
	"time"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("ticker")
)

// InitTicker initializes a global ticker that is configured via
// pubsub, and triggers global proxy updates also through pubsub.
// Upon this function return, the ticker is guaranteed to be started
// and ready to receive new events.
func InitTicker() {
	doneInit := make(chan struct{})
	go ticker(doneInit)
	<-doneInit
}

func ticker(ready chan struct{}) {
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
		}
	}
}
