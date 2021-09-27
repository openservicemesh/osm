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
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"

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
	k8sInterfaces "github.com/openservicemesh/osm/pkg/k8s/interfaces"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test Kube client Provider (w/o kubecontroller)", func() {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8sInterfaces.MockController
		mockConfigurator   *configurator.MockConfigurator
		c                  *client
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8sInterfaces.NewMockController(mockCtrl)
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
	mockConfigController := config.NewMockController(mockCtrl)

	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()

	BeforeEach(func() {
		c = NewClient(mockKubeController, mockConfigController, mockConfigurator)
	})

	It("tests GetID", func() {
		Expect(c.GetID()).To(Equal(providerName))
	})

	meshSvc := service.MeshService{
		Name:       "test",
		Namespace:  "default",
		TargetPort: 90,
	}

	It("should correctly return a list of endpoints for a service", func() {
		// Should be empty for now
		mockKubeController.EXPECT().GetEndpoints(meshSvc).Return(&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: meshSvc.Namespace,
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
							Port: int32(meshSvc.TargetPort), // Must match meshSvc.TargetPort
						},
						{
							Port: 8888, // Does not match meshSvc.TargetPort, should be ignored
						},
					},
				},
			},
		}, nil)
		mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()

		Expect(c.ListEndpointsForService(meshSvc)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: endpoint.Port(meshSvc.TargetPort),
			},
		}))
	})

	It("should not filter the endpoints of a MeshService whose TargetPort is not known", func() {
		svc := service.MeshService{
			Name:      "test",
			Namespace: "default",
			// No TargetPort
		}

		mockKubeController.EXPECT().GetEndpoints(svc).Return(&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: svc.Namespace,
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
							Port: 80,
						},
						{
							Port: 90,
						},
					},
				},
			},
		}, nil)
		mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()

		Expect(c.ListEndpointsForService(svc)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: 80,
			},
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: 90,
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

		Expect(c.GetResolvableEndpointsForService(tests.BookbuyerService)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(192, 168, 0, 1),
				Port: tests.ServicePort,
			},
		}))
	})

	It("GetResolvableEndpoints should properly return actual endpoints without ClusterIP when ClusterIP is not set", func() {
		// Expect the individual pod endpoints, when no cluster IP is assigned to the service
		mockKubeController.EXPECT().GetService(meshSvc).Return(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      meshSvc.Name,
				Namespace: meshSvc.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:     "servicePort",
					Protocol: corev1.ProtocolTCP,
					Port:     int32(meshSvc.Port),
				}},
				Selector: map[string]string{
					"some-label": "test",
				},
			},
		})

		mockKubeController.EXPECT().GetEndpoints(meshSvc).Return(&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: meshSvc.Namespace,
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
							Port:     int32(meshSvc.TargetPort),
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
		}, nil)

		Expect(c.GetResolvableEndpointsForService(meshSvc)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: endpoint.Port(meshSvc.TargetPort),
			},
		}))
	})

	It("GetResolvableEndpoints should properly return actual endpoints when ClusterIP is none", func() {

		// If the service has cluster IP set to none, expect the individual pod endpoints
		mockKubeController.EXPECT().GetService(meshSvc).Return(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      meshSvc.Name,
				Namespace: meshSvc.Namespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: corev1.ClusterIPNone,
				Ports: []corev1.ServicePort{{
					Name:       "servicePort",
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(meshSvc.Port),
					TargetPort: intstr.FromInt(int(meshSvc.TargetPort)),
				}},
				Selector: map[string]string{
					"some-label": "test",
				},
			},
		})

		mockKubeController.EXPECT().GetEndpoints(meshSvc).Return(&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: meshSvc.Namespace,
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
							Port:     int32(meshSvc.TargetPort),
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
		}, nil)

		Expect(c.GetResolvableEndpointsForService(meshSvc)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: endpoint.Port(meshSvc.TargetPort),
			},
		}))
	})
})

var _ = Describe("Test Kube client Provider (/w kubecontroller)", func() {
	var (
		mockCtrl         *gomock.Controller
		kubeController   k8sInterfaces.Controller
		mockConfigurator *configurator.MockConfigurator
		fakeClientSet    *testclient.Clientset
		c                *client
		err              error
	)
	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
	mockConfigController := config.NewMockController(mockCtrl)

	testNamespace := "testNamespace"
	meshName := "meshName"
	stop := make(chan struct{})

	BeforeEach(func() {
		fakeClientSet = testclient.NewSimpleClientset()
		kubeController, err = k8s.NewKubernetesController(fakeClientSet, nil, meshName, stop)

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

		c = NewClient(kubeController, mockConfigController, mockConfigurator)
		Expect(c).ToNot(BeNil())
	})

	AfterEach(func() {
		stop <- struct{}{}
	})

	It("should return an empty list when a pod matching the selector doesn't exist", func() {
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

		services := c.GetServicesForServiceIdentity(tests.BookbuyerServiceIdentity)
		Expect(services).To(HaveLen(0))

		err = fakeClientSet.CoreV1().Services(testNamespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return a service that matches the ServiceAccount associated with the Pod", func() {
		podsAndServiceChannel := events.Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated,
			announcements.ServiceAdded,
			announcements.ServiceDeleted,
			announcements.ServiceUpdated,
		)
		defer events.Unsub(podsAndServiceChannel)

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

		// Expect MeshServices that matches the Pod spec labels
		expectedMeshervices := []service.MeshService{
			{Namespace: svc.Namespace, Name: svc.Name, Port: uint16(svc.Spec.Ports[0].Port), Protocol: constants.ProtocolHTTP},
		}

		meshSvcs := c.GetServicesForServiceIdentity(givenSvcAccount.ToServiceIdentity())
		expectedMeshSvcs := expectedMeshervices
		Expect(meshSvcs).To(Equal(expectedMeshSvcs))

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsAndServiceChannel
	})

	It("should return an an empty list when the Service selector doesn't match the pod", func() {
		podsChannel := events.Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated)
		defer events.Unsub(podsChannel)

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

		// Expect no matching MeshService objects
		svcs := c.GetServicesForServiceIdentity(givenSvcAccount.ToServiceIdentity())
		Expect(svcs).To(HaveLen(0))

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel
	})

	It("should return an empty list when the service doesn't have a selector", func() {
		podsChannel := events.Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated)
		defer events.Unsub(podsChannel)

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

		// Expect no matching MeshService objects
		svcs := c.GetServicesForServiceIdentity(givenSvcAccount.ToServiceIdentity())
		Expect(svcs).To(HaveLen(0))

		err = fakeClientSet.CoreV1().Pods(testNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
		<-podsChannel
	})

	It("should return all services when multiple services match the same Pod", func() {
		// This test is meant to ensure the
		// service selector logic works as expected when multiple services
		// have the same selector match.
		podsAndServiceChannel := events.Subscribe(announcements.PodAdded,
			announcements.PodDeleted,
			announcements.PodUpdated,
			announcements.ServiceAdded,
			announcements.ServiceDeleted,
			announcements.ServiceUpdated,
		)
		defer events.Unsub(podsAndServiceChannel)

		// Create a Service
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-1",
				Namespace: testNamespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Name:        "servicePort",
					Protocol:    corev1.ProtocolTCP,
					Port:        tests.ServicePort,
					AppProtocol: pointer.StringPtr("http"),
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

		meshServices := c.GetServicesForServiceIdentity(givenSvcAccount.ToServiceIdentity())
		expectedServices := []service.MeshService{
			{Name: "test-1", Namespace: testNamespace, Port: tests.ServicePort, Protocol: "http"},
			{Name: "test-2", Namespace: testNamespace, Port: tests.ServicePort, Protocol: "http"},
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

			mockKubeController := k8sInterfaces.NewMockController(mockCtrl)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockConfigController := config.NewMockController(mockCtrl)
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()

			provider := NewClient(mockKubeController, mockConfigController, mockConfigurator)

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

	mockKubeController := k8sInterfaces.NewMockController(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigController := config.NewMockController(mockCtrl)

	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()
	provider := NewClient(mockKubeController, mockConfigController, mockConfigurator)

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
						Address:  "5.6.7.8:8080",
						Name:     "remote-cluster-2",
						Weight:   1,
						Priority: 2,
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
			IP:       net.ParseIP("1.2.3.4"),
			Port:     8080,
			Weight:   0,
			Priority: 0,
			Zone:     "remote-cluster-1",
		},
		{
			IP:       net.ParseIP("5.6.7.8"),
			Port:     8080,
			Weight:   1,
			Priority: 2,
			Zone:     "remote-cluster-2",
		},
	})
}
