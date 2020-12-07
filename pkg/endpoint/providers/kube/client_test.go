package kube

import (
	"context"
	"fmt"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/utils"
)

var _ = Describe("Test Kube Client Provider", func() {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		mockConfigurator   *configurator.MockConfigurator

		fakeClientSet *fake.Clientset
		provider      endpoint.Provider
		err           error
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	stopChan := make(chan struct{})
	providerID := "provider"

	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()

	BeforeEach(func() {
		fakeClientSet = fake.NewSimpleClientset()
		provider, err = NewProvider(fakeClientSet, mockKubeController, stopChan, providerID, mockConfigurator)
		Expect(err).ToNot(HaveOccurred())
	})

	It("tests GetID", func() {
		Expect(provider.GetID()).To(Equal(providerID))
	})

	It("should correctly return a list of endpoints for a service", func() {
		// Should be empty for now
		endpointChannel := events.GetPubSubInstance().Subscribe(announcements.EndpointAdded,
			announcements.EndpointDeleted,
			announcements.EndpointUpdated)
		defer events.GetPubSubInstance().Unsub(endpointChannel)

		Expect(provider.ListEndpointsForService(tests.BookbuyerService)).To(BeNil())

		// Create bookbuyer endpoint in Bookbuyer namespace
		endp := &corev1.Endpoints{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Endpoints",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tests.BookbuyerService.Name,
				Namespace: tests.BookbuyerService.Namespace,
			},
			Subsets: []v1.EndpointSubset{
				{
					Addresses: []v1.EndpointAddress{
						{
							IP: "8.8.8.8",
						},
					},
					Ports: []v1.EndpointPort{
						{
							Name:     "port",
							Port:     88,
							Protocol: v1.ProtocolTCP,
						},
					},
				},
			},
		}

		_, err = fakeClientSet.CoreV1().Endpoints(tests.BookbuyerService.Namespace).Create(context.TODO(), endp, metav1.CreateOptions{})
		Expect(err).To(BeNil())
		<-endpointChannel

		Expect(provider.ListEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: 88,
			},
		}))
	})

	It("GetResolvableEndpoints should properly return endpoints based on ClusterIP when set", func() {
		// If the service has cluster IP, expect the cluster IP + port
		mockKubeController.EXPECT().GetService(tests.BookbuyerService).Return(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tests.BookbuyerService.Name,
				Namespace: tests.BookbuyerService.Namespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "192.168.0.1",
				Ports: []corev1.ServicePort{{
					Name:     "servicePort",
					Protocol: corev1.ProtocolTCP,
					Port:     tests.ServicePort,
				}},
				Selector: map[string]string{
					"some-label": "test",
				},
			},
		})

		Expect(provider.GetResolvableEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(192, 168, 0, 1),
				Port: tests.ServicePort,
			},
		}))
	})

	It("GetResolvableEndpoints should properly return actual endpoints without ClusterIP when ClusterIP is not set", func() {
		endpointChannel := events.GetPubSubInstance().Subscribe(announcements.EndpointAdded,
			announcements.EndpointDeleted,
			announcements.EndpointUpdated)
		defer events.GetPubSubInstance().Unsub(endpointChannel)

		// Expect the individual pod endpoints, when no cluster IP is assigned to the service
		mockKubeController.EXPECT().GetService(tests.BookbuyerService).Return(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tests.BookbuyerService.Name,
				Namespace: tests.BookbuyerService.Namespace,
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
		})

		// Create bookbuyer endpoint in Bookbuyer namespace
		endp := &corev1.Endpoints{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Endpoints",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tests.BookbuyerService.Name,
				Namespace: tests.BookbuyerService.Namespace,
			},
			Subsets: []v1.EndpointSubset{
				{
					Addresses: []v1.EndpointAddress{
						{
							IP: "8.8.8.8",
						},
					},
					Ports: []v1.EndpointPort{
						{
							Name:     "port",
							Port:     88,
							Protocol: v1.ProtocolTCP,
						},
					},
				},
			},
		}

		_, err = fakeClientSet.CoreV1().Endpoints(tests.BookbuyerService.Namespace).Create(context.TODO(), endp, metav1.CreateOptions{})
		Expect(err).To(BeNil())
		<-endpointChannel

		Expect(provider.GetResolvableEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: 88,
			},
		}))
	})

	It("should correctly return the port to protocol mapping for a service's endpoints", func() {
		endpointChannel := events.GetPubSubInstance().Subscribe(announcements.EndpointAdded,
			announcements.EndpointDeleted,
			announcements.EndpointUpdated)
		defer events.GetPubSubInstance().Unsub(endpointChannel)

		appProtoHTTP := "http"
		appProtoTCP := "tcp"

		// Create bookbuyer endpoint in Bookbuyer namespace
		endp := &corev1.Endpoints{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Endpoints",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tests.BookbuyerService.Name,
				Namespace: tests.BookbuyerService.Namespace,
			},
			Subsets: []v1.EndpointSubset{
				{
					Addresses: []v1.EndpointAddress{
						{
							IP: "8.8.8.8",
						},
					},
					Ports: []v1.EndpointPort{
						{
							Name:        "port",
							Port:        88,
							Protocol:    v1.ProtocolTCP,
							AppProtocol: &appProtoTCP,
						},
						{
							Name:        "port",
							Port:        80,
							Protocol:    v1.ProtocolTCP,
							AppProtocol: &appProtoHTTP,
						},
					},
				},
			},
		}

		_, err = fakeClientSet.CoreV1().Endpoints(tests.BookbuyerService.Namespace).
			Create(context.TODO(), endp, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		<-endpointChannel
		portToProtocolMap, err := provider.GetPortToProtocolMappingForService(tests.BookbuyerService)
		Expect(err).To(BeNil())

		expectedPortToProtocolMap := map[uint32]string{88: appProtoTCP, 80: appProtoHTTP}
		Expect(portToProtocolMap).To(Equal(expectedPortToProtocolMap))
	})

	It("tests GetAnnouncementChannel", func() {
		ch := provider.GetAnnouncementsChannel()

		// Can only expect to not be null
		Expect(ch).ToNot(BeNil())
	})
})

var _ = Describe("When getting a Service associated with a ServiceAccount", func() {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		mockConfigurator   *configurator.MockConfigurator
	)
	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	var (
		fakeClientSet *fake.Clientset
		provider      endpoint.Provider
		err           error
	)

	providerID := "test-provider"
	testNamespace := "test"
	stop := make(chan struct{})

	// Configure the controller
	listMonitoredNs := []string{
		testNamespace,
	}
	mockKubeController.EXPECT().ListServices().DoAndReturn(func() []*corev1.Service {
		var services []*corev1.Service

		for _, ns := range listMonitoredNs {
			// simulate lookup on controller cache
			svcList, _ := fakeClientSet.CoreV1().Services(ns).List(context.TODO(), metav1.ListOptions{})
			for serviceIdx := range svcList.Items {
				services = append(services, &svcList.Items[serviceIdx])
			}
		}

		return services
	}).AnyTimes()
	mockKubeController.EXPECT().GetService(gomock.Any()).DoAndReturn(func(msh service.MeshService) *v1.Service {
		// simulate lookup on controller cache
		vv, err := fakeClientSet.CoreV1().Services(msh.Namespace).Get(context.TODO(), msh.Name, metav1.GetOptions{})

		if err != nil {
			return nil
		}

		return vv
	}).AnyTimes()

	mockKubeController.EXPECT().IsMonitoredNamespace(testNamespace).Return(true).AnyTimes()

	BeforeEach(func() {
		fakeClientSet = fake.NewSimpleClientset()
		provider, err = NewProvider(fakeClientSet, mockKubeController, stop, providerID, mockConfigurator)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return not return a service when a pod matching the selector doesn't exist", func() {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: testNamespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:     "servicePort",
					Protocol: corev1.ProtocolTCP,
					Port:     tests.ServicePort,
				}},
				Selector: map[string]string{
					"app": "test",
				},
			},
		}

		_, err := fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		services, err := provider.GetServicesForServiceAccount(tests.BookbuyerServiceAccount)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(services)).To(Equal(1))
		expectedServiceName := fmt.Sprintf("bookbuyer.default.osm.synthetic-%s", service.SyntheticServiceSuffix)
		Expect(services[0].Name).To(Equal(expectedServiceName))

		err = fakeClientSet.CoreV1().Services(testNamespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return a service that matches the ServiceAccount associated with the Pod", func() {
		podsChannel := events.GetPubSubInstance().Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated)
		defer events.GetPubSubInstance().Unsub(podsChannel)

		// Create a Service
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: testNamespace,
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

		_, err := fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a pod with labels that match the service selector
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"some-label": "test",
					"version":    "v1",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-service-account",
				Containers: []corev1.Container{
					{
						Name:  "BookbuyerContainerA",
						Image: "random",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http",
								Protocol:      corev1.ProtocolTCP,
								ContainerPort: 80,
							},
						},
					},
				},
			},
		}

		_, err = fakeClientSet.CoreV1().Pods(testNamespace).Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel

		givenSvcAccount := service.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Pod spec above
		}

		// Expect a MeshService that corresponds to a Service that matches the Pod spec labels
		expectedMeshSvc := utils.K8sSvcToMeshSvc(svc)

		meshSvcs, err := provider.GetServicesForServiceAccount(givenSvcAccount)
		Expect(err).ToNot(HaveOccurred())
		expectedMeshSvcs := []service.MeshService{expectedMeshSvc}
		Expect(meshSvcs).To(Equal(expectedMeshSvcs))

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel
	})

	It("should return an error when the Service selector doesn't match the pod", func() {
		podsChannel := events.GetPubSubInstance().Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated)
		defer events.GetPubSubInstance().Unsub(podsChannel)

		// Create a Service
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: testNamespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:     "servicePort",
					Protocol: corev1.ProtocolTCP,
					Port:     tests.ServicePort,
				}},
				Selector: map[string]string{
					"app":                         "test",
					"key-specified-in-deployment": "no", // Since this label is missing in the deployment, the selector match should fail
				},
			},
		}

		_, err := fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a Pod with labels that match the service selector
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": "test",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-service-account",
				Containers: []corev1.Container{
					{
						Name:  "BookbuyerContainerA",
						Image: "random",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http",
								Protocol:      corev1.ProtocolTCP,
								ContainerPort: 80,
							},
						},
					},
				},
			},
		}

		_, err = fakeClientSet.CoreV1().Pods(testNamespace).Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel

		givenSvcAccount := service.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Deployment spec above
		}

		// Expect a MeshService that corresponds to a Service that matches the Deployment spec labels
		svcs, err := provider.GetServicesForServiceAccount(givenSvcAccount)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(svcs)).To(Equal(1))
		expectedServiceName := fmt.Sprintf("test-service-account.test.osm.synthetic-%s", service.SyntheticServiceSuffix)
		Expect(svcs[0].Name).To(Equal(expectedServiceName))

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel
	})

	It("should return all services when multiple services match the same Pod", func() {
		// This test is meant to ensure the
		// service selector logic works as expected when multiple services
		// have the same selector match.
		podsChannel := events.GetPubSubInstance().Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated)
		defer events.GetPubSubInstance().Unsub(podsChannel)

		// Create a Service
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: testNamespace,
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

		_, err := fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a second service with the same selector as the first service
		svc2 := *svc
		svc2.Name = "test-2"
		_, err = fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), &svc2, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a Pod with labels that match the service selector
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"some-label": "test",
					"version":    "v1",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-service-account",
				Containers: []corev1.Container{
					{
						Name:  "BookbuyerContainerA",
						Image: "random",
						Ports: []corev1.ContainerPort{
							{
								Name:          "http",
								Protocol:      corev1.ProtocolTCP,
								ContainerPort: 80,
							},
						},
					},
				},
			},
		}

		_, err = fakeClientSet.CoreV1().Pods(testNamespace).Create(context.Background(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel

		givenSvcAccount := service.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Deployment spec above
		}

		meshServices, err := provider.GetServicesForServiceAccount(givenSvcAccount)
		Expect(err).ToNot(HaveOccurred())
		expectedServices := []service.MeshService{
			{Name: "test-1", Namespace: testNamespace},
			{Name: "test-2", Namespace: testNamespace},
		}
		Expect(len(meshServices)).To(Equal(len(expectedServices)))
		Expect(meshServices[0]).To(BeElementOf(expectedServices))
		Expect(meshServices[1]).To(BeElementOf(expectedServices))

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel
	})
})
