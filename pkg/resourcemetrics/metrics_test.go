package resourcemetrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func setup() {
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter,
	)
}

func teardown() {
	metricsstore.DefaultMetricsStore.Stop(
		metricsstore.DefaultMetricsStore.MonitoredNamespaceCounter,
	)
}

func TestNamespaceUpdateEvent(t *testing.T) {
	setup()
	defer teardown()

	a := assert.New(t)

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace",
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: "osm",
			},
		},
	}

	namespace2 := namespace
	namespace2.Labels = map[string]string{
		"testlabel": "testvalue",
	}

	testCases := []struct {
		name                   string
		event                  events.PubSubMessage
		expectedNamespaceCount string
	}{
		{
			name: "namespace added event",
			event: events.PubSubMessage{
				Kind:   announcements.NamespaceAdded,
				OldObj: nil,
				NewObj: &namespace,
			},
			expectedNamespaceCount: "1",
		},
		{
			name: "namespace updated event",
			event: events.PubSubMessage{
				Kind:   announcements.NamespaceUpdated,
				OldObj: &namespace,
				NewObj: &namespace2,
			},
			expectedNamespaceCount: "1",
		},
		{
			name: "namespace deleted event",
			event: events.PubSubMessage{
				Kind:   announcements.NamespaceDeleted,
				OldObj: &namespace2,
				NewObj: nil,
			},
			expectedNamespaceCount: "0",
		},
	}

	stop := make(chan struct{})
	defer close(stop)
	msgBroker := messaging.NewBroker(stop)

	go StartNamespaceCounter(msgBroker, stop)
	// Subscription should happen before an event is published by the test, so
	// add a delay before the test triggers events
	time.Sleep(500 * time.Millisecond)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msgBroker.GetKubeEventPubSub().Pub(tc.event, tc.event.Kind.String())

			// Add a delay before making a request to allow time for the msg
			// to be processed and the metric updated
			time.Sleep(500 * time.Millisecond)

			handler := metricsstore.DefaultMetricsStore.Handler()

			req, err := http.NewRequest("GET", "/metrics", nil)
			a.Nil(err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			a.Equal(http.StatusOK, rr.Code)

			expectedResp := fmt.Sprintf(`# HELP osm_resource_namespace_count Represents the number of monitored namespaces in the service mesh
# TYPE osm_resource_namespace_count gauge
osm_resource_namespace_count{namespace="namespace"} %s
`, tc.expectedNamespaceCount /* monitored namespace count */)
			a.Contains(rr.Body.String(), expectedResp)
		})
	}
}
