package main

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rs/zerolog"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/k8s/events"
)

func TestGlobalLogLevelHandler(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	stop := make(chan struct{})
	defer close(stop)

	mockConfigurator.EXPECT().GetOSMLogLevel().Return("trace").Times(1)
	StartGlobalLogLevelHandler(mockConfigurator, stop)

	// Set log level through a meshconfig event
	mockConfigurator.EXPECT().GetOSMLogLevel().Return("warn").Times(1)
	events.Publish(events.PubSubMessage{
		Kind: announcements.MeshConfigUpdated,
	})

	assert.Eventually(func() bool {
		return zerolog.GlobalLevel() == zerolog.WarnLevel
	}, 2*time.Second, 25*time.Millisecond, "Global log level did not change in specified time")

	// Reset back
	mockConfigurator.EXPECT().GetOSMLogLevel().Return("trace").Times(1)
	events.Publish(events.PubSubMessage{
		Kind: announcements.MeshConfigUpdated,
	})

	assert.Eventually(func() bool {
		return zerolog.GlobalLevel() == zerolog.TraceLevel
	}, 2*time.Second, 25*time.Millisecond, "Global log level did not reset to trace")
}
