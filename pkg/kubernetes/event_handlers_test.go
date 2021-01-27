package kubernetes

import (
	"reflect"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
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
			ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
			return ns == testNamespace
		}

		It("Should add the event to the announcement channel", func() {
			podAddChannel := events.GetPubSubInstance().Subscribe(announcements.PodAdded)
			defer events.GetPubSubInstance().Unsub(podAddChannel)

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
			podObj, castOk := pubsubMsg.NewObj.(*v1.Pod)
			Expect(castOk).To(BeTrue())
			Expect(podObj.Name).To(Equal("pod-name"))
			Expect(podObj.Namespace).To(Equal(testNamespace))
		})

		It("Should not add the event to the announcement channel", func() {
			podAddChannel := events.GetPubSubInstance().Subscribe(announcements.PodAdded)
			defer events.GetPubSubInstance().Unsub(podAddChannel)

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
