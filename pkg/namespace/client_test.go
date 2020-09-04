package namespace

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/tests"
)

var (
	testMeshName = "mesh"
)

var _ = Describe("Test Namespace Controller Methods", func() {
	Context("Testing namespace controller", func() {
		It("should return a new namespace controller", func() {
			kubeClient := testclient.NewSimpleClientset()
			stop := make(chan struct{})
			namespaceController := k8s.NewKubernetesClient(kubeClient, testMeshName, stop)
			Expect(namespaceController).ToNot(BeNil())
		})
	})

	Context("Testing ListMonitoredNamespaces", func() {
		It("should return monitored namespaces", func() {
			// Create namespace controller
			kubeClient := testclient.NewSimpleClientset()
			stop := make(chan struct{})
			namespaceController := k8s.NewKubernetesClient(kubeClient, testMeshName, stop)

			// Create a test namespace that is monitored
			testNamespaceName := fmt.Sprintf("%s-1", tests.Namespace)
			testNamespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNamespaceName,
					Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMeshName},
				},
			}
			if _, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), &testNamespace, metav1.CreateOptions{}); err != nil {
				log.Fatal().Err(err).Msgf("Error creating Namespace %v", testNamespace)
			}
			<-namespaceController.GetAnnouncementsChannel()

			monitoredNamespaces, err := namespaceController.ListMonitoredNamespaces()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(monitoredNamespaces)).To(Equal(1))
			Expect(testNamespaceName).To(BeElementOf(monitoredNamespaces))
		})
	})

	Context("Testing IsMonitoredNamespace", func() {
		It("should work as expected", func() {
			// Create namespace controller
			kubeClient := testclient.NewSimpleClientset()
			stop := make(chan struct{})
			namespaceController := k8s.NewKubernetesClient(kubeClient, testMeshName, stop)

			// Create a test namespace that is monitored
			testNamespaceName := fmt.Sprintf("%s-1", tests.Namespace)
			testNamespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNamespaceName,
					Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMeshName},
				},
			}

			if _, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), &testNamespace, metav1.CreateOptions{}); err != nil {
				log.Fatal().Err(err).Msgf("Error creating Namespace %v", testNamespace)
			}
			<-namespaceController.GetAnnouncementsChannel()

			namespaceIsMonitored := namespaceController.IsMonitoredNamespace(testNamespaceName)
			Expect(namespaceIsMonitored).To(BeTrue())

			fakeNamespaceIsMonitored := namespaceController.IsMonitoredNamespace("fake")
			Expect(fakeNamespaceIsMonitored).ToNot(BeTrue())
		})
	})
})
