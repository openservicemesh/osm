package kubernetes

import (
	"reflect"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openservicemesh/osm/pkg/tests"
	corev1 "k8s.io/api/core/v1"
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
			announcements := make(chan interface{}, 1)
			pod := tests.NewPodTestFixture(testNamespace, "pod-name")
			addEvent(testInformer, testProvider, announcements, shouldObserve, "ADD")(&pod)
			Expect(len(announcements)).To(Equal(1))
			<-announcements
		})

		It("Should not add the event to the announcement channel", func() {
			announcements := make(chan interface{}, 1)
			var pod corev1.Pod
			pod.Namespace = "not-a-monitored-namespace"
			addEvent(testInformer, testProvider, announcements, shouldObserve, "ADD")(&pod)
			Expect(len(announcements)).To(Equal(0))
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
