package kube

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test Kube client Provider (w/o kubecontroller)", func() {
	var (
		mockCtrl           *gomock.Controller
		mockKubeController *k8s.MockController
		c                  *client
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockKubeController = k8s.NewMockController(mockCtrl)

	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()

	BeforeEach(func() {
		c = NewClient(mockKubeController)
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

		Expect(c.ListEndpointsForService(meshSvc)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: endpoint.Port(meshSvc.TargetPort),
			},
		}))
	})

	It("should correctly filter endpoints for a headless service pod endpoint", func() {
		subdomainedSvc := service.MeshService{
			Name:       "subdomain-0.test",
			Namespace:  "default",
			TargetPort: 90,
		}
		// Should be empty for now
		mockKubeController.EXPECT().GetEndpoints(subdomainedSvc).Return(&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: subdomainedSvc.Namespace,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP:       "1.1.1.1",
							Hostname: "subdomain-0",
						},
						{
							IP:       "8.8.8.8",
							Hostname: "subdomain-1",
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Port: int32(subdomainedSvc.TargetPort), // Must match subdomainedSvc.TargetPort
						},
						{
							Port: 8888, // Does not match subdomainedSvc.TargetPort, should be ignored
						},
					},
				},
			},
		}, nil)

		Expect(c.ListEndpointsForService(subdomainedSvc)).To(Equal([]endpoint.Endpoint{
			{
				IP:   net.IPv4(1, 1, 1, 1),
				Port: endpoint.Port(subdomainedSvc.TargetPort),
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

			provider := NewClient(mockKubeController)

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
				_, err := kubeClient.CoreV1().Pods(sa.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
				assert.Nil(err)
				pods = append(pods, pod)
			}
			mockKubeController.EXPECT().ListPods().Return(pods).AnyTimes()

			actual := provider.ListEndpointsForIdentity(tc.serviceAccount)
			assert.NotNil(actual)
			assert.ElementsMatch(actual, tc.expectedEndpoints)
		})
	}
}

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
			svcIdentity: identity.ServiceIdentity("sa1.ns1"), // Matches pod ns1/p1
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
			stop := make(chan struct{})
			defer close(stop)

			objs := make([]runtime.Object, 0, len(tc.pods)+len(tc.services))

			namespaces := make(map[string]interface{})
			for _, pod := range tc.pods {
				objs = append(objs, pod)
				namespaces[pod.Namespace] = nil
			}
			for _, svc := range tc.services {
				objs = append(objs, svc)
				namespaces[svc.Namespace] = nil
			}
			for ns := range namespaces {
				objs = append(objs, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: ns,
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: "test-mesh",
						},
					},
				})
			}
			testClient := testclient.NewSimpleClientset(objs...)
			ic, err := informers.NewInformerCollection("test-mesh", stop, informers.WithKubeClient(testClient))
			assert.NoError(err)
			c := &client{
				kubeController: k8s.NewClient("osm-ns", ic, nil, messaging.NewBroker(stop)),
			}
			actual := c.GetServicesForServiceIdentity(tc.svcIdentity)
			assert.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestGetHostnamesForServicePort(t *testing.T) {
	testCases := []struct {
		name              string
		service           service.MeshService
		localNamespace    bool
		expectedHostnames []string
	}{
		{
			name:           "hostnames corresponding to a service in the same namespace",
			service:        service.MeshService{Namespace: "ns1", Name: "s1", Port: 90},
			localNamespace: true,
			expectedHostnames: []string{
				"s1",
				"s1:90",
				"s1.ns1",
				"s1.ns1:90",
				"s1.ns1.svc",
				"s1.ns1.svc:90",
				"s1.ns1.svc.cluster",
				"s1.ns1.svc.cluster:90",
				"s1.ns1.svc.cluster.local",
				"s1.ns1.svc.cluster.local:90",
			},
		},
		{
			name:           "hostnames corresponding to a service in different namespace",
			service:        service.MeshService{Namespace: "ns1", Name: "s1", Port: 90},
			localNamespace: false,
			expectedHostnames: []string{
				"s1.ns1",
				"s1.ns1:90",
				"s1.ns1.svc",
				"s1.ns1.svc:90",
				"s1.ns1.svc.cluster",
				"s1.ns1.svc.cluster:90",
				"s1.ns1.svc.cluster.local",
				"s1.ns1.svc.cluster.local:90",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := (&client{}).GetHostnamesForService(tc.service, tc.localNamespace)
			assert.ElementsMatch(actual, tc.expectedHostnames)
			assert.Len(actual, len(tc.expectedHostnames))
		})
	}
}

func TestIsMetricsEnabled(t *testing.T) {
	testCases := []struct {
		name        string
		informerOpt informers.InformerCollectionOption

		pod           *corev1.Pod
		expectEnabled bool
		expectErr     bool
	}{
		{
			name: "pod without prometheus scraping annotation",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
			expectEnabled: false,
		},
		{
			name: "pod with prometheus scraping annotation set to true",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						constants.PrometheusScrapeAnnotation: "true",
					},
				},
			},
			expectEnabled: true,
		},
		{
			name: "pod with prometheus scraping annotation set to false",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						constants.PrometheusScrapeAnnotation: "false",
					},
				},
			},
			expectEnabled: false,
		},
		{
			name: "pod with incorrect prometheus scraping annotation",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						constants.PrometheusScrapeAnnotation: "no",
					},
				},
			},
			expectEnabled: false,
			expectErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			k := k8s.NewMockController(mockCtrl)
			if tc.pod != nil {
				k.EXPECT().GetPodForProxy(gomock.Any()).Return(tc.pod, nil)
			} else {
				k.EXPECT().GetPodForProxy(gomock.Any()).Return(nil, errors.New("not found"))
			}
			c := NewClient(k)

			actual, err := c.IsMetricsEnabled(&envoy.Proxy{})
			assert.Equal(tc.expectEnabled, actual)
			if tc.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
