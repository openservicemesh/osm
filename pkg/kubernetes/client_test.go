package kubernetes

import (
	"context"
	"fmt"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var (
	testMeshName = "mesh"
)

const (
	nsInformerSyncTimeout = 3 * time.Second
)

var _ = Describe("Test Namespace KubeController Methods", func() {
	Context("Testing ListMonitoredNamespaces", func() {
		It("should return monitored namespaces", func() {
			// Create namespace controller
			kubeClient := testclient.NewSimpleClientset()
			stop := make(chan struct{})
			kubeController, err := NewKubernetesController(kubeClient, testMeshName, stop)
			Expect(err).ToNot(HaveOccurred())
			Expect(kubeController).ToNot(BeNil())

			// Create a test namespace that is monitored
			testNamespaceName := fmt.Sprintf("%s-1", tests.Namespace)
			testNamespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNamespaceName,
					Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMeshName},
				},
			}
			_, err = kubeClient.CoreV1().Namespaces().Create(context.TODO(), &testNamespace, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// Eventually asserts that all return values apart from the first value are nil or zero-valued,
			// so asserting that an error is nil is implicit.
			Eventually(func() ([]string, error) {
				return kubeController.ListMonitoredNamespaces()
			}, nsInformerSyncTimeout).Should(Equal([]string{testNamespaceName}))
		})
	})

	Context("Testing GetNamespace", func() {
		It("should return existing namespace if it exists", func() {
			// Create namespace controller
			kubeClient := testclient.NewSimpleClientset()
			stop := make(chan struct{})
			kubeController, err := NewKubernetesController(kubeClient, testMeshName, stop)
			Expect(err).ToNot(HaveOccurred())
			Expect(kubeController).ToNot(BeNil())

			// Create a test namespace that is monitored
			testNamespaceName := fmt.Sprintf("%s-1", tests.Namespace)
			testNamespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNamespaceName,
					Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMeshName},
				},
			}

			// Create it
			nsCreate, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), &testNamespace, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// Check it is present
			Eventually(func() *corev1.Namespace {
				return kubeController.GetNamespace(testNamespaceName)
			}, nsInformerSyncTimeout).Should(Equal(nsCreate))

			// Delete it
			err = kubeClient.CoreV1().Namespaces().Delete(context.TODO(), testNamespaceName, metav1.DeleteOptions{})
			Expect(err).To(BeNil())

			// Check it is gone
			Eventually(func() *corev1.Namespace {
				return kubeController.GetNamespace(testNamespaceName)
			}, nsInformerSyncTimeout).Should(BeNil())
		})
	})

	Context("Testing IsMonitoredNamespace", func() {
		It("should work as expected", func() {
			// Create namespace controller
			kubeClient := testclient.NewSimpleClientset()
			stop := make(chan struct{})
			kubeController, err := NewKubernetesController(kubeClient, testMeshName, stop)
			Expect(err).ToNot(HaveOccurred())
			Expect(kubeController).ToNot(BeNil())

			// Create a test namespace that is monitored
			testNamespaceName := fmt.Sprintf("%s-1", tests.Namespace)
			testNamespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNamespaceName,
					Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMeshName},
				},
			}

			_, err = kubeClient.CoreV1().Namespaces().Create(context.TODO(), &testNamespace, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			Eventually(func() bool {
				return kubeController.IsMonitoredNamespace(testNamespaceName)
			}, nsInformerSyncTimeout).Should(BeTrue())

			fakeNamespaceIsMonitored := kubeController.IsMonitoredNamespace("fake")
			Expect(fakeNamespaceIsMonitored).To(BeFalse())
		})
	})

	Context("service controller", func() {
		var kubeClient *testclient.Clientset
		var kubeController Controller
		var err error

		BeforeEach(func() {
			kubeClient = testclient.NewSimpleClientset()
			kubeController, err = NewKubernetesController(kubeClient, testMeshName, make(chan struct{}))
			Expect(err).ToNot(HaveOccurred())
			Expect(kubeController).ToNot(BeNil())
		})

		It("should create and delete services, and be detected if NS is monitored", func() {
			meshSvc := tests.BookbuyerService
			serviceChannel := events.GetPubSubInstance().Subscribe(announcements.ServiceAdded,
				announcements.ServiceDeleted,
				announcements.ServiceUpdated)
			defer events.GetPubSubInstance().Unsub(serviceChannel)

			// Create monitored namespace for this service
			testNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   tests.BookbuyerService.Namespace,
					Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMeshName},
				},
			}
			_, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
			Expect(err).To(BeNil())
			// Wait on namespace to be ready so that resources in this namespace are marked as monitored as soon as possible
			Eventually(func() bool {
				return kubeController.IsMonitoredNamespace(testNamespace.Name)
			}, nsInformerSyncTimeout).Should(BeTrue())

			svc := tests.NewServiceFixture(meshSvc.Name, meshSvc.Namespace, nil)
			_, err = kubeClient.CoreV1().Services(meshSvc.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			<-serviceChannel

			svcIncache := kubeController.GetService(meshSvc)
			Expect(svcIncache).To(Equal(svc))

			err = kubeClient.CoreV1().Services(meshSvc.Namespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			<-serviceChannel

			svcIncache = kubeController.GetService(meshSvc)
			Expect(svcIncache).To(BeNil())
		})

		It("should return nil when the given MeshService is not found", func() {
			meshSvc := tests.BookbuyerService

			svcIncache := kubeController.GetService(meshSvc)
			Expect(svcIncache).To(BeNil())
		})

		It("should return an empty list when no services are found", func() {
			services := kubeController.ListServices()
			Expect(len(services)).To(Equal(0))
		})

		It("should return a list of Services", func() {
			// Define services to test with
			serviceChannel := events.GetPubSubInstance().Subscribe(announcements.ServiceAdded,
				announcements.ServiceDeleted,
				announcements.ServiceUpdated)
			defer events.GetPubSubInstance().Unsub(serviceChannel)
			testSvcs := []service.MeshService{
				{Name: uuid.New().String(), Namespace: "ns-1"},
				{Name: uuid.New().String(), Namespace: "ns-2"},
			}

			// Test services could belong to the same namespace, so ensure we create a list of unique namespaces
			testNamespaces := mapset.NewSet()
			for _, svc := range testSvcs {
				testNamespaces.Add(svc.Namespace)
			}

			// Create a namespace resource for each namespace
			for ns := range testNamespaces.Iter() {
				namespace := ns.(string)

				testNamespace := corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   namespace,
						Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMeshName},
					},
				}
				_, err = kubeClient.CoreV1().Namespaces().Create(context.TODO(), &testNamespace, metav1.CreateOptions{})
				Expect(err).To(BeNil())
			}
			for ns := range testNamespaces.Iter() {
				namespace := ns.(string)
				// Wait on namespace to be ready so that resources in this namespace are marked as monitored as soon as possible
				Eventually(func() bool {
					return kubeController.IsMonitoredNamespace(namespace)
				}, nsInformerSyncTimeout).Should(BeTrue())
			}

			// Add services
			for _, svcAdd := range testSvcs {
				svcSpec := tests.NewServiceFixture(svcAdd.Name, svcAdd.Namespace, nil)
				_, err := kubeClient.CoreV1().Services(svcAdd.Namespace).Create(context.TODO(), svcSpec, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			// Wait for all the service related events: 1 for each service created
			for range testSvcs {
				<-serviceChannel
			}

			services := kubeController.ListServices()
			Expect(len(testSvcs)).To(Equal(len(services)))
		})
	})

	Context("Test ListServiceAccountsForService()", func() {
		var kubeClient *testclient.Clientset
		var kubeController Controller
		var err error
		testMeshName := "foo"

		BeforeEach(func() {
			kubeClient = testclient.NewSimpleClientset()
			kubeController, err = NewKubernetesController(kubeClient, testMeshName, make(chan struct{}))
			Expect(err).ToNot(HaveOccurred())
			Expect(kubeController).ToNot(BeNil())
		})

		It("should correctly return the ServiceAccounts associated with a service", func() {
			testNamespaceName := "test-ns"
			testSvcAccountName1 := "test-service-account-1"
			testSvcAccountName2 := "test-service-account-2"

			serviceChannel := events.GetPubSubInstance().Subscribe(announcements.ServiceAdded,
				announcements.ServiceDeleted,
				announcements.ServiceUpdated)
			defer events.GetPubSubInstance().Unsub(serviceChannel)
			podsChannel := events.GetPubSubInstance().Subscribe(announcements.PodAdded,
				announcements.PodDeleted,
				announcements.PodUpdated)
			defer events.GetPubSubInstance().Unsub(podsChannel)

			// Create a namespace
			testNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNamespaceName,
					Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: testMeshName},
				},
			}
			_, err = kubeClient.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
			Expect(err).To(BeNil())
			// Wait on namespace to be ready so that resources in this namespace are marked as monitored as soon as possible
			Eventually(func() bool {
				return kubeController.IsMonitoredNamespace(testNamespace.Name)
			}, nsInformerSyncTimeout).Should(BeTrue())

			// Create pods with labels that match the service
			pod1 := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: testNamespaceName,
					Labels: map[string]string{
						"some-label": "test",
						"version":    "v1",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: testSvcAccountName1,
				},
			}
			_, err = kubeClient.CoreV1().Pods(testNamespaceName).Create(context.TODO(), pod1, metav1.CreateOptions{})
			Expect(err).To(BeNil())
			<-podsChannel

			pod2 := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: testNamespaceName,
					Labels: map[string]string{
						"some-label": "test",
						"version":    "v2",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: testSvcAccountName2,
				},
			}
			_, err = kubeClient.CoreV1().Pods(testNamespaceName).Create(context.TODO(), pod2, metav1.CreateOptions{})
			Expect(err).To(BeNil())
			<-podsChannel

			// Create a service with selector that matches the pods above
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: testNamespaceName,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{
						Name:     "servicePort",
						Protocol: corev1.ProtocolTCP,
						Port:     tests.ServicePort,
					}},
					Selector: map[string]string{
						"some-label": "test",
					},
				},
			}

			_, err := kubeClient.CoreV1().Services(testNamespaceName).Create(context.TODO(), svc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			<-serviceChannel

			meshSvc := service.MeshService{Name: svc.Name, Namespace: svc.Namespace}

			svcAccounts, err := kubeController.ListServiceAccountsForService(meshSvc)

			Expect(err).ToNot(HaveOccurred())

			expectedSvcAccounts := []service.K8sServiceAccount{
				{Name: pod1.Spec.ServiceAccountName, Namespace: pod1.Namespace},
				{Name: pod2.Spec.ServiceAccountName, Namespace: pod2.Namespace},
			}
			Expect(svcAccounts).Should(HaveLen(len(expectedSvcAccounts)))
			Expect(svcAccounts).Should(ConsistOf(expectedSvcAccounts))
		})

	})

})
