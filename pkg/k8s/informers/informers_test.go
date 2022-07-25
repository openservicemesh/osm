package informers

import (
	"context"
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
		expectedEventCount uint64
	}{
		{
			name:               "add/update/delete events when the events are observed",
			expectedEventCount: 3,
		},
		{
			name:               "add/update/delete events when the events are not observed",
			expectedEventCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			ctx := context.Background()
			stop := make(chan struct{})
			msgBroker := messaging.NewBroker(stop)

			fakeK8sClient := fake.NewSimpleClientset()

			_, err := NewInformerCollection("test", msgBroker, stop, WithKubeClient(fakeK8sClient))
			a.NoError(err)

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "p1",
				},
			}
			_, err = fakeK8sClient.CoreV1().Pods("ns1").Create(ctx, pod, metav1.CreateOptions{})
			a.NoError(err)

			// get the pod so we can update it.
			pod, err = fakeK8sClient.CoreV1().Pods("ns1").Get(ctx, "p1", metav1.GetOptions{})
			a.NoError(err)

			_, err = fakeK8sClient.CoreV1().Pods("ns1").Update(ctx, pod, metav1.UpdateOptions{})
			a.NoError(err)

			err = fakeK8sClient.CoreV1().Pods("ns1").Delete(ctx, "p1", metav1.DeleteOptions{})
			a.NoError(err)

			// Verify that if the filter observes the event, 3 events are queued
			// Events are rate limited, wait for 1s
			a.Eventually(func() bool {
				return msgBroker.GetTotalQEventCount() == tc.expectedEventCount
			}, 1*time.Second, 10*time.Millisecond)
		})
	}
}
