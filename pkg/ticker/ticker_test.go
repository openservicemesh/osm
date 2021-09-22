package ticker

import (
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/k8s/events"
)

func teardownTicker() {
	close(rTicker.stopConfigRoutine)
	close(rTicker.stopTickerRoutine)
	rTicker = nil
}

func TestInitTicker(t *testing.T) {
	assert := assert.New(t)
	mockConfigurator := configurator.NewMockConfigurator(gomock.NewController(t))
	mockConfigurator.EXPECT().GetConfigResyncInterval().Return(time.Duration(0))

	events.Subscribe(announcements.TickerStart)
	events.Subscribe(announcements.TickerStop)

	assert.Nil(rTicker)
	ticker := InitTicker(mockConfigurator)
	assert.NotNil(ticker)
	assert.NotNil(rTicker)

	newTicker := InitTicker(mockConfigurator)
	assert.Same(ticker, newTicker)

	// clean up
	teardownTicker()
}

func TestTicker(t *testing.T) {
	assert := assert.New(t)

	broadcastEvents := events.Subscribe(announcements.ScheduleProxyBroadcast)
	defer events.Unsub(broadcastEvents)

	var counterMutex sync.Mutex
	broadcastsReceived := 0
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-broadcastEvents:
				counterMutex.Lock()
				broadcastsReceived++
				counterMutex.Unlock()
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
	events.Publish(events.PubSubMessage{
		Kind:   announcements.TickerStart,
		NewObj: time.Duration(100 * time.Millisecond),
	})

	// broadcast events should increase in the next few seconds
	assert.Eventually(func() bool {
		counterMutex.Lock()
		defer counterMutex.Unlock()
		return broadcastsReceived > 0
	}, 3*time.Second, 500*time.Millisecond)

	// Stop the ticker
	events.Publish(events.PubSubMessage{
		Kind: announcements.TickerStop,
	})

	// Should stop increasing
	assert.Eventually(func() bool {
		counterMutex.Lock()
		defer counterMutex.Unlock()
		firstRead := broadcastsReceived
		time.Sleep(1 * time.Second)
		secondRead := broadcastsReceived

		return firstRead == secondRead
	}, 6*time.Second, 2*time.Second)
}

// Test the MeshConfig event listener code for ticker
func TestTickerConfigurator(t *testing.T) {
	assert := assert.New(t)
	mockConfigurator := configurator.NewMockConfigurator(gomock.NewController(t))

	tickerStartEvents := events.Subscribe(announcements.TickerStart)
	tickerStopEvents := events.Subscribe(announcements.TickerStop)

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
		// Simulate a meshconfig change, expect the right calls if it is enabled
		mockConfigurator.EXPECT().GetConfigResyncInterval().Return(test.mockTickerDurationVal)
		events.Publish(events.PubSubMessage{
			Kind: announcements.MeshConfigUpdated,
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
