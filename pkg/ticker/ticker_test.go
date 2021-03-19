package ticker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

func TestTicker(t *testing.T) {
	assert := assert.New(t)

	broadcastEvents := events.GetPubSubInstance().Subscribe(announcements.ScheduleProxyBroadcast)
	broadcastsReceived := 0
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-broadcastEvents:
				broadcastsReceived++
			case <-stop:
				return
			}
		}
	}()

	// Start the ticker routine
	InitTicker()

	// Start ticker, tick at 100ms rate
	events.GetPubSubInstance().Publish(events.PubSubMessage{
		AnnouncementType: announcements.TickerStart,
		NewObj:           time.Duration(100 * time.Millisecond),
	})

	// broadcast events should increase in the next few seconds
	assert.Eventually(func() bool {
		return broadcastsReceived > 0
	}, 3*time.Second, 500*time.Millisecond)

	// Stop the ticker
	events.GetPubSubInstance().Publish(events.PubSubMessage{
		AnnouncementType: announcements.TickerStop,
	})

	// Should stop increasing
	assert.Eventually(func() bool {
		firstRead := broadcastsReceived
		time.Sleep(1 * time.Second)
		secondRead := broadcastsReceived

		return firstRead == secondRead
	}, 6*time.Second, 2*time.Second)

	close(stop)
}
