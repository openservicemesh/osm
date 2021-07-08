package kube

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/config"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/utils"
)

const (
	providerID    = "test-provider"
	testNamespace = "testNamespace"
	meshName      = "meshName"
)

var _ = Describe("Test Kube Client Provider (w/o kubecontroller)", func() {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		mockConfigurator   *configurator.MockConfigurator
		client             *Client
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
	mockConfigController := config.NewMockController(mockCtrl)

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableMulticlusterMode: true,
	}).AnyTimes()

	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()
	mockConfigController.EXPECT().ListMultiClusterServices().Return(
		[]*v1alpha1.MultiClusterService{tests.BookstoreMCS, tests.BookbuyerMCS}).AnyTimes()
	mockConfigController.EXPECT().GetMultiClusterService(
		tests.BookbuyerService.Name, tests.BookbuyerService.Namespace).Return(tests.BookbuyerMCS).AnyTimes()
	BeforeEach(func() {
		client = NewClient(mockKubeController, mockConfigController, providerID, mockConfigurator)
	})

	It("tests GetID", func() {
		Expect(client.GetID()).To(Equal(providerID))
	})

	It("should correctly return a list of endpoints for a service", func() {
		// Should be empty for now
		mockKubeController.EXPECT().GetEndpoints(tests.BookbuyerService).Return(&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tests.BookbuyerService.Namespace,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "8.8.8.8",
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Port: 88,
						},
					},
				},
			},
		}, nil)

		Expect(client.ListEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: 88,
			},
		}))
	})

	It("should correctly return a list of endpoints for a global service", func() {
		mockConfigController.EXPECT().GetMultiClusterService(tests.BookbuyerGlobalService.Name, tests.BookbuyerGlobalService.Namespace).Return(tests.BookbuyerMCS)

		Expect(client.ListEndpointsForService(tests.BookbuyerGlobalService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(10, 10, 10, 11),
				Port: 80,
			},
			{
				IP:   net.IPv4(10, 10, 10, 12),
				Port: 90,
			},
		}))
	})

	It("should correctly return a list of endpoints for a remote service", func() {
		mockConfigController.EXPECT().GetMultiClusterService(tests.BookbuyerClusterXService.Name, tests.BookbuyerClusterXService.Namespace).Return(tests.BookbuyerMCS)

		Expect(client.ListEndpointsForService(tests.BookbuyerClusterXService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(10, 10, 10, 11),
				Port: 80,
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

		Expect(client.GetResolvableEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(192, 168, 0, 1),
				Port: tests.ServicePort,
			},
		}))
	})

	It("GetResolvableEndpoints should properly return actual endpoints without ClusterIP when ClusterIP is not set", func() {
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

		mockKubeController.EXPECT().GetEndpoints(tests.BookbuyerService).Return(&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tests.BookbuyerService.Namespace,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "8.8.8.8",
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Name:     "port",
							Port:     88,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
		}, nil)

		Expect(client.GetResolvableEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: 88,
			},
		}))
	})

	It("GetResolvableEndpoints should properly return actual endpoints when ClusterIP is none", func() {

		// If the service has cluster IP set to none, expect the individual pod endpoints
		mockKubeController.EXPECT().GetService(tests.BookbuyerService).Return(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tests.BookbuyerService.Name,
				Namespace: tests.BookbuyerService.Namespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: corev1.ClusterIPNone,
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

		mockKubeController.EXPECT().GetEndpoints(tests.BookbuyerService).Return(&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tests.BookbuyerService.Namespace,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "8.8.8.8",
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Name:     "port",
							Port:     88,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
		}, nil)

		Expect(client.GetResolvableEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: 88,
			},
		}))
	})

	It("GetResolvableEndpoints should properly return the global IP when the service is global", func() {
		Expect(client.GetResolvableEndpointsForService(tests.BookbuyerGlobalService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(10, 10, 10, 30),
				Port: 8082,
			},
			{
				IP:   net.IPv4(10, 10, 10, 30),
				Port: 9091,
			},
		}))
	})

	It("GetResolvableEndpoints should properly return a single endpoint when domain is a remote cluster", func() {
		Expect(client.GetResolvableEndpointsForService(tests.BookbuyerClusterXService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(10, 10, 10, 11),
				Port: 8082,
			},
			{
				IP:   net.IPv4(10, 10, 10, 11),
				Port: 9091,
			},
		}))
	})

	It("should correctly return the port to protocol mapping for a service's endpoints", func() {

		appProtoHTTP := "http"
		appProtoTCP := "tcp"

		mockKubeController.EXPECT().GetEndpoints(tests.BookbuyerService).Return(&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tests.BookbuyerService.Namespace,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "8.8.8.8",
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Name:        "port1", // appProtocol specified
							Port:        70,
							Protocol:    corev1.ProtocolTCP,
							AppProtocol: &appProtoTCP,
						},
						{
							Name:        "port2", // appProtocol specified
							Port:        80,
							Protocol:    corev1.ProtocolTCP,
							AppProtocol: &appProtoHTTP,
						},
						{
							Name:     "http-port3", // appProtocol derived from port name
							Port:     90,
							Protocol: corev1.ProtocolTCP,
						},
						{
							Name:     "tcp-port4", // appProtocol derived from port name
							Port:     100,
							Protocol: corev1.ProtocolTCP,
						},
						{
							Name:     "grpc-port5", // appProtocol derived from port name
							Port:     110,
							Protocol: corev1.ProtocolTCP,
						},
						{
							Name:     "no-protocol-prefix", // appProtocol defaults to http
							Port:     120,
							Protocol: corev1.ProtocolTCP,
						},
						{
							Name:        "http-prefix",
							Port:        130,
							Protocol:    corev1.ProtocolTCP,
							AppProtocol: &appProtoTCP, // AppProtocol takes precedence over Name
						},
					},
				},
			},
		}, nil)

		portToProtocolMap, err := client.GetTargetPortToProtocolMappingForService(tests.BookbuyerService)
		Expect(err).To(BeNil())

		expectedPortToProtocolMap := map[uint32]string{70: "tcp", 80: "http", 90: "http", 100: "tcp", 110: "grpc", 120: "http", 130: "tcp"}
		Expect(portToProtocolMap).To(Equal(expectedPortToProtocolMap))
	})

	It("should return an error for remote service", func() {
		_, err := client.GetTargetPortToProtocolMappingForService(tests.BookbuyerClusterXService)
		Expect(err).NotTo(BeNil())
	})

	It("should return an error for global service", func() {
		_, err := client.GetTargetPortToProtocolMappingForService(tests.BookbuyerGlobalService)
		Expect(err).NotTo(BeNil())
	})
})

var _ = Describe("Test Kube Client Provider (/w kubecontroller)", func() {
	var (
		mockCtrl         *gomock.Controller
		kubeController   k8s.Controller
		mockConfigurator *configurator.MockConfigurator
		fakeClientSet    *testclient.Clientset
		client           *Client
		err              error
	)
	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
	mockConfigController := config.NewMockController(mockCtrl)

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableMulticlusterMode: true,
	}).AnyTimes()

	stop := make(chan struct{})

	mockConfigController.EXPECT().ListMultiClusterServices().Return(
		[]*v1alpha1.MultiClusterService{tests.BookstoreMCS, tests.BookbuyerMCS,
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-2",
					Namespace: testNamespace,
				},
				Spec: v1alpha1.MultiClusterServiceSpec{
					ServiceAccount: "test-service-account",
					Ports: []v1alpha1.PortSpec{
						{
							Port:     8082,
							Protocol: "HTTP",
						},
						{
							Port:     9091,
							Protocol: "TCP",
						},
					},
					Clusters: []v1alpha1.ClusterSpec{
						{
							Name:    "cluster-x",
							Address: "10.10.10.11:80",
						},
						{
							Name:    "cluster-y",
							Address: "10.10.10.12:90",
						},
					},
				},
			},
		}).AnyTimes()
	mockConfigController.EXPECT().GetMultiClusterService(
		tests.BookbuyerService.Name, tests.BookbuyerService.Namespace).Return(tests.BookbuyerMCS).AnyTimes()

	BeforeEach(func() {
		fakeClientSet = testclient.NewSimpleClientset()
		kubeController, err = k8s.NewKubernetesController(fakeClientSet, meshName, stop)

		// Add the monitored namespace
		testNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   testNamespace,
				Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: meshName},
			},
		}
		_, err = fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		Eventually(func() bool {
			return kubeController.IsMonitoredNamespace(testNamespace.Name)
		}, 3*time.Second).Should(BeTrue())

		Expect(err).ToNot(HaveOccurred())
		client = NewClient(kubeController, mockConfigController, providerID, mockConfigurator)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		stop <- struct{}{}
	})

	It("should return an error when a pod matching the selector doesn't exist", func() {
		// create a separate client scoped to this function only
		mockCtrl := gomock.NewController(GinkgoT())
		mockConfigController := config.NewMockController(mockCtrl)

		stop := make(chan struct{})

		mockConfigController.EXPECT().ListMultiClusterServices().Return(nil)
		kubeController, err := k8s.NewKubernetesController(fakeClientSet, meshName, stop)
		Expect(err).To(BeNil())

		client := NewClient(kubeController, mockConfigController, providerID, mockConfigurator)

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

		_, err = fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		services, err := client.GetServicesForServiceIdentity(tests.BookbuyerServiceIdentity)
		Expect(err).To(HaveOccurred())

		Expect(services).To(BeNil())

		err = fakeClientSet.CoreV1().Services(testNamespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return a service that matches the ServiceAccount associated with the Pod and MultiClusterServices", func() {
		podsAndServiceChannel := events.GetPubSubInstance().Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated,
			announcements.ServiceAdded,
			announcements.ServiceDeleted,
			announcements.ServiceUpdated,
		)
		defer events.GetPubSubInstance().Unsub(podsAndServiceChannel)

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
		<-podsAndServiceChannel

		// Create a pod with labels that match the service selector
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"some-label": "test",
					"version":    "v1",
				},
				Namespace: testNamespace,
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
		<-podsAndServiceChannel

		givenSvcAccount := identity.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Pod spec above
		}

		// Expect a MeshService that corresponds to a Service that matches the Pod spec labels
		expectedMeshSvc := utils.K8sSvcToMeshSvc(svc)

		meshSvcs, err := client.GetServicesForServiceIdentity(givenSvcAccount.ToServiceIdentity())
		Expect(err).ToNot(HaveOccurred())
		expectedMeshSvcs := []service.MeshService{expectedMeshSvc,
			{
				Namespace:     "testNamespace",
				Name:          "test-2",
				ClusterDomain: "cluster-x",
			},
			{
				Namespace:     "testNamespace",
				Name:          "test-2",
				ClusterDomain: "cluster-y",
			},
		}
		Expect(meshSvcs).To(ConsistOf(expectedMeshSvcs))

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsAndServiceChannel
	})

	It("should return an error when the Service selector doesn't match the pod", func() {
		podsChannel := events.GetPubSubInstance().Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated)
		defer events.GetPubSubInstance().Unsub(podsChannel)

		// create a separate client scoped to this function only so it doesn't return multi cluster services
		mockCtrl := gomock.NewController(GinkgoT())
		mockConfigController := config.NewMockController(mockCtrl)

		stop := make(chan struct{})

		mockConfigController.EXPECT().ListMultiClusterServices().Return(nil)
		kubeController, err := k8s.NewKubernetesController(fakeClientSet, meshName, stop)
		Expect(err).To(BeNil())
		client := NewClient(kubeController, mockConfigController, providerID, mockConfigurator)

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

		_, err = fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a Pod with labels that match the service selector
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": "test",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-service-account-2",
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

		givenSvcAccount := identity.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account-2", // Should match the service account in the Deployment spec above
		}

		// Expect a MeshService that corresponds to a Service that matches the Deployment spec labels
		svcs, err := client.GetServicesForServiceIdentity(givenSvcAccount.ToServiceIdentity())
		Expect(err).To(HaveOccurred())
		Expect(svcs).To(BeNil())

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel
	})

	It("should return an error when the service doesn't have a selector", func() {
		// create a separate client scoped to this function only
		mockCtrl := gomock.NewController(GinkgoT())
		mockConfigController := config.NewMockController(mockCtrl)

		stop := make(chan struct{})

		mockConfigController.EXPECT().ListMultiClusterServices().Return(nil)
		kubeController, err := k8s.NewKubernetesController(fakeClientSet, meshName, stop)
		Expect(err).To(BeNil())
		client := NewClient(kubeController, mockConfigController, providerID, mockConfigurator)

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
			},
		}

		_, err = fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Create a Pod with labels that match the service selector
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": "test",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "test-service-account-2",
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

		givenSvcAccount := identity.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account-2", // Should match the service account in the Deployment spec above
		}

		// Expect a MeshService that corresponds to a Service that matches the Deployment spec labels
		svcs, err := client.GetServicesForServiceIdentity(givenSvcAccount.ToServiceIdentity())
		Expect(err).To(HaveOccurred())
		Expect(svcs).To(BeNil())

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel
	})

	It("should return all services when multiple services match the same Pod", func() {
		// This test is meant to ensure the
		// service selector logic works as expected when multiple services
		// have the same selector match.
		podsAndServiceChannel := events.GetPubSubInstance().Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated,
			announcements.ServiceAdded,
			announcements.ServiceDeleted,
			announcements.ServiceUpdated,
		)
		defer events.GetPubSubInstance().Unsub(podsAndServiceChannel)

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
		<-podsAndServiceChannel

		// Create a second service with the same selector as the first service
		svc2 := *svc
		svc2.Name = "test-2"
		_, err = fakeClientSet.CoreV1().Services(testNamespace).Create(context.TODO(), &svc2, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsAndServiceChannel

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
		<-podsAndServiceChannel

		givenSvcAccount := identity.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Deployment spec above
		}

		meshServices, err := client.GetServicesForServiceIdentity(givenSvcAccount.ToServiceIdentity())
		Expect(err).ToNot(HaveOccurred())
		expectedServices := []service.MeshService{
			{Name: "test-1", Namespace: testNamespace, ClusterDomain: constants.LocalDomain},
			{Name: "test-2", Namespace: testNamespace, ClusterDomain: constants.LocalDomain},
			{Name: "test-2", Namespace: testNamespace, ClusterDomain: constants.ClusterDomain("cluster-x")},
			{Name: "test-2", Namespace: testNamespace, ClusterDomain: constants.ClusterDomain("cluster-y")},
		}
		Expect(len(meshServices)).To(Equal(len(expectedServices)))
		Expect(meshServices[0]).To(BeElementOf(expectedServices))
		Expect(meshServices[1]).To(BeElementOf(expectedServices))

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsAndServiceChannel
	})
})

func TestListEndpointsForIdentity(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                            string
		serviceAccount                  identity.ServiceIdentity
		outboundServiceAccountEndpoints map[identity.ServiceIdentity][]endpoint.Endpoint
		expectedEndpoints               []endpoint.Endpoint
	}{
		{
			name:           "get endpoints for pod with only one ip",
			serviceAccount: tests.BookstoreServiceIdentity,
			outboundServiceAccountEndpoints: map[identity.ServiceIdentity][]endpoint.Endpoint{
				tests.BookstoreServiceIdentity: {{
					IP: net.ParseIP(tests.ServiceIP),
				}},
			},
			expectedEndpoints: []endpoint.Endpoint{
				{
					IP: net.ParseIP(tests.ServiceIP),
				},
				// Values taken from the tests.BookstoreMCS fixture
				{
					IP:   net.ParseIP("10.10.10.15"),
					Port: 80,
				},
				{
					IP:   net.ParseIP("10.10.10.16"),
					Port: 90,
				},
			},
		},
		{
			name:           "get endpoints for pod with multiple ips",
			serviceAccount: tests.BookstoreServiceIdentity,
			outboundServiceAccountEndpoints: map[identity.ServiceIdentity][]endpoint.Endpoint{
				tests.BookstoreServiceIdentity: {
					endpoint.Endpoint{
						IP: net.ParseIP(tests.ServiceIP),
					},
					endpoint.Endpoint{
						IP: net.ParseIP("9.9.9.9"),
					},
				},
			},
			expectedEndpoints: []endpoint.Endpoint{
				{
					IP: net.ParseIP(tests.ServiceIP),
				},
				{
					IP: net.ParseIP("9.9.9.9"),
				},
				// Values taken from the tests.BookstoreMCS fixture
				{
					IP:   net.ParseIP("10.10.10.15"),
					Port: 80,
				},
				{
					IP:   net.ParseIP("10.10.10.16"),
					Port: 90,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			kubeClient := testclient.NewSimpleClientset()
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
				EnableMulticlusterMode: true,
			}).AnyTimes()
			mockConfigController := config.NewMockController(mockCtrl)

			mockConfigController.EXPECT().ListMultiClusterServices().Return(
				[]*v1alpha1.MultiClusterService{tests.BookstoreMCS, tests.BookbuyerMCS}).AnyTimes()

			provider := NewClient(mockKubeController, mockConfigController, providerID, mockConfigurator)

			var pods []*corev1.Pod
			for serviceIdentity, endpoints := range tc.outboundServiceAccountEndpoints {
				podlabels := map[string]string{
					tests.SelectorKey:                tests.SelectorValue,
					constants.EnvoyUniqueIDLabelName: uuid.New().String(),
				}
				sa := serviceIdentity.ToK8sServiceAccount()
				pod := tests.NewPodFixture(sa.Namespace, sa.Name, sa.Name, podlabels)
				var podIps []corev1.PodIP
				for _, ep := range endpoints {
					podIps = append(podIps, corev1.PodIP{IP: ep.IP.String()})
				}
				pod.Status.PodIPs = podIps
				_, err := kubeClient.CoreV1().Pods(sa.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
				assert.Nil(err)
				pods = append(pods, &pod)
			}
			mockKubeController.EXPECT().ListPods().Return(pods).AnyTimes()

			actual := provider.ListEndpointsForIdentity(tc.serviceAccount)
			assert.NotNil(actual)
			assert.ElementsMatch(actual, tc.expectedEndpoints)
		})
	}
}

func TestListServices(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(GinkgoT())
	mockKubeController := k8s.NewMockController(mockCtrl)
	mockConfigClient := config.NewMockController(mockCtrl)

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableMulticlusterMode: true,
	}).AnyTimes()

	mockKubeController.EXPECT().ListServices().Return([]*corev1.Service{
		tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, map[string]string{
			tests.SelectorKey: tests.SelectorValue,
		}),
		tests.NewServiceFixture(tests.BookwarehouseServiceName, tests.Namespace, map[string]string{
			tests.SelectorKey: tests.SelectorValue,
		}),
	}).AnyTimes()

	mockConfigClient.EXPECT().ListMultiClusterServices().Return([]*v1alpha1.MultiClusterService{tests.BookbuyerMCS, tests.BookstoreMCS}).AnyTimes()

	c := Client{configurator: mockConfigurator, kubeController: mockKubeController, configClient: mockConfigClient}

	svcs, err := c.ListServices()
	assert.NoError(err)
	assert.ElementsMatch([]service.MeshService{
		tests.BookbuyerService,
		tests.BookwarehouseService,
		tests.BookbuyerClusterXService,
		tests.BookbuyerClusterYService,
		tests.BookstoreClusterXService,
		tests.BookstoreClusterYService,
	}, svcs)
}

func TestListServiceIdentitiesForService(t *testing.T) {
	assert := tassert.New(t)

	// create a separate client scoped to this function only so it doesn't return multi cluster services
	mockCtrl := gomock.NewController(GinkgoT())
	mockConfigController := config.NewMockController(mockCtrl)
	mockKubeController := k8s.NewMockController(mockCtrl)

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableMulticlusterMode: true,
	}).AnyTimes()

	client := NewClient(mockKubeController, mockConfigController, providerID, mockConfigurator)

	unknownLocal := service.MeshService{Name: "unknown", Namespace: "unknown", ClusterDomain: constants.LocalDomain}
	unknownRemote := service.MeshService{Name: "unknown", Namespace: "unknown", ClusterDomain: "cluster-x"}
	knownLocal := service.MeshService{Name: "known", Namespace: "known", ClusterDomain: constants.LocalDomain}
	knownRemote := service.MeshService{Name: "known", Namespace: "known", ClusterDomain: "cluster-x"}

	mockKubeController.EXPECT().ListServiceAccountsForService(unknownLocal).Return(nil,
		errors.New("Error fetching service \"unknown/unknown/local\": Service not found")).AnyTimes()
	mockKubeController.EXPECT().ListServiceAccountsForService(knownLocal).Return(
		[]identity.K8sServiceAccount{{Namespace: "known", Name: "local-identity"}},
		nil).AnyTimes()

	mockConfigController.EXPECT().GetMultiClusterService(unknownRemote.Name, unknownRemote.Namespace).Return(nil).AnyTimes()
	mockConfigController.EXPECT().GetMultiClusterService(knownRemote.Name, knownRemote.Namespace).Return(
		&v1alpha1.MultiClusterService{ObjectMeta: metav1.ObjectMeta{Namespace: knownRemote.Namespace}, Spec: v1alpha1.MultiClusterServiceSpec{ServiceAccount: "remote-identity"}}).AnyTimes()

	testCases := []struct {
		name               string
		svc                service.MeshService
		expectedIdentities []identity.ServiceIdentity
		expectedErr        error
	}{
		{
			name:        "no local service accounts returns an error",
			svc:         unknownLocal,
			expectedErr: errors.New("Error fetching service \"unknown/unknown/local\": Service not found"),
		},
		{
			name:        "no remote service accounts returns an error",
			svc:         unknownRemote,
			expectedErr: fmt.Errorf("Error getting ServiceAccounts for Service unknown/unknown/cluster-x"),
		},
		{
			name:               "local service with identities",
			svc:                knownLocal,
			expectedIdentities: []identity.ServiceIdentity{"local-identity.known.cluster.local"},
		},
		{
			name:               "remote service with identities",
			svc:                knownRemote,
			expectedIdentities: []identity.ServiceIdentity{"remote-identity.known.cluster.local"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			identities, err := client.ListServiceIdentitiesForService(tc.svc)
			if tc.expectedErr != nil {
				assert.Error(err)
				assert.Equal(tc.expectedErr.Error(), err.Error())
			}
			assert.ElementsMatch(identities, tc.expectedIdentities)
		})
	}
}

func TestGetPortToProtocolMappingForService(t *testing.T) {
	assert := tassert.New(t)

	// create a separate client scoped to this function only so it doesn't return multi cluster services
	mockCtrl := gomock.NewController(GinkgoT())
	mockConfigController := config.NewMockController(mockCtrl)
	mockKubeController := k8s.NewMockController(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableMulticlusterMode: true,
	}).AnyTimes()

	unknownLocal := service.MeshService{Name: "unknown", Namespace: "unknown", ClusterDomain: constants.LocalDomain}
	unknownRemote := service.MeshService{Name: "unknown", Namespace: "unknown", ClusterDomain: "cluster-x"}
	knownLocal := service.MeshService{Name: "known", Namespace: "known", ClusterDomain: constants.LocalDomain}
	knownRemote := service.MeshService{Name: "known", Namespace: "known", ClusterDomain: "cluster-x"}

	protocol := string(constants.ProtocolHTTP)
	mockKubeController.EXPECT().GetService(unknownLocal).Return(nil)
	mockKubeController.EXPECT().GetService(knownLocal).Return(&corev1.Service{
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:        80,
					AppProtocol: &protocol,
				},
				{
					Port: 90,
					Name: "tcp-test",
				},
			},
		},
	})
	mockConfigController.EXPECT().GetMultiClusterService(unknownRemote.Name, unknownRemote.Namespace).Return(nil)
	mockConfigController.EXPECT().GetMultiClusterService(knownRemote.Name, knownRemote.Namespace).Return(
		&v1alpha1.MultiClusterService{
			Spec: v1alpha1.MultiClusterServiceSpec{
				Ports: []v1alpha1.PortSpec{
					{
						Port:     8080,
						Protocol: "grpc",
					},
					{
						Port:     9090,
						Protocol: "tcp",
					},
				},
			},
		})

	client := NewClient(mockKubeController, mockConfigController, providerID, mockConfigurator)

	testCases := []struct {
		name          string
		svc           service.MeshService
		expectedPorts map[uint32]string
		expectedErr   error
	}{
		{
			name:          "unknown local returns error",
			svc:           unknownLocal,
			expectedPorts: nil,
			expectedErr:   errors.Wrapf(errServiceNotFound, "Error retrieving k8s service %s", unknownLocal),
		},
		{
			name:          "unknown remote returns error",
			svc:           unknownRemote,
			expectedPorts: nil,
			expectedErr:   fmt.Errorf("Error getting MultiClusterService for Service %s", unknownRemote),
		},
		{
			name: "known local returns mapping",
			svc:  knownLocal,
			expectedPorts: map[uint32]string{
				80: "http",
				90: "tcp",
			},
			expectedErr: nil,
		},
		{
			name: "known remote returns mapping",
			svc:  knownRemote,
			expectedPorts: map[uint32]string{
				8080: "grpc",
				9090: "tcp",
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mappings, err := client.GetPortToProtocolMappingForService(tc.svc)
			if tc.expectedErr != nil {
				assert.Error(err)
				assert.Equal(tc.expectedErr.Error(), err.Error())
			}
			assert.Equal(mappings, tc.expectedPorts)
		})
	}
}

func TestGetHostnamesForService(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(GinkgoT())
	mockKubeController := k8s.NewMockController(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigClient := config.NewMockController(mockCtrl)

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableMulticlusterMode: true,
	}).AnyTimes()
	mockKubeController.EXPECT().GetService(tests.BookbuyerService).Return(tests.NewServiceFixtureWithMultiplePorts(tests.BookbuyerServiceName, tests.Namespace, map[string]string{
		tests.SelectorKey: tests.SelectorValue,
	})).AnyTimes()

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{
		EnableMulticlusterMode: true,
	}).AnyTimes()

	mockConfigurator.EXPECT().GetClusterDomain().Return("cluster-x").AnyTimes()

	mockConfigClient.EXPECT().GetMultiClusterService(tests.BookbuyerClusterYService.Name,
		tests.BookbuyerClusterYService.Namespace).Return(tests.BookbuyerMCS).AnyTimes()
	testCases := []struct {
		name              string
		service           service.MeshService
		locality          service.Locality
		expectedHostnames []string
	}{
		{
			name:     "hostnames corresponding to a service in the same namespace",
			service:  tests.BookbuyerService,
			locality: service.LocalNS,
			expectedHostnames: []string{
				tests.BookbuyerServiceName,
				fmt.Sprintf("%s:8082", tests.BookbuyerServiceName),
				fmt.Sprintf("%s:9091", tests.BookbuyerServiceName),
				fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s:9091", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc:9091", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster:9091", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local:9091", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-x", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-x:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-x:9091", tests.BookbuyerServiceName, tests.Namespace),
			},
		},
		{
			name:     "hostnames corresponding to a service NOT in the same namespace",
			service:  tests.BookbuyerService,
			locality: service.LocalCluster,
			expectedHostnames: []string{
				fmt.Sprintf("%s.%s", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s:9091", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc:9091", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster:9091", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local:9091", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-x", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-x:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-x:9091", tests.BookbuyerServiceName, tests.Namespace),
			},
		},
		{
			name:     "hostnames corresponding to a local service from outside the local cluster",
			service:  tests.BookbuyerService,
			locality: service.RemoteCluster,
			expectedHostnames: []string{
				fmt.Sprintf("%s.%s.svc.cluster.cluster-x", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-x:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-x:9091", tests.BookbuyerServiceName, tests.Namespace),
			},
		},
		{
			name:     "hostnames corresponding to a remote service from the local cluster",
			service:  tests.BookbuyerClusterYService,
			locality: service.LocalCluster,
			expectedHostnames: []string{
				fmt.Sprintf("%s.%s.svc.cluster.cluster-y", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-y:8082", tests.BookbuyerServiceName, tests.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.cluster-y:9091", tests.BookbuyerServiceName, tests.Namespace),
			},
		},
	}
	c := Client{configurator: mockConfigurator, kubeController: mockKubeController, configClient: mockConfigClient}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log.Error().Msgf("running test %s", tc.name)

			actual := c.GetHostnamesForService(tc.service, tc.locality)
			assert.ElementsMatch(tc.expectedHostnames, actual)
			assert.Len(actual, len(tc.expectedHostnames))
		})
	}
}

func TestGetServiceByNameNamespace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	configClient := config.NewMockController(mockCtrl)

	mockKubeController.EXPECT().GetService(service.MeshService{
		Name:          "foo",
		Namespace:     "bar",
		ClusterDomain: constants.LocalDomain,
	}).Return(nil).AnyTimes()

	mockKubeController.EXPECT().GetService(service.MeshService{
		Name:          "baz",
		Namespace:     "qux",
		ClusterDomain: constants.LocalDomain,
	}).Return(&corev1.Service{}).AnyTimes()

	configClient.EXPECT().GetMultiClusterService("baz", "qux").Return(&v1alpha1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "baz",
			Namespace: "qux",
		},
		Spec: v1alpha1.MultiClusterServiceSpec{
			Clusters: []v1alpha1.ClusterSpec{
				{
					Name: "cluster-x",
				},
				{
					Name: "cluster-y",
				},
				{
					Name: "cluster-z",
				},
			},
		},
	}).AnyTimes()

	configClient.EXPECT().GetMultiClusterService("foo", "bar").Return(&v1alpha1.MultiClusterService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: v1alpha1.MultiClusterServiceSpec{
			Clusters: []v1alpha1.ClusterSpec{
				{
					Name: "cluster-a",
				},
				{
					Name: "cluster-b",
				},
				{
					Name: "cluster-c",
				},
			},
		},
	}).AnyTimes()

	type testCase struct {
		name         string
		svcName      string
		svcNamespace string
		expected     []service.MeshService
	}

	testCases := []testCase{
		{
			name:         "Global service with only remote backends",
			svcName:      "foo",
			svcNamespace: "bar",
			expected: []service.MeshService{
				{
					Name:          "foo",
					Namespace:     "bar",
					ClusterDomain: "cluster-a",
				},
				{
					Name:          "foo",
					Namespace:     "bar",
					ClusterDomain: "cluster-b",
				},
				{
					Name:          "foo",
					Namespace:     "bar",
					ClusterDomain: "cluster-c",
				},
			},
		},

		{
			name:         "Global service with remote and local backends",
			svcName:      "baz",
			svcNamespace: "qux",
			expected: []service.MeshService{
				{
					Name:          "baz",
					Namespace:     "qux",
					ClusterDomain: constants.LocalDomain,
				},
				{
					Name:          "baz",
					Namespace:     "qux",
					ClusterDomain: "cluster-x",
				},
				{
					Name:          "baz",
					Namespace:     "qux",
					ClusterDomain: "cluster-y",
				},
				{
					Name:          "baz",
					Namespace:     "qux",
					ClusterDomain: "cluster-z",
				},
			},
		},
	}

	provider := NewClient(mockKubeController, configClient, "fake-provider", nil)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			assert.Equal(tc.expected, provider.GetServicesByNameNamespace(tc.svcName, tc.svcNamespace))
		})
	}
}
