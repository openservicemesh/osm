package k8s

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/tests"
)

const (
	testProvider  = "test-provider"
	testInformer  = "test-informer"
	testNamespace = "test-namespace"
)

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
	handlers := GetKubernetesEventHandlers(testInformer, testProvider, shouldObserve, eventTypes)
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
	handlers = GetKubernetesEventHandlers(testInformer, testProvider, shouldObserve, EventTypes{})
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
	handlers := GetKubernetesEventHandlers(testInformer, testProvider, nil, EventTypes{Update: announcements.PodUpdated})
	handlers.UpdateFunc(&originalPod, &updatedPod)

	// Compare old vs new object
	msg, ok := <-updateChan
	a.True(ok)

	a.Equal(&originalPod, msg.(events.PubSubMessage).OldObj.(*corev1.Pod))
	a.Equal(&updatedPod, msg.(events.PubSubMessage).NewObj.(*corev1.Pod))
}
