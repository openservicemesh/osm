package k8s

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/tests"
)

const (
	testNamespace = "test-namespace"
)

func setup() {
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.NamespaceCount,
	)
}

func teardown() {
	metricsstore.DefaultMetricsStore.Stop(
		metricsstore.DefaultMetricsStore.NamespaceCount,
	)
}

func TestGetKubernetesEventHandlers(t *testing.T) {
	a := assert.New(t)

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return testNamespace == object.GetNamespace()
	}

	podAddChannel := events.Subscribe(announcements.PodAdded)
	defer events.Unsub(podAddChannel)

	pod := tests.NewPodFixture(testNamespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
	eventTypes := EventTypes{
		Add:    announcements.PodAdded,
		Update: announcements.PodUpdated,
		Delete: announcements.PodDeleted,
	}
	handlers := GetKubernetesEventHandlers(shouldObserve, eventTypes)
	handlers.AddFunc(&pod)
	an := <-podAddChannel
	a.Len(podAddChannel, 0)

	// Pubsub msg
	pubsubMsg, castOk := an.(events.PubSubMessage)
	a.True(castOk)
	a.Equal(pubsubMsg.Kind, announcements.PodAdded)
	a.Nil(pubsubMsg.OldObj)

	// Cast New obj, expect v1.Pod
	podObj, castOk := pubsubMsg.NewObj.(*corev1.Pod)
	a.True(castOk)
	a.Equal(podObj.Name, "pod-name")
	a.Equal(podObj.Namespace, testNamespace)

	podAddChannel = events.Subscribe(announcements.PodAdded)
	defer events.Unsub(podAddChannel)

	// should not add the event to the announcement channel
	pod.Namespace = "not-a-monitored-namespace"
	handlers = GetKubernetesEventHandlers(shouldObserve, EventTypes{})
	handlers.AddFunc(&pod)
	a.Len(podAddChannel, 0)

	ns := uuid.New().String()
	pod = tests.NewPodFixture(ns, uuid.New().String(), tests.BookstoreServiceAccountName, tests.PodLabels)
	actual := getNamespace(&pod)
	a.Equal(actual, ns)
}

func TestUpdateEvent(t *testing.T) {
	a := assert.New(t)

	updateChan := events.Subscribe(announcements.PodUpdated)
	defer events.Unsub(updateChan)

	// Add and update a pod
	originalPod := tests.NewPodFixture(testNamespace, "pod-name", tests.BookstoreServiceAccountName, tests.PodLabels)
	updatedPod := originalPod
	updatedPod.Labels = nil // updated does not have any labels

	// Invoke update handler
	handlers := GetKubernetesEventHandlers(nil, EventTypes{Update: announcements.PodUpdated})
	handlers.UpdateFunc(&originalPod, &updatedPod)

	// Compare old vs new object
	msg, ok := <-updateChan
	a.True(ok)

	a.Equal(&originalPod, msg.(events.PubSubMessage).OldObj.(*corev1.Pod))
	a.Equal(&updatedPod, msg.(events.PubSubMessage).NewObj.(*corev1.Pod))
}

func TestNamespaceUpdateEvent(t *testing.T) {
	setup()
	defer teardown()

	a := assert.New(t)

	updateChan := events.Subscribe(announcements.NamespaceUpdated)
	defer events.Unsub(updateChan)

	// Add and update a pod
	originalNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oldnamespace",
		},
	}

	updatedNamespace := originalNamespace
	updatedNamespace.Labels = map[string]string{
		constants.OSMKubeResourceMonitorAnnotation: "osm",
	} // updated has a monitored by label

	handler := metricsstore.DefaultMetricsStore.Handler()

	req, err := http.NewRequest("GET", "/metrics", nil)
	a.Nil(err)

	rr := httptest.NewRecorder()

	t.Run("AddMonitoredByLabel", func(t *testing.T) {
		a := assert.New(t)

		// Check metrics count before update
		handler.ServeHTTP(rr, req)

		a.Equal(http.StatusOK, rr.Code)

		expectedResp := fmt.Sprintf(`# HELP osm_resource_namespace_count Represents the number of Namespaces monitored by OSM
# TYPE osm_resource_namespace_count gauge
osm_resource_namespace_count %d
`, 0)
		a.Contains(rr.Body.String(), expectedResp)

		// Invoke update handler
		eventTypes := EventTypes{
			Add:    announcements.NamespaceAdded,
			Update: announcements.NamespaceUpdated,
			Delete: announcements.NamespaceDeleted,
		}
		handlers := GetKubernetesEventHandlers(nil, eventTypes)
		handlers.UpdateFunc(&originalNamespace, &updatedNamespace)

		// Compare old vs new object
		msg, ok := <-updateChan
		a.True(ok)

		a.Equal(&originalNamespace, msg.(events.PubSubMessage).OldObj.(*corev1.Namespace))
		a.Equal(&updatedNamespace, msg.(events.PubSubMessage).NewObj.(*corev1.Namespace))

		// Check metrics count after update
		handler.ServeHTTP(rr, req)

		a.Equal(http.StatusOK, rr.Code)

		expectedResp = fmt.Sprintf(`# HELP osm_resource_namespace_count Represents the number of Namespaces monitored by OSM
# TYPE osm_resource_namespace_count gauge
osm_resource_namespace_count %d
`, 1)
		a.Contains(rr.Body.String(), expectedResp)
	})

	t.Run("RemoveMonitoredByLabel", func(t *testing.T) {
		a := assert.New(t)

		// Invoke update handler
		eventTypes := EventTypes{
			Add:    announcements.NamespaceAdded,
			Update: announcements.NamespaceUpdated,
			Delete: announcements.NamespaceDeleted,
		}
		handlers := GetKubernetesEventHandlers(nil, eventTypes)
		handlers.UpdateFunc(&updatedNamespace, &originalNamespace)

		// Compare old vs new object
		msg, ok := <-updateChan
		a.True(ok)

		a.Equal(&updatedNamespace, msg.(events.PubSubMessage).OldObj.(*corev1.Namespace))
		a.Equal(&originalNamespace, msg.(events.PubSubMessage).NewObj.(*corev1.Namespace))

		// Check metrics count after update
		handler.ServeHTTP(rr, req)

		a.Equal(http.StatusOK, rr.Code)

		expectedResp := fmt.Sprintf(`# HELP osm_resource_namespace_count Represents the number of Namespaces monitored by OSM
# TYPE osm_resource_namespace_count gauge
osm_resource_namespace_count %d
`, 0)
		a.Contains(rr.Body.String(), expectedResp)
	})
}
