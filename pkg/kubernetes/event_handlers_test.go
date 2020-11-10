package kubernetes

import (
	"reflect"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/announcements"
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
			ann := make(chan announcements.Announcement, 1)
			pod := tests.NewPodTestFixture(testNamespace, "pod-name")
			addEvent(testInformer, testProvider, ann, shouldObserve, "ADD")(&pod)
			Expect(len(ann)).To(Equal(1))
			<-ann
		})

		It("Should not add the event to the announcement channel", func() {
			ann := make(chan announcements.Announcement, 1)
			var pod corev1.Pod
			pod.Namespace = "not-a-monitored-namespace"
			addEvent(testInformer, testProvider, ann, shouldObserve, "ADD")(&pod)
			Expect(len(ann)).To(Equal(0))
		})
	})

	Context("create getNamespace", func() {
		It("gets the namespace name", func() {
			ns := uuid.New().String()
			pod := tests.NewPodTestFixture(ns, uuid.New().String())
			actual := getNamespace(&pod)
			Expect(actual).To(Equal(ns))
		})
	})
})
