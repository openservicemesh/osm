package ticker

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

func TestResyncTicker(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	msgBroker := messaging.NewBroker(stop)

	minTickerInterval := 100 * time.Millisecond
	r := NewResyncTicker(msgBroker, minTickerInterval)
	// Start the ResyncTicker
	r.Start(stop)
	// Give enough time for Ticker to start and subscribe to MeshConfig updates
	time.Sleep(500 * time.Millisecond)

	type test struct {
		name                 string
		event                events.PubSubMessage
		waitUntil            time.Duration
		minExpectedTicks     uint64
		expectedInvalidCount int
	}

	testCases := []test{
		{
			name: "Start ticker that ticks every 1s",
			event: events.PubSubMessage{
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "",
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "1s",
						},
					},
				},
				Kind: events.MeshConfig,
				Type: events.Updated,
			},
			waitUntil:        6 * time.Second,
			minExpectedTicks: 5,
		},
		{
			name: "Update ticker from 1s to 500ms",
			event: events.PubSubMessage{
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "1s",
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "500ms",
						},
					},
				},
				Kind: events.MeshConfig,
				Type: events.Updated,
			},
			waitUntil:        6 * time.Second,
			minExpectedTicks: 10,
		},
		{
			name: "Stop ticker - 500ms to 0",
			event: events.PubSubMessage{
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "500",
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "0",
						},
					},
				},
				Kind: events.MeshConfig,
				Type: events.Updated,
			},
			waitUntil:        2 * time.Second,
			minExpectedTicks: 0,
		},
		{
			name: "Restart ticker from 0 to 500ms",
			event: events.PubSubMessage{
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "0",
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "500ms",
						},
					},
				},
				Kind: events.MeshConfig,
				Type: events.Updated,
			},
			waitUntil:        3 * time.Second,
			minExpectedTicks: 4,
		},
		{
			name: "Ticker continues to operate when the tick value is unchanged",
			event: events.PubSubMessage{
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "500ms",
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "500ms",
						},
					},
				},
				Kind: events.MeshConfig,
				Type: events.Updated,
			},
			waitUntil:        3 * time.Second,
			minExpectedTicks: 4,
		},
		{
			name: "Set ticker interval below min allowed and verify it is ignored",
			event: events.PubSubMessage{
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "0",
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "1ms", // Less than 'minTickerInterval'
						},
					},
				},
				Kind: events.MeshConfig,
				Type: events.Updated,
			},
			waitUntil:            1 * time.Second,
			minExpectedTicks:     0,
			expectedInvalidCount: 1,
		},
		{
			name: "Restart ticker from invalid interval to 500ms",
			event: events.PubSubMessage{
				OldObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "1ms",
						},
					},
				},
				NewObj: &configv1alpha2.MeshConfig{
					Spec: configv1alpha2.MeshConfigSpec{
						Sidecar: configv1alpha2.SidecarSpec{
							ConfigResyncInterval: "500ms",
						},
					},
				},
				Kind: events.MeshConfig,
				Type: events.Updated,
			},
			waitUntil:            3 * time.Second,
			minExpectedTicks:     4,
			expectedInvalidCount: 1, // From previous test case
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			done := false

			msgBroker.PublishKubeEvent(tc.event)
			timeout := time.After(tc.waitUntil)
			for !done {
				select {
				case <-timeout:
					done = true
					log.Debug().Msg("Done!")
				default:
					// Process next select statement
				}
			}

			a.GreaterOrEqual(msgBroker.GetTotalQProxyEventCount(), tc.minExpectedTicks)
			a.EqualValues(tc.expectedInvalidCount, atomic.LoadUint64(&r.invalidIntervalCounter))
		})
	}
}
