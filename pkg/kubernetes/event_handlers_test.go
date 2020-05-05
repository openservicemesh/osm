package kubernetes

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-service-mesh/osm/pkg/namespace"
)

const (
	testProvider  = "test-provider"
	testInformer  = "test-informer"
	testNamespace = "test-namespace"
)

var (
	monitoredNamespaces = []string{testNamespace}
)

var _ = Describe("Testing event handlers", func() {
	Context("Test Add on a monitored namespace", func() {
		fakeNamespaceController := namespace.NewFakeNamespaceController(monitoredNamespaces)

		It("Should add the event to the announcement channel", func() {
			announcements := make(chan interface{}, 1)
			var pod corev1.Pod
			pod.Namespace = testNamespace
			Add(testInformer, testProvider, announcements, fakeNamespaceController)(&pod)
			Expect(len(announcements)).To(Equal(1))
			<-announcements
		})

		It("Should not add the event to the announcement channel", func() {
			announcements := make(chan interface{}, 1)
			var pod corev1.Pod
			pod.Namespace = "not-a-monitored-namespace"
			Add(testInformer, testProvider, announcements, fakeNamespaceController)(&pod)
			Expect(len(announcements)).To(Equal(0))
		})
	})
})
