package kube

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	fakePolicyClient "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"

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

var testMeshName = "mesh"

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

	meshSvc := service.MeshService{
		Name:       "test",
		Namespace:  "default",
		TargetPort: 90,
	}

	It("should correctly return a list of endpoints for a service", func() {
		// Should be empty for now
		mockKubeController.EXPECT().GetEndpoints(meshSvc.Name, meshSvc.Namespace).Return(&corev1.Endpoints{
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
			Name:       "test",
			Subdomain:  "subdomain-0",
			Namespace:  "default",
			TargetPort: 90,
		}
		// Should be empty for now
		mockKubeController.EXPECT().GetEndpoints(subdomainedSvc.Name, subdomainedSvc.Namespace).Return(&corev1.Endpoints{
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

		mockKubeController.EXPECT().GetEndpoints(svc.Name, svc.Namespace).Return(&corev1.Endpoints{
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
		mockKubeController.EXPECT().GetService(tests.BookbuyerService.Name, tests.BookbuyerService.Namespace).Return(&corev1.Service{
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
		mockKubeController.EXPECT().GetService(meshSvc.Name, meshSvc.Namespace).Return(&corev1.Service{
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

		mockKubeController.EXPECT().GetEndpoints(meshSvc.Name, meshSvc.Namespace).Return(&corev1.Endpoints{
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
		mockKubeController.EXPECT().GetService(meshSvc.Name, meshSvc.Namespace).Return(&corev1.Service{
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

		mockKubeController.EXPECT().GetEndpoints(meshSvc.Name, meshSvc.Namespace).Return(&corev1.Endpoints{
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
				kubeController: k8s.NewClient("osm-ns", tests.OsmMeshConfigName, ic, nil, nil, messaging.NewBroker(stop)),
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

func TestListServicesForProxy(t *testing.T) {
	goodUUID := uuid.New()
	badUUID := uuid.New()
	testCases := []struct {
		name      string
		endpoints identity.ServiceIdentity
		pods      []*corev1.Pod
		proxy     *envoy.Proxy
		services  []*corev1.Service
		expected  []service.MeshService
		expectErr bool
	}{
		{
			name:  "Returns the list of MeshServices matching the given pod",
			proxy: envoy.NewProxy(envoy.KindSidecar, goodUUID, identity.New("sa1", "ns1"), &net.IPAddr{}, 1),
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "p1",
						Labels: map[string]string{
							constants.EnvoyUniqueIDLabelName: goodUUID.String(),
							"k1":                             "v1", // matches selector for service ns1/s1
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
							constants.EnvoyUniqueIDLabelName: badUUID.String(),
							"k1":                             "v2", // does not match selector for service ns1/s1
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa1",
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
							"k1": "v1",
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
							"k2": "v2", // does not match labels on pod ns1/p1
						},
						Ports: []corev1.ServicePort{{}},
					},
				},
			},
			expected: []service.MeshService{
				{Namespace: "ns1", Name: "s1", Protocol: "http"}, // ns1/s1 matches pod ns1/p1 with service account ns1/sa1
			},
		},
		{
			name:  "No matching services found",
			proxy: envoy.NewProxy(envoy.KindSidecar, goodUUID, identity.New("sa1", "ns1"), &net.IPAddr{}, 1),
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "p2",
						Labels: map[string]string{
							constants.EnvoyUniqueIDLabelName: goodUUID.String(),
							"k3":                             "v1", // matches for service ns1/s1
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "sa1",
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
							"k2": "v2", // does not match labels on pod ns1/p1
						},
						Ports: []corev1.ServicePort{{}},
					},
				},
			},
		},
		{
			name:  "Error: pod not found",
			proxy: envoy.NewProxy(envoy.KindSidecar, goodUUID, identity.New("sa1", "ns1"), &net.IPAddr{}, 1),
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "p2",
						Labels: map[string]string{
							constants.EnvoyUniqueIDLabelName: badUUID.String(),
							"k1":                             "v1", // matches for service ns1/s1
							"k2":                             "v2", // does not match selector for service ns1/s2
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
							"k2": "v2", // does not match labels on pod ns1/p1
						},
					},
				},
			},
			expectErr: true,
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
				namespaces[pod.Namespace] = true
			}
			for _, svc := range tc.services {
				objs = append(objs, svc)
				namespaces[svc.Namespace] = true
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
				kubeController: k8s.NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, ic, nil, nil, messaging.NewBroker(stop)),
			}
			actual, err := c.ListServicesForProxy(tc.proxy)
			assert.ElementsMatch(tc.expected, actual)
			if tc.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestDetectIngressBackendConflicts(t *testing.T) {
	testCases := []struct {
		name              string
		x                 policyv1alpha1.IngressBackend
		y                 policyv1alpha1.IngressBackend
		conflictsExpected int
	}{
		{
			name: "single backend conflict",
			x: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			y: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-2",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			conflictsExpected: 1,
		},
		{
			name: "Unique backends per policy",
			x: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			y: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-2",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend2",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			conflictsExpected: 0,
		},
		{
			name: "multiple backends conflict",
			x: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
						{
							Name: "backend2",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			y: policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-2",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
						{
							Name: "backend2",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
			conflictsExpected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			conflicts := DetectIngressBackendConflicts(tc.x, tc.y)
			a.Len(conflicts, tc.conflictsExpected)
		})
	}
}

func TestListEgressPoliciesForSourceAccount(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockKubeController.EXPECT().IsMonitoredNamespace("test").Return(true).AnyTimes()

	testCases := []struct {
		name             string
		allEgresses      []*policyv1alpha1.Egress
		source           identity.K8sServiceAccount
		expectedEgresses []*policyv1alpha1.Egress
	}{
		{
			name: "matching egress policy not found for source identity test/sa-3",
			allEgresses: []*policyv1alpha1.Egress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "egress-1",
						Namespace: "test",
					},
					Spec: policyv1alpha1.EgressSpec{
						Sources: []policyv1alpha1.EgressSourceSpec{
							{
								Kind:      "ServiceAccount",
								Name:      "sa-1",
								Namespace: "test",
							},
							{
								Kind:      "ServiceAccount",
								Name:      "sa-2",
								Namespace: "test",
							},
						},
						Hosts: []string{"foo.com"},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   80,
								Protocol: "http",
							},
						},
					},
				},
			},
			source:           identity.K8sServiceAccount{Name: "sa-3", Namespace: "test"},
			expectedEgresses: nil,
		},
		{
			name: "matching egress policy found for source identity test/sa-1",
			allEgresses: []*policyv1alpha1.Egress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "egress-1",
						Namespace: "test",
					},
					Spec: policyv1alpha1.EgressSpec{
						Sources: []policyv1alpha1.EgressSourceSpec{
							{
								Kind:      "ServiceAccount",
								Name:      "sa-1",
								Namespace: "test",
							},
							{
								Kind:      "ServiceAccount",
								Name:      "sa-2",
								Namespace: "test",
							},
						},
						Hosts: []string{"foo.com"},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   80,
								Protocol: "http",
							},
						},
					},
				},
			},
			source: identity.K8sServiceAccount{Name: "sa-1", Namespace: "test"},
			expectedEgresses: []*policyv1alpha1.Egress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "egress-1",
						Namespace: "test",
					},
					Spec: policyv1alpha1.EgressSpec{
						Sources: []policyv1alpha1.EgressSourceSpec{
							{
								Kind:      "ServiceAccount",
								Name:      "sa-1",
								Namespace: "test",
							},
							{
								Kind:      "ServiceAccount",
								Name:      "sa-2",
								Namespace: "test",
							},
						},
						Hosts: []string{"foo.com"},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   80,
								Protocol: "http",
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			ic, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)

			c := NewClient(mockKubeController)
			a.Nil(err)
			a.NotNil(c)

			// Create fake egress policies
			for _, egressPolicy := range tc.allEgresses {
				_ = ic.Add(informers.InformerKeyEgress, egressPolicy, t)
			}

			mockKubeController.EXPECT().ListEgressPolicies().Return(tc.allEgresses).AnyTimes()
			actual := c.ListEgressPoliciesForServiceAccount(tc.source)
			a.ElementsMatch(tc.expectedEgresses, actual)
		})
	}
}

func TestGetIngressBackendPolicy(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockKubeController.EXPECT().IsMonitoredNamespace("test").Return(true).AnyTimes()

	testCases := []struct {
		name                   string
		allResources           []*policyv1alpha1.IngressBackend
		backend                service.MeshService
		expectedIngressBackend *policyv1alpha1.IngressBackend
	}{
		{
			name: "IngressBackend policy found",
			allResources: []*policyv1alpha1.IngressBackend{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-1",
						Namespace: "test",
					},
					Spec: policyv1alpha1.IngressBackendSpec{
						Backends: []policyv1alpha1.BackendSpec{
							{
								Name: "backend1", // matches the backend specified in the test case
								Port: policyv1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
							{
								Name: "backend2",
								Port: policyv1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
						Sources: []policyv1alpha1.IngressSourceSpec{
							{
								Kind:      "Service",
								Name:      "client",
								Namespace: "foo",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-2",
						Namespace: "test",
					},
					Spec: policyv1alpha1.IngressBackendSpec{
						Backends: []policyv1alpha1.BackendSpec{
							{
								Name: "backend3", // does not match the backend specified in the test case
								Port: policyv1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
						Sources: []policyv1alpha1.IngressSourceSpec{
							{
								Kind:      "Service",
								Name:      "client",
								Namespace: "foo",
							},
						},
					},
				},
			},
			backend: service.MeshService{Name: "backend1", Namespace: "test", TargetPort: 80, Protocol: "http"},
			expectedIngressBackend: &policyv1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "test",
				},
				Spec: policyv1alpha1.IngressBackendSpec{
					Backends: []policyv1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
						{
							Name: "backend2",
							Port: policyv1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyv1alpha1.IngressSourceSpec{
						{
							Kind:      "Service",
							Name:      "client",
							Namespace: "foo",
						},
					},
				},
			},
		},
		{
			name: "IngressBackend policy namespace does not match MeshService.Namespace",
			allResources: []*policyv1alpha1.IngressBackend{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-1",
						Namespace: "test",
					},
					Spec: policyv1alpha1.IngressBackendSpec{
						Backends: []policyv1alpha1.BackendSpec{
							{
								Name: "backend1", // matches the backend specified in the test case
								Port: policyv1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
							{
								Name: "backend2",
								Port: policyv1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
						Sources: []policyv1alpha1.IngressSourceSpec{
							{
								Kind:      "Service",
								Name:      "client",
								Namespace: "foo",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-2",
						Namespace: "test",
					},
					Spec: policyv1alpha1.IngressBackendSpec{
						Backends: []policyv1alpha1.BackendSpec{
							{
								Name: "backend2", // does not match the backend specified in the test case
								Port: policyv1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
						Sources: []policyv1alpha1.IngressSourceSpec{
							{
								Kind:      "Service",
								Name:      "client",
								Namespace: "foo",
							},
						},
					},
				},
			},
			backend:                service.MeshService{Name: "backend1", Namespace: "test-1"}, // Namespace does not match IngressBackend.Namespace
			expectedIngressBackend: nil,
		},
		{
			name:                   "IngressBackend policy not found",
			allResources:           []*policyv1alpha1.IngressBackend{},
			backend:                service.MeshService{Name: "backend1", Namespace: "test"},
			expectedIngressBackend: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			ic, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)

			c := NewClient(mockKubeController)
			a.Nil(err)
			a.NotNil(c)

			// Create fake ingress backends
			for _, ingressBackend := range tc.allResources {
				_ = ic.Add(informers.InformerKeyIngressBackend, ingressBackend, t)
			}

			mockKubeController.EXPECT().ListIngressBackendPolicies().Return(tc.allResources).AnyTimes()

			actual := c.GetIngressBackendPolicyForService(tc.backend)
			a.Equal(tc.expectedIngressBackend, actual)
		})
	}
}

func TestListRetryPolicy(t *testing.T) {
	var thresholdUintVal uint32 = 3
	thresholdTimeoutDuration := metav1.Duration{Duration: time.Duration(5 * time.Second)}
	thresholdBackoffDuration := metav1.Duration{Duration: time.Duration(1 * time.Second)}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockKubeController.EXPECT().IsMonitoredNamespace("test").Return(true).AnyTimes()

	testCases := []struct {
		name            string
		allRetries      []*policyv1alpha1.Retry
		source          identity.K8sServiceAccount
		expectedRetries []*policyv1alpha1.Retry
	}{
		{
			name: "matching retry policy not found for source identity test/sa-3",
			allRetries: []*policyv1alpha1.Retry{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "retry-1",
						Namespace: "test",
					},
					Spec: policyv1alpha1.RetrySpec{
						Source: policyv1alpha1.RetrySrcDstSpec{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "test",
						},
						Destinations: []policyv1alpha1.RetrySrcDstSpec{
							{
								Kind:      "Service",
								Name:      "s1",
								Namespace: "test",
							},
							{
								Kind:      "Service",
								Name:      "s2",
								Namespace: "test",
							},
						},
						RetryPolicy: policyv1alpha1.RetryPolicySpec{
							RetryOn:                  "",
							NumRetries:               &thresholdUintVal,
							PerTryTimeout:            &thresholdTimeoutDuration,
							RetryBackoffBaseInterval: &thresholdBackoffDuration,
						},
					},
				},
			},
			source:          identity.K8sServiceAccount{Name: "sa-3", Namespace: "test"},
			expectedRetries: nil,
		},
		{
			name: "matching retry policy found for source identity test/sa-1",
			allRetries: []*policyv1alpha1.Retry{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "retry-1",
						Namespace: "test",
					},
					Spec: policyv1alpha1.RetrySpec{
						Source: policyv1alpha1.RetrySrcDstSpec{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "test",
						},
						Destinations: []policyv1alpha1.RetrySrcDstSpec{
							{
								Kind:      "Service",
								Name:      "s1",
								Namespace: "test",
							},
							{
								Kind:      "Service",
								Name:      "s2",
								Namespace: "test",
							},
						},
						RetryPolicy: policyv1alpha1.RetryPolicySpec{
							RetryOn:                  "",
							NumRetries:               &thresholdUintVal,
							PerTryTimeout:            &thresholdTimeoutDuration,
							RetryBackoffBaseInterval: &thresholdBackoffDuration,
						},
					},
				},
			},
			source: identity.K8sServiceAccount{Name: "sa-1", Namespace: "test"},
			expectedRetries: []*policyv1alpha1.Retry{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "retry-1",
						Namespace: "test",
					},
					Spec: policyv1alpha1.RetrySpec{
						Source: policyv1alpha1.RetrySrcDstSpec{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "test",
						},
						Destinations: []policyv1alpha1.RetrySrcDstSpec{
							{
								Kind:      "Service",
								Name:      "s1",
								Namespace: "test",
							},
							{
								Kind:      "Service",
								Name:      "s2",
								Namespace: "test",
							},
						},
						RetryPolicy: policyv1alpha1.RetryPolicySpec{
							RetryOn:                  "",
							NumRetries:               &thresholdUintVal,
							PerTryTimeout:            &thresholdTimeoutDuration,
							RetryBackoffBaseInterval: &thresholdBackoffDuration,
						},
					},
				},
			},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			ic, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)

			c := NewClient(mockKubeController)
			a.Nil(err)
			a.NotNil(c)

			// Create fake retry policies
			for _, retryPolicy := range tc.allRetries {
				err := ic.Add(informers.InformerKeyRetry, retryPolicy, t)
				a.Nil(err)
			}

			mockKubeController.EXPECT().ListRetryPolicies().Return(tc.allRetries).AnyTimes()

			actual := c.ListRetryPoliciesForServiceAccount(tc.source)
			a.ElementsMatch(tc.expectedRetries, actual)
		})
	}
}

func TestGetProxyStatsHeaders(t *testing.T) {
	uuid1 := uuid.New()
	tr := true
	const namespace = "ns1"
	testCases := []struct {
		name      string
		proxy     *envoy.Proxy
		pod       *corev1.Pod
		expected  map[string]string
		expectErr bool
	}{
		{
			name:      "pod not found",
			proxy:     envoy.NewProxy(envoy.KindSidecar, uuid1, identity.New("sa1", namespace), &net.IPAddr{}, 1),
			expectErr: true,
		},
		{
			name:  "pod has bad service account",
			proxy: envoy.NewProxy(envoy.KindSidecar, uuid1, identity.New("sa1", namespace), &net.IPAddr{}, 1),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uuid1.String(),
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa2",
				},
			},
			expectErr: true,
		},
		{
			name:  "full stats headers from deployment",
			proxy: envoy.NewProxy(envoy.KindSidecar, uuid1, identity.New("sa1", namespace), &net.IPAddr{}, 1),

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: namespace,
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uuid1.String(),
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: "bad-controller",
							Kind: "Invalid",
						},
						{
							Name:       "good-dep",
							Kind:       "Deployment",
							Controller: &tr,
						},
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa1",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "Deployment",
				"osm-stats-name":      "good-dep",
				"osm-stats-namespace": namespace,
				"osm-stats-pod":       "pod",
			},
		},
		{
			name:  "no owner references",
			proxy: envoy.NewProxy(envoy.KindSidecar, uuid1, identity.New("sa1", namespace), &net.IPAddr{}, 1),

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: namespace,
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uuid1.String(),
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: "bad-controller",
							Kind: "Invalid",
						},
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa1",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "unknown",
				"osm-stats-name":      "unknown",
				"osm-stats-namespace": namespace,
				"osm-stats-pod":       "pod",
			},
		},
		{
			name:  "full stats headers from replicaset with hyphen",
			proxy: envoy.NewProxy(envoy.KindSidecar, uuid1, identity.New("sa1", namespace), &net.IPAddr{}, 1),

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: namespace,
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uuid1.String(),
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: "bad-controller",
							Kind: "Invalid",
						},
						{
							Name:       "good-controller",
							Kind:       "ReplicaSet",
							Controller: &tr,
						},
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa1",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "Deployment",
				"osm-stats-name":      "good",
				"osm-stats-namespace": namespace,
				"osm-stats-pod":       "pod",
			},
		},
		{
			name:  "full stats headers from replicaset without hyphen",
			proxy: envoy.NewProxy(envoy.KindSidecar, uuid1, identity.New("sa1", namespace), &net.IPAddr{}, 1),

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: namespace,
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: uuid1.String(),
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "goodcontroller",
							Kind:       "ReplicaSet",
							Controller: &tr,
						},
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "sa1",
				},
			},
			expected: map[string]string{
				"osm-stats-kind":      "ReplicaSet",
				"osm-stats-name":      "goodcontroller",
				"osm-stats-namespace": namespace,
				"osm-stats-pod":       "pod",
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			objects := []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{
						constants.OSMKubeResourceMonitorAnnotation: tests.MeshName,
					}},
				},
			}
			if test.pod != nil {
				objects = append(objects, test.pod)
			}
			fakeClient := fake.NewSimpleClientset(objects...)

			stop := make(chan struct{})
			defer close(stop)

			informer, err := informers.NewInformerCollection(tests.MeshName, stop,
				informers.WithKubeClient(fakeClient),
			)
			assert.NoError(err)
			controller := k8s.NewClient("ns", "", informer, nil, nil, messaging.NewBroker(stop))
			c := NewClient(controller)
			actual, err := c.GetProxyStatsHeaders(test.proxy)
			if test.expectErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			assert.Equal(test.expected, actual)
		})
	}
}

func TestGetUpstreamTrafficSettingByService(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockKubeController.EXPECT().IsMonitoredNamespace("test").Return(true).AnyTimes()

	testCases := []struct {
		name         string
		allResources []*policyv1alpha1.UpstreamTrafficSetting
		service      *service.MeshService
		expected     *policyv1alpha1.UpstreamTrafficSetting
	}{
		{
			name: "MeshService has matching UpstreamTrafficSetting",
			allResources: []*policyv1alpha1.UpstreamTrafficSetting{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u1",
						Namespace: "ns1",
					},
					Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
						Host: "s1.ns1.svc.cluster.local",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u2",
						Namespace: "ns1",
					},
					Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
						Host: "s2.ns1.svc.cluster.local",
					},
				},
			},
			service: &service.MeshService{Name: "s1", Namespace: "ns1"},
			expected: &policyv1alpha1.UpstreamTrafficSetting{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "u1",
					Namespace: "ns1",
				},
				Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
					Host: "s1.ns1.svc.cluster.local",
				},
			},
		},
		{
			name: "MeshService that does not match any UpstreamTrafficSetting",
			allResources: []*policyv1alpha1.UpstreamTrafficSetting{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u1",
						Namespace: "ns1",
					},
					Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
						Host: "s1.ns1.svc.cluster.local",
					},
				},
			},
			service:  &service.MeshService{Name: "s3", Namespace: "ns1"},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			ic, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)

			c := NewClient(mockKubeController)
			a.Nil(err)
			a.NotNil(c)

			// Create fake egress policies
			for _, resource := range tc.allResources {
				_ = ic.Add(informers.InformerKeyUpstreamTrafficSetting, resource, t)
			}

			mockKubeController.EXPECT().ListUpstreamTrafficSettings().Return(tc.allResources).AnyTimes()

			actual := c.GetUpstreamTrafficSettingByService(tc.service)
			a.Equal(tc.expected, actual)
		})
	}
}

func TestGetUpstreamTrafficSettingByNamespace(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockKubeController.EXPECT().IsMonitoredNamespace("ns1").Return(true).AnyTimes()

	name1 := &types.NamespacedName{Namespace: "ns1", Name: "u1"}
	name2 := &types.NamespacedName{Namespace: "ns1", Name: "u2"}
	name3 := &types.NamespacedName{Namespace: "ns1", Name: "u3"}
	resource1 := &policyv1alpha1.UpstreamTrafficSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u1",
			Namespace: "ns1",
		},
		Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
			Host: "s1.ns1.svc.cluster.local",
		},
	}
	resource2 := &policyv1alpha1.UpstreamTrafficSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u2",
			Namespace: "ns1",
		},
		Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
			Host: "s2.ns1.svc.cluster.local",
		},
	}

	testCases := []struct {
		name         string
		allResources []*policyv1alpha1.UpstreamTrafficSetting
		namespace    *types.NamespacedName
		expected     *policyv1alpha1.UpstreamTrafficSetting
	}{
		{
			name:         "UpstreamTrafficSetting namespaced name found",
			allResources: []*policyv1alpha1.UpstreamTrafficSetting{resource1, resource2},
			namespace:    name1,
			expected:     resource1,
		},
		{
			name:         "UpstreamTrafficSetting namespaced name not found",
			allResources: []*policyv1alpha1.UpstreamTrafficSetting{resource1},
			namespace:    name3,
			expected:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			ic, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)

			c := NewClient(mockKubeController)
			a.Nil(err)
			a.NotNil(c)

			// Create fake egress policies
			for _, resource := range tc.allResources {
				_ = ic.Add(informers.InformerKeyUpstreamTrafficSetting, resource, t)
			}

			mockKubeController.EXPECT().ListUpstreamTrafficSettings().Return(tc.allResources).AnyTimes()
			mockKubeController.EXPECT().GetUpstreamTrafficSetting(name1).Return(resource1).AnyTimes()
			mockKubeController.EXPECT().GetUpstreamTrafficSetting(name2).Return(resource2).AnyTimes()
			mockKubeController.EXPECT().GetUpstreamTrafficSetting(name3).Return(nil).AnyTimes()

			actual := c.GetUpstreamTrafficSettingByNamespace(tc.namespace)
			a.Equal(tc.expected, actual)
		})
	}
}

func TestListServiceIdentitiesForService(t *testing.T) {
	testCases := []struct {
		name      string
		namespace *corev1.Namespace
		pods      []*corev1.Pod
		service   *corev1.Service
		svc       service.MeshService
		expected  []identity.ServiceIdentity
		expectErr bool
	}{
		{
			name: "returns the service accounts for the given MeshService",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
				},
			},
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
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: "ns1",
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"k1": "v1", // matches labels on pod ns1/p1
					},
				},
			},
			svc:       service.MeshService{Name: "s1", Namespace: "ns1"}, // Matches service ns1/s1
			expected:  []identity.ServiceIdentity{identity.New("sa1", "ns1")},
			expectErr: false,
		},
		{
			name: "returns an error when the given MeshService is not found",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
				},
			},
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: "ns1",
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"k1": "v1", // matches labels on pod ns1/p1
					},
				},
			},
			svc:       service.MeshService{Name: "invalid", Namespace: "ns1"}, // Does not match service ns1/s1
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			controller := k8s.NewMockController(mockCtrl)
			controller.EXPECT().ListPods().Return(tc.pods).AnyTimes()
			if tc.svc.Name == tc.service.Name && tc.svc.Namespace == tc.service.Namespace {
				controller.EXPECT().GetService(tc.svc.Name, tc.svc.Namespace).Return(tc.service).AnyTimes()
			} else {
				controller.EXPECT().GetService(tc.svc.Name, tc.svc.Namespace).Return(nil).AnyTimes()
			}
			c := NewClient(controller)
			actual, err := c.ListServiceIdentitiesForService(tc.svc.Name, tc.svc.Namespace)
			a.Equal(tc.expectErr, err != nil)
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestK8sServicesToMeshServices(t *testing.T) {
	testCases := []struct {
		name         string
		svc          corev1.Service
		svcEndpoints []runtime.Object
		expected     []service.MeshService
	}{
		{
			name: "k8s service with single port and endpoint, no appProtocol set",
			// Single port on the service maps to a single MeshService.
			// Since no appProtocol is specified, MeshService.Protocol should default
			// to http.
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "s1",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "p1",
							Port: 80,
						},
					},
					ClusterIP: "10.0.0.1",
				},
			},
			svcEndpoints: []runtime.Object{
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						// Should match svc.Name and svc.Namespace
						Namespace: "ns1",
						Name:      "s1",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Ports: []corev1.EndpointPort{
								{
									// Must match the port of 'svc.Spec.Ports[0]'
									Port: 8080, // TargetPort
								},
							},
						},
					},
				},
			},
			expected: []service.MeshService{
				{
					Namespace:  "ns1",
					Name:       "s1",
					Port:       80,
					TargetPort: 8080,
					Protocol:   "http",
				},
			},
		},
		{
			name: "k8s service with single port and endpoint, no appProtocol set, protocol in port name",
			// Single port on the service maps to a single MeshService.
			// Since no appProtocol is specified, MeshService.Protocol should match
			// the protocol specified in the port name
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "s1",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "tcp-p1",
							Port: 80,
						},
					},
					ClusterIP: "10.0.0.1",
				},
			},
			svcEndpoints: []runtime.Object{
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						// Should match svc.Name and svc.Namespace
						Namespace: "ns1",
						Name:      "s1",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Ports: []corev1.EndpointPort{
								{
									// Must match the port of 'svc.Spec.Ports[0]'
									Port: 8080, // TargetPort
								},
							},
						},
					},
				},
			},
			expected: []service.MeshService{
				{
					Namespace:  "ns1",
					Name:       "s1",
					Port:       80,
					TargetPort: 8080,
					Protocol:   "tcp",
				},
			},
		},
		{
			name: "k8s headless service with single port and endpoint, no appProtocol set",
			// Single port on the service maps to a single MeshService.
			// Since no appProtocol is specified, MeshService.Protocol should default
			// to http.
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "s1",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "p1",
							Port: 80,
						},
					},
					ClusterIP: corev1.ClusterIPNone,
				},
			},
			svcEndpoints: []runtime.Object{
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						// Should match svc.Name and svc.Namespace
						Namespace: "ns1",
						Name:      "s1",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Addresses: []corev1.EndpointAddress{
								{
									IP:       "10.1.0.1",
									Hostname: "pod-0",
								},
							},
							Ports: []corev1.EndpointPort{
								{
									// Must match the port of 'svc.Spec.Ports[0]'
									Port: 8080, // TargetPort
								},
							},
						},
					},
				},
			},
			expected: []service.MeshService{
				{
					Namespace:  "ns1",
					Name:       "s1",
					Subdomain:  "pod-0",
					Port:       80,
					TargetPort: 8080,
					Protocol:   "http",
				},
			},
		},
		{
			name: "multiple ports on k8s service with appProtocol specified",
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "s1",
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "10.0.0.1",
					Ports: []corev1.ServicePort{
						{
							Name:        "p1",
							Port:        80,
							AppProtocol: pointer.StringPtr("http"),
						},
						{
							Name:        "p2",
							Port:        90,
							AppProtocol: pointer.StringPtr("tcp"),
						},
					},
				},
			},
			svcEndpoints: []runtime.Object{
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						// Should match svc.Name and svc.Namespace
						Namespace: "ns1",
						Name:      "s1",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Ports: []corev1.EndpointPort{
								{
									// Must match the port of 'svc.Spec.Ports[0]'
									Name:        "p1",
									Port:        8080, // TargetPort
									AppProtocol: pointer.StringPtr("http"),
								},
								{
									// Must match the port of 'svc.Spec.Ports[1]'
									Name:        "p2",
									Port:        9090, // TargetPort
									AppProtocol: pointer.StringPtr("tcp"),
								},
							},
						},
					},
				},
			},
			expected: []service.MeshService{
				{
					Namespace:  "ns1",
					Name:       "s1",
					Port:       80,
					TargetPort: 8080,
					Protocol:   "http",
				},
				{
					Namespace:  "ns1",
					Name:       "s1",
					Port:       90,
					TargetPort: 9090,
					Protocol:   "tcp",
				},
			},
		},
		{
			name: "multiple ports on k8s headless service with appProtocol specified",
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns1",
					Name:      "s1",
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: corev1.ClusterIPNone,
					Ports: []corev1.ServicePort{
						{
							Name:        "p1",
							Port:        80,
							AppProtocol: pointer.StringPtr("http"),
						},
						{
							Name:        "p2",
							Port:        90,
							AppProtocol: pointer.StringPtr("tcp"),
						},
					},
				},
			},
			svcEndpoints: []runtime.Object{
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						// Should match svc.Name and svc.Namespace
						Namespace: "ns1",
						Name:      "s1",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Addresses: []corev1.EndpointAddress{
								{
									IP:       "10.1.0.1",
									Hostname: "pod-0",
								},
							},
							Ports: []corev1.EndpointPort{
								{
									// Must match the port of 'svc.Spec.Ports[0]'
									Name:        "p1",
									Port:        8080, // TargetPort
									AppProtocol: pointer.StringPtr("http"),
								},
								{
									// Must match the port of 'svc.Spec.Ports[1]'
									Name:        "p2",
									Port:        9090, // TargetPort
									AppProtocol: pointer.StringPtr("tcp"),
								},
							},
						},
					},
				},
			},
			expected: []service.MeshService{
				{
					Namespace:  "ns1",
					Name:       "s1",
					Subdomain:  "pod-0",
					Port:       80,
					TargetPort: 8080,
					Protocol:   "http",
				},
				{
					Namespace:  "ns1",
					Name:       "s1",
					Subdomain:  "pod-0",
					Port:       90,
					TargetPort: 9090,
					Protocol:   "tcp",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			fakeClient := testclient.NewSimpleClientset(tc.svcEndpoints...)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(fakeClient))
			assert.Nil(err)

			kubecontroller := k8s.NewClient("", "", ic, nil, nil, messaging.NewBroker(make(<-chan struct{})))

			c := NewClient(kubecontroller)

			actual := c.serviceToMeshServices(tc.svc)
			assert.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestGetMeshService(t *testing.T) {
	osmNamespace := "osm"
	testCases := []struct {
		name               string
		svc                *corev1.Service
		endpoints          *corev1.Endpoints
		namespacedSvc      types.NamespacedName
		port               uint16
		expectedTargetPort uint16
		expectErr          bool
	}{
		{
			name: "TargetPort found",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: "ns1",
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "10.10.10.10",
					Ports: []corev1.ServicePort{{
						Name: "p1",
						Port: 80,
					}},
				},
			},
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: "ns1",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Ports: []corev1.EndpointPort{
							{
								Name: "p1",
								Port: 8080,
							},
						},
					},
				},
			},
			namespacedSvc:      types.NamespacedName{Namespace: "ns1", Name: "s1"}, // matches svc
			port:               80,                                                 // matches svc
			expectedTargetPort: 8080,                                               // matches endpoint's 'p1' port
			expectErr:          false,
		},
		{
			name: "TargetPort not found as given service name does not exist",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: "ns1",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{
						Name: "p1",
						Port: 80,
					}},
				},
			},
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: "ns1",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Ports: []corev1.EndpointPort{
							{
								Name: "p1",
								Port: 8080,
							},
						},
					},
				},
			},
			namespacedSvc:      types.NamespacedName{Namespace: "ns1", Name: "invalid"}, // does not match svc
			port:               80,                                                      // matches svc
			expectedTargetPort: 0,                                                       // matches endpoint's 'p1' port
			expectErr:          true,
		},
		{
			name: "TargetPort not found as Endpoint does not exist",
			svc: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: "ns1",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{
						Name: "p1",
						Port: 80,
					}},
				},
			},
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: "ns1",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Ports: []corev1.EndpointPort{
							{
								Name: "invalid", // does not match svc port
								Port: 8080,
							},
						},
					},
				},
			},
			namespacedSvc:      types.NamespacedName{Namespace: "ns1", Name: "s1"}, // matches svc
			port:               80,                                                 // matches svc
			expectedTargetPort: 0,                                                  // matches endpoint's 'p1' port
			expectErr:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			_ = ic.Add(informers.InformerKeyService, tc.svc, t)
			_ = ic.Add(informers.InformerKeyEndpoints, tc.endpoints, t)

			controller := k8s.NewClient(osmNamespace, "", ic, nil, nil, messaging.NewBroker(make(chan struct{})))

			c := NewClient(controller)

			actual, err := c.GetMeshService(tc.namespacedSvc.Name, tc.namespacedSvc.Namespace, tc.port)
			a.Equal(tc.expectedTargetPort, actual.TargetPort)
			a.Equal(tc.expectErr, err != nil)
		})
	}
}
