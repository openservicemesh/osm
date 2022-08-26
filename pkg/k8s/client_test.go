package k8s

import (
	"fmt"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/envoy"
	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	fakePolicyClient "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/tests"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	testMeshName = "mesh"
)

func TestIsMonitoredNamespace(t *testing.T) {
	testCases := []struct {
		name      string
		namespace *corev1.Namespace
		ns        string
		expected  bool
	}{
		{
			name: "namespace is monitored if is found in the namespace cache",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			ns:       "foo",
			expected: true,
		},
		{
			name: "namespace is not monitored if is not in the namespace cache",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			ns:       "invalid",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)

			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			_ = ic.Add(informers.InformerKeyNamespace, tc.namespace, t)

			actual := c.IsMonitoredNamespace(tc.ns)
			a.Equal(tc.expected, actual)
		})
	}
}

func TestGetNamespace(t *testing.T) {
	testCases := []struct {
		name      string
		namespace *corev1.Namespace
		ns        string
		expected  bool
	}{
		{
			name: "gets the namespace from the cache given its key",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			ns:       "foo",
			expected: true,
		},
		{
			name: "returns nil if the namespace is not found in the cache",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			ns:       "invalid",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			_ = ic.Add(informers.InformerKeyNamespace, tc.namespace, t)

			actual := c.GetNamespace(tc.ns)
			if tc.expected {
				a.Equal(tc.namespace, actual)
			} else {
				a.Nil(actual)
			}
		})
	}
}

func TestListMonitoredNamespaces(t *testing.T) {
	testCases := []struct {
		name       string
		namespaces []*corev1.Namespace
		expected   []string
	}{
		{
			name: "gets the namespace from the cache given its key",
			namespaces: []*corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns2",
					},
				},
			},
			expected: []string{"ns1", "ns2"},
		},
		{
			name:       "gets the namespace from the cache given its key",
			namespaces: nil,
			expected:   []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			for _, ns := range tc.namespaces {
				_ = ic.Add(informers.InformerKeyNamespace, ns, t)
			}

			actual, err := c.ListMonitoredNamespaces()
			a.Nil(err)
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestGetService(t *testing.T) {
	testCases := []struct {
		name     string
		service  *corev1.Service
		svc      service.MeshService
		expected bool
	}{
		{
			name: "gets the service from the cache given its key",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
				},
			},
			svc:      service.MeshService{Name: "foo", Namespace: "ns1"},
			expected: true,
		},
		{
			name: "returns nil if the service is not found in the cache",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
				},
			},
			svc:      service.MeshService{Name: "invalid", Namespace: "ns1"},
			expected: false,
		},
		{
			name: "gets the headless service from the cache from a subdomained MeshService",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-headless",
					Namespace: "ns1",
				},
			},
			svc:      service.MeshService{Name: "foo-headless", Namespace: "ns1", Subdomain: "foo-0"},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			_ = ic.Add(informers.InformerKeyService, tc.service, t)

			actual := c.GetService(tc.svc)
			if tc.expected {
				a.Equal(tc.service, actual)
			} else {
				a.Nil(actual)
			}
		})
	}
}

func TestListServices(t *testing.T) {
	testCases := []struct {
		name      string
		namespace *corev1.Namespace
		services  []*corev1.Service
		expected  []*corev1.Service
	}{
		{
			name: "gets the k8s services if their namespaces are monitored",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
				},
			},
			services: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns2",
						Name:      "s2",
					},
				},
			},
			expected: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			_ = ic.Add(informers.InformerKeyNamespace, tc.namespace, t)

			for _, s := range tc.services {
				_ = ic.Add(informers.InformerKeyService, s, t)
			}

			actual := c.ListServices()
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestListServiceAccounts(t *testing.T) {
	testCases := []struct {
		name      string
		namespace *corev1.Namespace
		sa        []*corev1.ServiceAccount
		expected  []*corev1.ServiceAccount
	}{
		{
			name: "gets the k8s service accounts if their namespaces are monitored",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
				},
			},
			sa: []*corev1.ServiceAccount{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns2",
						Name:      "s2",
					},
				},
			},
			expected: []*corev1.ServiceAccount{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			_ = ic.Add(informers.InformerKeyNamespace, tc.namespace, t)

			for _, s := range tc.sa {
				_ = ic.Add(informers.InformerKeyServiceAccount, s, t)
			}

			actual := c.ListServiceAccounts()
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestListPods(t *testing.T) {
	testCases := []struct {
		name      string
		namespace *corev1.Namespace
		pods      []*corev1.Pod
		expected  []*corev1.Pod
	}{
		{
			name: "gets the k8s pods if their namespaces are monitored",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
				},
			},
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns2",
						Name:      "s2",
					},
				},
			},
			expected: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			_ = ic.Add(informers.InformerKeyNamespace, tc.namespace, t)

			for _, p := range tc.pods {
				_ = ic.Add(informers.InformerKeyPod, p, t)
			}

			actual := c.ListPods()
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestGetEndpoints(t *testing.T) {
	testCases := []struct {
		name      string
		endpoints *corev1.Endpoints
		svc       service.MeshService
		expected  *corev1.Endpoints
	}{
		{
			name: "gets the service from the cache given its key",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
				},
			},
			svc: service.MeshService{Name: "foo", Namespace: "ns1"},
			expected: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
				},
			},
		},
		{
			name: "returns nil if the service is not found in the cache",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
				},
			},
			svc:      service.MeshService{Name: "invalid", Namespace: "ns1"},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			_ = ic.Add(informers.InformerKeyEndpoints, tc.endpoints, t)

			actual, err := c.GetEndpoints(tc.svc)
			a.Nil(err)
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
		expected  []identity.K8sServiceAccount
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
			svc: service.MeshService{Name: "s1", Namespace: "ns1"}, // Matches service ns1/s1
			expected: []identity.K8sServiceAccount{
				{Namespace: "ns1", Name: "sa1"},
			},
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

			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			_ = ic.Add(informers.InformerKeyNamespace, tc.namespace, t)
			for _, p := range tc.pods {
				_ = ic.Add(informers.InformerKeyPod, p, t)
			}
			_ = ic.Add(informers.InformerKeyService, tc.service, t)

			actual, err := c.ListServiceIdentitiesForService(tc.svc)
			a.Equal(tc.expectErr, err != nil)
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	testCases := []struct {
		name             string
		existingResource interface{}
		updatedResource  interface{}
		expectErr        bool
	}{
		{
			name: "valid IngressBackend resource",
			existingResource: &policyv1alpha1.IngressBackend{
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
			updatedResource: &policyv1alpha1.IngressBackend{
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
				Status: policyv1alpha1.IngressBackendStatus{
					CurrentStatus: "valid",
					Reason:        "valid",
				},
			},
		},
		{
			name: "valid UpstreamTrafficSetting resource",
			existingResource: &policyv1alpha1.UpstreamTrafficSetting{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
					Host: "foo.bar.svc.cluster.local",
				},
			},
			updatedResource: &policyv1alpha1.UpstreamTrafficSetting{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
					Host: "foo.bar.svc.cluster.local",
				},
				Status: policyv1alpha1.UpstreamTrafficSettingStatus{
					CurrentStatus: "valid",
					Reason:        "valid",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			kubeClient := testclient.NewSimpleClientset()
			policyClient := fakePolicyClient.NewSimpleClientset(tc.existingResource.(runtime.Object))
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(kubeClient), informers.WithPolicyClient(policyClient))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, policyClient, nil)
			switch v := tc.updatedResource.(type) {
			case *policyv1alpha1.IngressBackend:
				_, err = c.UpdateIngressBackendStatus(v)
				a.Equal(tc.expectErr, err != nil)
			case *policyv1alpha1.UpstreamTrafficSetting:
				_, err = c.UpdateUpstreamTrafficSettingStatus(v)
				a.Equal(tc.expectErr, err != nil)
			}
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

			kubeController := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil)
			assert.NotNil(kubeController)

			actual := kubeController.ServiceToMeshServices(tc.svc)
			assert.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestGetPodForProxy(t *testing.T) {
	assert := tassert.New(t)
	stop := make(chan struct{})
	defer close(stop)

	proxyUUID := uuid.New()
	someOtherEnvoyUID := uuid.New()
	namespace := tests.BookstoreServiceAccount.Namespace

	podlabels := map[string]string{
		constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
	}
	someOthePodLabels := map[string]string{
		constants.AppLabel:               tests.SelectorValue,
		constants.EnvoyUniqueIDLabelName: someOtherEnvoyUID.String(),
	}

	pod := tests.NewPodFixture(namespace, "pod-1", tests.BookstoreServiceAccountName, podlabels)
	kubeClient := fake.NewSimpleClientset(
		monitoredNS(namespace),
		monitoredNS("bad-namespace"),
		tests.NewPodFixture(namespace, "pod-0", tests.BookstoreServiceAccountName, someOthePodLabels),
		pod,
		tests.NewPodFixture(namespace, "pod-2", tests.BookstoreServiceAccountName, someOthePodLabels),
	)

	ic, err := informers.NewInformerCollection(testMeshName, stop, informers.WithKubeClient(kubeClient))
	assert.Nil(err)

	kubeController := NewClient("osm", tests.OsmMeshConfigName, ic, nil, messaging.NewBroker(nil))

	testCases := []struct {
		name  string
		pod   *corev1.Pod
		proxy *envoy.Proxy
		err   error
	}{
		{
			name:  "fails when UUID does not match",
			proxy: envoy.NewProxy(envoy.KindSidecar, uuid.New(), tests.BookstoreServiceIdentity, nil, 1),
			err:   errDidNotFindPodForUUID,
		},
		{
			name:  "fails when service account does not match certificate",
			proxy: &envoy.Proxy{UUID: proxyUUID, Identity: identity.New("bad-name", namespace)},
			err:   errServiceAccountDoesNotMatchProxy,
		},
		{
			name:  "2 pods with same uuid",
			proxy: envoy.NewProxy(envoy.KindSidecar, someOtherEnvoyUID, tests.BookstoreServiceIdentity, nil, 1),
			err:   errMoreThanOnePodForUUID,
		},
		{
			name:  "fails when namespace does not match certificate",
			proxy: envoy.NewProxy(envoy.KindSidecar, proxyUUID, identity.New(tests.BookstoreServiceAccountName, "bad-namespace"), nil, 1),
			err:   errNamespaceDoesNotMatchProxy,
		},
		{
			name:  "works as expected",
			pod:   pod,
			proxy: envoy.NewProxy(envoy.KindSidecar, proxyUUID, tests.BookstoreServiceIdentity, nil, 1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			pod, err := kubeController.GetPodForProxy(tc.proxy)

			assert.Equal(tc.pod, pod)
			assert.Equal(tc.err, err)
		})
	}
}

func monitoredNS(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: testMeshName,
			},
		},
	}
}

func TestGetTargetPortForServicePort(t *testing.T) {
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
		{
			name: "TargetPort not found as Endpoint matching given service does not exist",
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
					Name:      "invalid", // does not match svc
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
			expectedTargetPort: 0,                                                  // matches endpoint's 'p1' port
			expectErr:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)

			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient(osmNamespace, tests.OsmMeshConfigName, ic, nil, nil)
			_ = ic.Add(informers.InformerKeyService, tc.svc, t)
			_ = ic.Add(informers.InformerKeyEndpoints, tc.endpoints, t)

			actual, err := c.GetTargetPortForServicePort(tc.namespacedSvc, tc.port)
			a.Equal(tc.expectedTargetPort, actual)
			a.Equal(tc.expectErr, err != nil)
		})
	}
}

func TestGetMeshConfig(t *testing.T) {
	a := assert.New(t)

	meshConfigClient := fakeConfig.NewSimpleClientset()
	stop := make(chan struct{})
	osmNamespace := "osm"
	osmMeshConfigName := "osm-mesh-config"

	ic, err := informers.NewInformerCollection("osm", stop, informers.WithConfigClient(meshConfigClient, osmMeshConfigName, osmNamespace))
	a.Nil(err)

	c := NewClient(osmNamespace, tests.OsmMeshConfigName, ic, nil, nil)

	// Returns empty MeshConfig if informer cache is empty
	a.Equal(configv1alpha2.MeshConfig{}, c.GetMeshConfig())

	newObj := &configv1alpha2.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.openservicemesh.io",
			Kind:       "MeshConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      osmMeshConfigName,
		},
	}
	err = c.informers.Add(informers.InformerKeyMeshConfig, newObj, t)
	a.Nil(err)
	a.Equal(*newObj, c.GetMeshConfig())
}

func TestMetricsHandler(t *testing.T) {
	a := assert.New(t)
	osmMeshConfigName := "osm-mesh-config"

	c := &Client{
		informers: &informers.InformerCollection{},
	}
	handlers := c.metricsHandler()
	metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.FeatureFlagEnabled)

	// Adding the MeshConfig
	handlers.OnAdd(&configv1alpha2.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: osmMeshConfigName,
		},
		Spec: configv1alpha2.MeshConfigSpec{
			FeatureFlags: configv1alpha2.FeatureFlags{
				EnableRetryPolicy: true,
			},
		},
	})
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 1` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 0` + "\n"))

	// Updating the MeshConfig
	handlers.OnUpdate(nil, &configv1alpha2.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: osmMeshConfigName,
		},
		Spec: configv1alpha2.MeshConfigSpec{
			FeatureFlags: configv1alpha2.FeatureFlags{
				EnableSnapshotCacheMode: true,
			},
		},
	})
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 0` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 1` + "\n"))

	// Deleting the MeshConfig
	handlers.OnDelete(&configv1alpha2.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: osmMeshConfigName,
		},
		Spec: configv1alpha2.MeshConfigSpec{
			FeatureFlags: configv1alpha2.FeatureFlags{
				EnableSnapshotCacheMode: true,
			},
		},
	})
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 0` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 0` + "\n"))
}

func TestListEgressPoliciesForSourceIdentity(t *testing.T) {
	egressNs := "test"
	egressNsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: egressNs,
		},
	}

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
						Namespace: egressNs,
					},
					Spec: policyv1alpha1.EgressSpec{
						Sources: []policyv1alpha1.EgressSourceSpec{
							{
								Kind:      "ServiceAccount",
								Name:      "sa-1",
								Namespace: egressNs,
							},
							{
								Kind:      "ServiceAccount",
								Name:      "sa-2",
								Namespace: egressNs,
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
						Namespace: egressNs,
					},
					Spec: policyv1alpha1.EgressSpec{
						Sources: []policyv1alpha1.EgressSourceSpec{
							{
								Kind:      "ServiceAccount",
								Name:      "sa-1",
								Namespace: egressNs,
							},
							{
								Kind:      "ServiceAccount",
								Name:      "sa-2",
								Namespace: egressNs,
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
			source: identity.K8sServiceAccount{Name: "sa-1", Namespace: egressNs},
			expectedEgresses: []*policyv1alpha1.Egress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "egress-1",
						Namespace: egressNs,
					},
					Spec: policyv1alpha1.EgressSpec{
						Sources: []policyv1alpha1.EgressSourceSpec{
							{
								Kind:      "ServiceAccount",
								Name:      "sa-1",
								Namespace: egressNs,
							},
							{
								Kind:      "ServiceAccount",
								Name:      "sa-2",
								Namespace: egressNs,
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
			informerCollection, err := informers.NewInformerCollection("osm", nil,
				informers.WithPolicyClient(fakeClient),
				informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)

			c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, fakeClient, nil)
			a.Nil(err)
			a.NotNil(c)

			// monitor namespaces
			err = c.informers.Add(informers.InformerKeyNamespace, egressNsObj, t)
			a.Nil(err)

			// Create fake egress policies
			for _, egressPolicy := range tc.allEgresses {
				_ = c.informers.Add(informers.InformerKeyEgress, egressPolicy, t)
			}

			actual := c.ListEgressPoliciesForSourceIdentity(tc.source)
			a.ElementsMatch(tc.expectedEgresses, actual)
		})
	}
}

func TestGetIngressBackendPolicy(t *testing.T) {
	testCases := []struct {
		name                   string
		allResources           []*policyv1alpha1.IngressBackend
		backend                service.MeshService
		expectedIngressBackend *policyv1alpha1.IngressBackend
	}{
		{
			name:                   "IngressBackend policy not found",
			allResources:           nil,
			backend:                service.MeshService{Name: "backend1", Namespace: "test"},
			expectedIngressBackend: nil,
		},
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			informerCollection, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, fakeClient, nil)
			a.Nil(err)
			a.NotNil(c)

			// Create fake egress policies
			for _, ingressBackend := range tc.allResources {
				_ = c.informers.Add(informers.InformerKeyIngressBackend, ingressBackend, t)
			}

			actual := c.GetIngressBackendPolicy(tc.backend)
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

	mockKubeController := NewMockController(mockCtrl)
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
			informerCollection, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, fakeClient, nil)
			a.Nil(err)
			a.NotNil(c)

			// Create fake retry policies
			for _, retryPolicy := range tc.allRetries {
				err := c.informers.Add(informers.InformerKeyRetry, retryPolicy, t)
				a.Nil(err)
			}

			actual := c.ListRetryPolicies(tc.source)
			a.ElementsMatch(tc.expectedRetries, actual)
		})
	}
}

func TestGetUpstreamTrafficSettingByService(t *testing.T) {
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
		{
			name: "no filter option specified",
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
			service:  nil,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			informerCollection, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, fakeClient, nil)
			a.Nil(err)
			a.NotNil(c)

			// Create fake egress policies
			for _, resource := range tc.allResources {
				_ = c.informers.Add(informers.InformerKeyUpstreamTrafficSetting, resource, t)
			}

			actual := c.GetUpstreamTrafficSettingByService(tc.service)
			a.Equal(tc.expected, actual)
		})
	}
}

func TestGetUpstreamTrafficSettingByNamespace(t *testing.T) {
	testCases := []struct {
		name         string
		allResources []*policyv1alpha1.UpstreamTrafficSetting
		namespace    *types.NamespacedName
		expected     *policyv1alpha1.UpstreamTrafficSetting
	}{
		{
			name: "UpstreamTrafficSetting namespaced name found",
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
			namespace: &types.NamespacedName{Namespace: "ns1", Name: "u1"},
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
			name: "UpstreamTrafficSetting namespaced name not found",
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
			namespace: &types.NamespacedName{Namespace: "ns1", Name: "u3"},
			expected:  nil,
		},
		{
			name: "no filter option specified",
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
			namespace: nil,
			expected:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			informerCollection, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, fakeClient, nil)
			a.Nil(err)
			a.NotNil(c)

			// Create fake egress policies
			for _, resource := range tc.allResources {
				_ = c.informers.Add(informers.InformerKeyUpstreamTrafficSetting, resource, t)
			}

			actual := c.GetUpstreamTrafficSettingByNamespace(tc.namespace)
			a.Equal(tc.expected, actual)
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
