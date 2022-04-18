package kube

import (
	"context"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/config"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test Kube client Provider (w/o kubecontroller)", func() {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		mockConfigurator   *configurator.MockConfigurator
		c                  *client
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)
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
		mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()

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
		mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()

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

func TestGetServicesForServiceIdentity(t *testing.T) {
	testCases := []struct {
		name        string
		svcIdentity identity.ServiceIdentity
		pods        []*corev1.Pod
		services    []*corev1.Service
		expected    []service.MeshService
	}{
		{
			name:        "Returns the list of MeshServices matching the given identity",
			svcIdentity: identity.ServiceIdentity("sa1.ns1.cluster.local"), // Matches pod ns1/p1
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "p1",
						Labels: map[string]string{
							"k1": "v1", // matches selector for service ns1/s1
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "p2",
						Labels: map[string]string{
							"k1": "v2", // does not match selector for service ns1/s1
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa2",
					},
				},
			},
			services: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "s1",
						Namespace: "ns1",
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							"k1": "v1", // matches labels on pod ns1/p1
						},
						Ports: []corev1.ServicePort{{}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "s2",
						Namespace: "ns1",
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							"k1": "v2", // does not match labels on pod ns1/p1
						},
					},
				},
			},
			expected: []service.MeshService{
				{Namespace: "ns1", Name: "s1", Protocol: "http"}, // ns1/s1 matches pod ns1/p1 with service account ns1/sa1
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			c := &client{
				kubeController: mockKubeController,
			}

			mockKubeController.EXPECT().ListPods().Return(tc.pods)
			mockKubeController.EXPECT().ListServices().Return(tc.services)
			mockKubeController.EXPECT().GetEndpoints(gomock.Any()).Return(nil, nil).AnyTimes()

			actual := c.GetServicesForServiceIdentity(tc.svcIdentity)
			assert.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestListEndpointsForIdentity(t *testing.T) {
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
			assert := tassert.New(t)

			mockCtrl := gomock.NewController(t)
			kubeClient := testclient.NewSimpleClientset()
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockConfigController := config.NewMockController(mockCtrl)
			mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: false}).AnyTimes()

			provider := NewClient(mockKubeController, mockConfigController, mockConfigurator)

			var pods []*corev1.Pod
			for serviceIdentity, endpoints := range tc.outboundServiceAccountEndpoints {
				podlabels := map[string]string{
					constants.AppLabel:               tests.SelectorValue,
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

	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()
	provider := NewClient(mockKubeController, mockConfigController, mockConfigurator)

	destServiceIdentity := tests.BookstoreServiceIdentity
	destSA := destServiceIdentity.ToK8sServiceAccount()

	mcServices := []configv1alpha3.MultiClusterService{
		{
			Spec: configv1alpha3.MultiClusterServiceSpec{
				Clusters: []configv1alpha3.ClusterSpec{
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
		_, err := configClient.ConfigV1alpha2().MultiClusterServices(tests.Namespace).Create(context.TODO(), &mcServicePtr, metav1.CreateOptions{})
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
