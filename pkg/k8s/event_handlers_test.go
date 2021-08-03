package k8s

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

var _ = Describe("Testing event handlers", func() {
	Context("Test add on a monitored namespace", func() {
		shouldObserve := func(obj interface{}) bool {
			object, ok := obj.(metav1.Object)
			if !ok {
				return false
			}
			return testNamespace == object.GetNamespace()
		}

		It("Should add the event to the announcement channel", func() {
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
			Expect(len(podAddChannel)).To(Equal(0))

			// Pubsub msg
			pubsubMsg, castOk := an.(events.PubSubMessage)
			Expect(castOk).To(BeTrue())
			Expect(pubsubMsg.AnnouncementType).To(Equal(announcements.PodAdded))
			Expect(pubsubMsg.OldObj).To(BeNil())

			// Cast New obj, expect v1.Pod
			podObj, castOk := pubsubMsg.NewObj.(*corev1.Pod)
			Expect(castOk).To(BeTrue())
			Expect(podObj.Name).To(Equal("pod-name"))
			Expect(podObj.Namespace).To(Equal(testNamespace))
		})

		It("Should not add the event to the announcement channel", func() {
			podAddChannel := events.Subscribe(announcements.PodAdded)
			defer events.Unsub(podAddChannel)

			var pod corev1.Pod
			pod.Namespace = "not-a-monitored-namespace"
			handlers := GetKubernetesEventHandlers(testInformer, testProvider, shouldObserve, EventTypes{})
			handlers.AddFunc(&pod)
			Expect(len(podAddChannel)).To(Equal(0))
		})
	})

	Context("create getNamespace", func() {
		It("gets the namespace name", func() {
			ns := uuid.New().String()
			pod := tests.NewPodFixture(ns, uuid.New().String(), tests.BookstoreServiceAccountName, tests.PodLabels)
			actual := getNamespace(&pod)
			Expect(actual).To(Equal(ns))
		})
	})
})

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
