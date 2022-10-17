package k8s

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/messaging"
)

func TestGetEventHandlers(t *testing.T) {
	testCases := []struct {
		name               string
		obj                interface{}
		expectedEventCount uint64
	}{
		{
			name: "add/update/delete events when the events are observed",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "p1",
				},
			},
			expectedEventCount: 3,
		},
		{
			name: "add/update/delete events when the events are not observed",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns2",
					Name:      "p2",
				},
			},
			expectedEventCount: 0,
		},
		{
			name: "add/update/delete events for a namespace are always observed",
			obj: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns2",
				},
			},
			expectedEventCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			stop := make(chan struct{})
			msgBroker := messaging.NewBroker(stop)

			c, err := NewClient("", "", msgBroker, WithKubeClient(fake.NewSimpleClientset(monitoredNS("ns1")), testMeshName))
			a.NoError(err)

			eventFuncs := c.defaultEventHandler()

			eventFuncs.OnAdd(tc.obj)
			eventFuncs.OnUpdate(tc.obj, tc.obj)
			eventFuncs.OnDelete(tc.obj)

			// Verify that if the filter observes the event, 3 events are queued
			// Events are rate limited, wait for 1s
			a.Eventually(func() bool {
				// Add 1 since the initial namespace will always trigger an OnAdd call.
				return msgBroker.GetTotalQEventCount() == tc.expectedEventCount+1
			}, 1*time.Second, 10*time.Millisecond)
		})
	}
}
