package kube

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

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

var _ = Describe("Test Kube Client Provider (w/o kubecontroller)", func() {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		mockConfigurator   *configurator.MockConfigurator
		client             *Client
	)
	const providerID = "provider"

	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
	mockConfigController := config.NewMockController(mockCtrl)

	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()

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
		mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()

		Expect(client.ListEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
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

	providerID := "test-provider"
	testNamespace := "testNamespace"
	meshName := "meshName"
	stop := make(chan struct{})

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

		services, err := client.GetServicesForServiceIdentity(tests.BookbuyerServiceIdentity)
		Expect(err).To(HaveOccurred())
		Expect(services).To(BeNil())

		err = fakeClientSet.CoreV1().Services(testNamespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return a service that matches the ServiceAccount associated with the Pod", func() {
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
		expectedMeshSvcs := []service.MeshService{expectedMeshSvc}
		Expect(meshSvcs).To(Equal(expectedMeshSvcs))

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsAndServiceChannel
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

		givenSvcAccount := identity.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Deployment spec above
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

		givenSvcAccount := identity.K8sServiceAccount{
			Namespace: testNamespace,
			Name:      "test-service-account", // Should match the service account in the Deployment spec above
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
			{Name: "test-1", Namespace: testNamespace},
			{Name: "test-2", Namespace: testNamespace},
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
	providerID := "provider"

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
			expectedEndpoints: []endpoint.Endpoint{{
				IP: net.ParseIP(tests.ServiceIP),
			}},
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
			expectedEndpoints: []endpoint.Endpoint{{
				IP: net.ParseIP(tests.ServiceIP),
			},
				{
					IP: net.ParseIP("9.9.9.9"),
				}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			kubeClient := testclient.NewSimpleClientset()
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockConfigController := config.NewMockController(mockCtrl)
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()

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

func TestGetMultiClusterServiceEndpointsForServiceAccount(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigController := config.NewMockController(mockCtrl)

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()
	providerID := "provider"
	provider := NewClient(mockKubeController, mockConfigController, providerID, mockConfigurator)

	destServiceIdentity := tests.BookstoreServiceIdentity
	destSA := destServiceIdentity.ToK8sServiceAccount()

	mcServices := []v1alpha1.MultiClusterService{
		{
			Spec: v1alpha1.MultiClusterServiceSpec{
				Clusters: []v1alpha1.ClusterSpec{
					{
						Address: "1.2.3.4:8080",
						Name:    "remote-cluster-1",
					},
					{
						Address: "5.6.7.8:8080",
						Name:    "remote-cluster-2",
					},
				},
				ServiceAccount: destSA.Name,
			},
		},
	}

	configClient := fakeConfig.NewSimpleClientset()
	for _, mcService := range mcServices {
		mcServicePtr := mcService
		_, err := configClient.ConfigV1alpha1().MultiClusterServices(tests.Namespace).Create(context.TODO(), &mcServicePtr, metav1.CreateOptions{})
		assert.Nil(err)
	}

	mockConfigController.EXPECT().GetMultiClusterServiceByServiceAccount(destSA.Name, destSA.Namespace).Return(mcServices).AnyTimes()

	endpoints := provider.getMultiClusterServiceEndpointsForServiceAccount(destSA.Name, destSA.Namespace)
	assert.Equal(len(endpoints), 2)

	assert.ElementsMatch(endpoints, []endpoint.Endpoint{
		{
			IP:   net.ParseIP("1.2.3.4"),
			Port: 8080,
		},
		{
			IP:   net.ParseIP("5.6.7.8"),
			Port: 8080,
		},
	})
}
