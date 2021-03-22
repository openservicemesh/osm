package ticker

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

func TestTicker(t *testing.T) {
	assert := assert.New(t)

	broadcastEvents := events.GetPubSubInstance().Subscribe(announcements.ScheduleProxyBroadcast)
	defer events.GetPubSubInstance().Unsub(broadcastEvents)

	broadcastsReceived := 0
	stop := make(chan struct{})
	defer close(stop)
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
	doneInit := make(chan struct{})
	stopTicker := make(chan struct{})
	defer close(stopTicker)
	go ticker(doneInit, stop)
	<-doneInit

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
}

// Test the ConfigMap event listener code for ticker
func TestTickerConfigurator(t *testing.T) {
	assert := assert.New(t)
	mockConfigurator := configurator.NewMockConfigurator(gomock.NewController(t))

	tickerStartEvents := events.GetPubSubInstance().Subscribe(announcements.TickerStart)
	tickerStopEvents := events.GetPubSubInstance().Subscribe(announcements.TickerStop)

	// First init will expect defaults to false
	mockConfigurator.EXPECT().GetConfigResyncInterval().Return(time.Duration(0))

	doneInit := make(chan struct{})
	stopConfig := make(chan struct{})
	defer close(stopConfig)
	go tickerConfigListener(mockConfigurator, doneInit, stopConfig)
	<-doneInit

	type tickerConfigTests struct {
		mockTickerDurationVal time.Duration
		expectStartEvent      int
		expectStopEvent       int
	}

	tickerConfTests := []tickerConfigTests{
		{time.Duration(2 * time.Minute), 1, 0},  // default (off) -> 2m, expect start
		{time.Duration(2 * time.Minute), 0, 0},  // No change, expect no event
		{time.Duration(3 * time.Minute), 1, 0},  // 2m -> enabled 3m, expect start
		{time.Duration(0), 0, 1},                // 2m -> stop, expect stop
		{time.Duration(30 * time.Second), 0, 1}, // stop -> still smaller than threshold, expect stop
		{time.Duration(0), 0, 1},                // stopped -> stopped, still trigger change
		{time.Duration(2 * time.Minute), 1, 0},  // stopped -> start, expect start
	}

	for _, test := range tickerConfTests {
		// Simulate a configmap change, expect the right calls if it is enabled
		mockConfigurator.EXPECT().GetConfigResyncInterval().Return(test.mockTickerDurationVal)
		events.GetPubSubInstance().Publish(events.PubSubMessage{
			AnnouncementType: announcements.ConfigMapUpdated,
		})

		receivedStartEvent := 0
		receivedStopEvent := 0
		done := false
		for !done {
			select {
			case <-tickerStartEvents:
				receivedStartEvent++
			case <-tickerStopEvents:
				receivedStopEvent++
			// 500mili should be plenty for this
			case <-time.After(500 * time.Millisecond):
				done = true
			}
		}

		assert.Equal(test.expectStartEvent, receivedStartEvent)
		assert.Equal(test.expectStopEvent, receivedStopEvent)
	}
}
