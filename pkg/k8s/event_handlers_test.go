package k8s

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/messaging"
)

func TestGetEventHandlers(t *testing.T) {
	testCases := []struct {
		name               string
		filter             observeFilter
		expectedEventCount uint64
	}{
		{
			name:               "add/update/delete events when the events are observed",
			filter:             func(_ interface{}) bool { return true },
			expectedEventCount: 3,
		},
		{
			name:               "add/update/delete events when the events are not observed",
			filter:             func(_ interface{}) bool { return false },
			expectedEventCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			stop := make(chan struct{})
			msgBroker := messaging.NewBroker(stop)

			eventType := EventTypes{}

			eventFuncs := GetEventHandlerFuncs(tc.filter, eventType, msgBroker)
			obj := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "p1",
				},
			}
			eventFuncs.OnAdd(obj)
			eventFuncs.OnUpdate(obj, obj)
			eventFuncs.OnDelete(obj)

			// Verify that if the filter observes the event, 3 events are queued
			// Events are rate limited, wait for 1s
			a.Eventually(func() bool {
				return msgBroker.GetTotalQEventCount() == tc.expectedEventCount
			}, 1*time.Second, 10*time.Millisecond)
		})
	}
}
