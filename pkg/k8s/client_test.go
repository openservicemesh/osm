package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiAccessClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	smiSpecClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned/fake"
	smiSplitClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned/fake"
	mcs "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
	mcsClientFake "sigs.k8s.io/mcs-api/pkg/client/clientset/versioned/fake"

	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	fakeConfigClient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	fakePolicyClient "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/tests"
)

var (
	testMeshName = "mesh"
	testNs       = "test-ns"
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
					Labels: map[string]string{
						constants.OSMKubeResourceMonitorAnnotation: testMeshName,
					},
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

			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(fake.NewSimpleClientset(tc.namespace), testMeshName))
			a.NoError(err)

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
					Labels: map[string]string{
						constants.OSMKubeResourceMonitorAnnotation: testMeshName,
					},
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
					Labels: map[string]string{
						constants.OSMKubeResourceMonitorAnnotation: testMeshName,
					},
				},
			},
			ns:       "invalid",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(fake.NewSimpleClientset(tc.namespace), testMeshName))
			a.NoError(err)

			actual := c.GetNamespace(tc.ns)
			if tc.expected {
				a.Equal(tc.namespace, actual)
			} else {
				a.Nil(actual)
			}
		})
	}
}

func TestListNamespaces(t *testing.T) {
	testCases := []struct {
		name       string
		namespaces []runtime.Object
		expected   []string
	}{
		{
			name: "gets the namespace from the cache given its key",
			namespaces: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns1",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: testMeshName,
						},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns2",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: testMeshName,
						},
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
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(fake.NewSimpleClientset(tc.namespaces...), testMeshName))
			a.NoError(err)

			actual, err := c.ListNamespaces()
			a.Nil(err)
			names := make([]string, 0, len(actual))
			for _, ns := range actual {
				names = append(names, ns.Name)
			}
			a.ElementsMatch(tc.expected, names)
		})
	}
}

func TestGetService(t *testing.T) {
	testCases := []struct {
		name         string
		service      *corev1.Service
		svcName      string
		svcNamespace string
		expected     bool
	}{
		{
			name: "gets the service from the cache given its key",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
				},
			},
			svcName:      "foo",
			svcNamespace: "ns1",
			expected:     true,
		},
		{
			name: "returns nil if the service is not found in the cache",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
				},
			},
			svcName:      "invalid",
			svcNamespace: "ns1",
			expected:     false,
		},
		{
			name: "gets the headless service from the cache from a subdomained MeshService",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-headless",
					Namespace: "ns1",
				},
			},
			svcName:      "foo-headless",
			svcNamespace: "ns1",
			expected:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(fake.NewSimpleClientset(tc.service), testMeshName))
			a.NoError(err)

			actual := c.GetService(tc.svcName, tc.svcNamespace)
			if tc.expected {
				a.Equal(tc.service, actual)
			} else {
				a.Nil(actual)
			}
		})
	}
}

func TestListSecrets(t *testing.T) {
	testCases := []struct {
		name     string
		secrets  []runtime.Object
		expected []*models.Secret
	}{
		{
			name: "get multiple k8s secrets",
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
						Labels:    map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns2",
						Name:      "s2",
						Labels:    map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue},
					},
				},
			},
			expected: []*models.Secret{
				{
					Namespace: "ns1",
					Name:      "s1",
				},
				{
					Namespace: "ns2",
					Name:      "s2",
				},
			},
		},
		{
			name: "get one k8s secret",
			secrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
						Labels:    map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue},
					},
				},
			},
			expected: []*models.Secret{
				{
					Namespace: "ns1",
					Name:      "s1",
				},
			},
		},
		{
			name: "no k8s secret",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(fake.NewSimpleClientset(tc.secrets...), testMeshName))
			a.Nil(err)

			actual := c.ListSecrets()
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestGetSecret(t *testing.T) {
	testCases := []struct {
		name       string
		secret     *corev1.Secret
		secretName string
		namespace  string
		expSecret  *models.Secret
	}{
		{
			name: "gets the secret from the cache",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
					Labels:    map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue},
				},
			},
			secretName: "foo",
			namespace:  "ns1",
			expSecret: &models.Secret{
				Name:      "foo",
				Namespace: "ns1",
			},
		},
		{
			name: "returns nil if the secret is not found in the cache",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
					Labels:    map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue},
				},
			},
			secretName: "doesntExist",
			namespace:  "ns1",
			expSecret:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker,
				WithKubeClient(fake.NewSimpleClientset(tc.secret), testMeshName))
			a.NoError(err)

			actual := c.GetSecret(tc.secretName, tc.namespace)
			a.Equal(tc.expSecret, actual)
		})
	}
}

func TestUpdateSecret(t *testing.T) {
	testCases := []struct {
		name         string
		corev1Secret *corev1.Secret
		secret       *models.Secret
		secretData   map[string][]byte
	}{
		{
			name: "Update secret",
			corev1Secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s1",
					Namespace: "ns1",
					Labels:    map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue},
				},
				Data: map[string][]byte{},
			},
			secret: &models.Secret{
				Name:      "s1",
				Namespace: "ns1",
				Data:      map[string][]byte{"a": {}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient(testNs, tests.OsmMeshConfigName, broker,
				WithKubeClient(fake.NewSimpleClientset(tc.corev1Secret), testMeshName))
			a.NoError(err)

			err = c.UpdateSecret(context.Background(), tc.secret)
			a.Nil(err)

			secret, err := c.kubeClient.CoreV1().Secrets("ns1").Get(context.Background(), tc.secret.Name, metav1.GetOptions{})
			a.Nil(err)
			a.Equal(tc.secret.Data, secret.Data)
		})
	}
}

func TestListServices(t *testing.T) {
	testCases := []struct {
		name      string
		namespace *corev1.Namespace
		services  []runtime.Object
		expected  []*corev1.Service
	}{
		{
			name: "gets the k8s services if their namespaces are monitored",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
					Labels: map[string]string{
						constants.OSMKubeResourceMonitorAnnotation: testMeshName,
					},
				},
			},
			services: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
				},
				&corev1.Service{
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
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(fake.NewSimpleClientset(append([]runtime.Object{tc.namespace}, tc.services...)...), testMeshName))
			a.NoError(err)

			actual := c.ListServices()
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestListServiceAccounts(t *testing.T) {
	testCases := []struct {
		name      string
		namespace *corev1.Namespace
		sa        []runtime.Object
		expected  []*corev1.ServiceAccount
	}{
		{
			name: "gets the k8s service accounts if their namespaces are monitored",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
					Labels: map[string]string{
						constants.OSMKubeResourceMonitorAnnotation: testMeshName,
					},
				},
			},
			sa: []runtime.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
				},
				&corev1.ServiceAccount{
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
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(fake.NewSimpleClientset(append([]runtime.Object{tc.namespace}, tc.sa...)...), testMeshName))
			a.NoError(err)

			actual := c.ListServiceAccounts()
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestListPods(t *testing.T) {
	testCases := []struct {
		name      string
		namespace *corev1.Namespace
		pods      []runtime.Object
		expected  []*corev1.Pod
	}{
		{
			name: "gets the k8s pods if their namespaces are monitored",
			namespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
					Labels: map[string]string{
						constants.OSMKubeResourceMonitorAnnotation: testMeshName,
					},
				},
			},
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
					},
				},
				&corev1.Pod{
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
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(fake.NewSimpleClientset(append([]runtime.Object{tc.namespace}, tc.pods...)...), testMeshName))
			a.NoError(err)

			actual := c.ListPods()
			a.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestGetEndpoints(t *testing.T) {
	testCases := []struct {
		name         string
		endpoints    *corev1.Endpoints
		svcName      string
		svcNamespace string
		expected     *corev1.Endpoints
	}{
		{
			name: "gets the service from the cache given its key",
			endpoints: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns1",
				},
			},
			svcName:      "foo",
			svcNamespace: "ns1",
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
			svcName:      "invalid",
			svcNamespace: "ns1",
			expected:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(fake.NewSimpleClientset(tc.endpoints), testMeshName))
			a.NoError(err)

			actual, err := c.GetEndpoints(tc.svcName, tc.svcNamespace)
			a.Nil(err)
			a.Equal(tc.expected, actual)
		})
	}
}

func TestPolicyUpdateStatus(t *testing.T) {
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
			kubeClient := fake.NewSimpleClientset()
			policyClient := fakePolicyClient.NewSimpleClientset(tc.existingResource.(runtime.Object))

			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithKubeClient(kubeClient, testMeshName), WithPolicyClient(policyClient))
			a.NoError(err)
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

func TestConfigUpdateStatus(t *testing.T) {
	testCases := []struct {
		name             string
		existingResource interface{}
		updatedResource  interface{}
	}{
		{
			name: "valid MeshRootCertificate resource",
			existingResource: &configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: configv1alpha2.MeshRootCertificateSpec{
					Role: configv1alpha2.ActiveRole,
					Provider: configv1alpha2.ProviderSpec{
						Tresor: &configv1alpha2.TresorProviderSpec{
							CA: configv1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
				},
			},
			updatedResource: &configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: configv1alpha2.MeshRootCertificateSpec{
					Role: configv1alpha2.ActiveRole,
					Provider: configv1alpha2.ProviderSpec{
						Tresor: &configv1alpha2.TresorProviderSpec{
							CA: configv1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: "Error",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			kubeClient := fake.NewSimpleClientset()
			configClient := fakeConfigClient.NewSimpleClientset(tc.existingResource.(runtime.Object))
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, broker, WithKubeClient(kubeClient, testMeshName), WithConfigClient(configClient))
			a.NoError(err)
			switch v := tc.updatedResource.(type) {
			case *configv1alpha2.MeshRootCertificate:
				_, err = c.UpdateMeshRootCertificateStatus(v)
				a.NoError(err)
			}
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

func TestGetMeshConfig(t *testing.T) {
	a := assert.New(t)

	osmNamespace := "osm"
	osmMeshConfigName := "osm-mesh-config"

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

	stop := make(chan struct{})
	broker := messaging.NewBroker(stop)

	meshConfigClient := fakeConfigClient.NewSimpleClientset()
	c, err := NewClient(osmNamespace, tests.OsmMeshConfigName, broker, WithConfigClient(meshConfigClient))
	a.NoError(err)

	// Returns empty MeshConfig if informer cache is empty
	a.Equal(configv1alpha2.MeshConfig{}, c.GetMeshConfig())

	meshConfigClient = fakeConfigClient.NewSimpleClientset(newObj)
	c, err = NewClient(osmNamespace, tests.OsmMeshConfigName, broker, WithConfigClient(meshConfigClient))
	a.NoError(err)

	a.Nil(err)
	a.Equal(*newObj, c.GetMeshConfig())
}

func TestMetricsHandler(t *testing.T) {
	a := assert.New(t)
	osmMeshConfigName := "osm-mesh-config"

	meshConfigClient := fakeConfigClient.NewSimpleClientset()
	stop := make(chan struct{})
	broker := messaging.NewBroker(stop)
	c, err := NewClient(tests.OsmNamespace, osmMeshConfigName, broker, WithConfigClient(meshConfigClient))
	a.NoError(err)
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

func TestListEgressPolicies(t *testing.T) {
	egressNsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: testMeshName,
			},
		},
	}

	outMeshResource := &policyv1alpha1.Egress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "egress-1",
			Namespace: "wrong-ns",
		},
		Spec: policyv1alpha1.EgressSpec{
			Sources: []policyv1alpha1.EgressSourceSpec{
				{
					Kind:      "ServiceAccount",
					Name:      "sa-1",
					Namespace: testNs,
				},
				{
					Kind:      "ServiceAccount",
					Name:      "sa-2",
					Namespace: testNs,
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
	}
	inMeshResource := &policyv1alpha1.Egress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "egress-1",
			Namespace: testNs,
		},
		Spec: policyv1alpha1.EgressSpec{
			Sources: []policyv1alpha1.EgressSourceSpec{
				{
					Kind:      "ServiceAccount",
					Name:      "sa-1",
					Namespace: testNs,
				},
				{
					Kind:      "ServiceAccount",
					Name:      "sa-2",
					Namespace: testNs,
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
	}

	testCases := []struct {
		name             string
		allEgresses      []runtime.Object
		expectedEgresses []*policyv1alpha1.Egress
	}{
		{
			name:             "Only return egress resources for monitored namespaces",
			allEgresses:      []runtime.Object{inMeshResource, outMeshResource},
			expectedEgresses: []*policyv1alpha1.Egress{inMeshResource},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset(tc.allEgresses...)

			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, broker, WithPolicyClient(fakeClient), WithKubeClient(fake.NewSimpleClientset(egressNsObj), testMeshName))
			a.NoError(err)
			// monitor namespaces

			policies := c.ListEgressPolicies()
			a.ElementsMatch(tc.expectedEgresses, policies)
		})
	}
}

func TestListRetryPolicy(t *testing.T) {
	policyNsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: testMeshName,
			},
		},
	}

	var thresholdUintVal uint32 = 3
	thresholdTimeoutDuration := metav1.Duration{Duration: time.Duration(5 * time.Second)}
	thresholdBackoffDuration := metav1.Duration{Duration: time.Duration(1 * time.Second)}

	outMeshResource := &policyv1alpha1.Retry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "retry-1",
			Namespace: "wrong-ns",
		},
		Spec: policyv1alpha1.RetrySpec{
			Source: policyv1alpha1.RetrySrcDstSpec{
				Kind:      "ServiceAccount",
				Name:      "sa-1",
				Namespace: testNs,
			},
			Destinations: []policyv1alpha1.RetrySrcDstSpec{
				{
					Kind:      "Service",
					Name:      "s1",
					Namespace: testNs,
				},
				{
					Kind:      "Service",
					Name:      "s2",
					Namespace: testNs,
				},
			},
			RetryPolicy: policyv1alpha1.RetryPolicySpec{
				RetryOn:                  "",
				NumRetries:               &thresholdUintVal,
				PerTryTimeout:            &thresholdTimeoutDuration,
				RetryBackoffBaseInterval: &thresholdBackoffDuration,
			},
		},
	}
	inMeshResource := &policyv1alpha1.Retry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "retry-1",
			Namespace: testNs,
		},
		Spec: policyv1alpha1.RetrySpec{
			Source: policyv1alpha1.RetrySrcDstSpec{
				Kind:      "ServiceAccount",
				Name:      "sa-1",
				Namespace: testNs,
			},
			Destinations: []policyv1alpha1.RetrySrcDstSpec{
				{
					Kind:      "Service",
					Name:      "s1",
					Namespace: testNs,
				},
				{
					Kind:      "Service",
					Name:      "s2",
					Namespace: testNs,
				},
			},
			RetryPolicy: policyv1alpha1.RetryPolicySpec{
				RetryOn:                  "",
				NumRetries:               &thresholdUintVal,
				PerTryTimeout:            &thresholdTimeoutDuration,
				RetryBackoffBaseInterval: &thresholdBackoffDuration,
			},
		},
	}

	testCases := []struct {
		name            string
		allRetries      []runtime.Object
		expectedRetries []*policyv1alpha1.Retry
	}{
		{
			name:            "Only return retry resources for monitored namespaces",
			allRetries:      []runtime.Object{inMeshResource, outMeshResource},
			expectedRetries: []*policyv1alpha1.Retry{inMeshResource},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset(tc.allRetries...)

			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, broker, WithPolicyClient(fakeClient), WithKubeClient(fake.NewSimpleClientset(policyNsObj), testMeshName))
			a.NoError(err)

			policies := c.ListRetryPolicies()
			a.ElementsMatch(tc.expectedRetries, policies)
		})
	}
}

func TestListUpstreamTrafficSetting(t *testing.T) {
	settingNsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: testMeshName,
			},
		},
	}

	inMeshResource := &policyv1alpha1.UpstreamTrafficSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u1",
			Namespace: testNs,
		},
		Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
			Host: "s1.ns1.svc.cluster.local",
		},
	}
	outMeshResource := &policyv1alpha1.UpstreamTrafficSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u1",
			Namespace: "wrong-ns",
		},
		Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
			Host: "s1.ns1.svc.cluster.local",
		},
	}
	testCases := []struct {
		name         string
		allResources []runtime.Object
		expected     []*policyv1alpha1.UpstreamTrafficSetting
	}{
		{
			name:         "Only return upstream traffic settings for monitored namespaces",
			allResources: []runtime.Object{inMeshResource, outMeshResource},
			expected:     []*policyv1alpha1.UpstreamTrafficSetting{inMeshResource},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset(tc.allResources...)
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithPolicyClient(fakeClient), WithKubeClient(fake.NewSimpleClientset(settingNsObj), testMeshName))
			a.NoError(err)

			actual := c.ListUpstreamTrafficSettings()
			a.Equal(tc.expected, actual)
		})
	}
}

func TestGetMeshRootCertificate(t *testing.T) {
	testCases := []struct {
		name                string
		meshRootCertificate *configv1alpha2.MeshRootCertificate
		mrcName             string
		expected            bool
	}{
		{
			name: "gets the MRC from the cache given its key",
			meshRootCertificate: &configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-system",
				},
			},
			mrcName:  "mrc",
			expected: true,
		},
		{
			name: "returns nil if the MRC is not found in the cache",
			meshRootCertificate: &configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-system",
				},
			},
			mrcName:  "mrc2",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			stop := make(chan struct{})
			broker := messaging.NewBroker(stop)

			c, err := NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, broker, WithConfigClient(fakeConfigClient.NewSimpleClientset(tc.meshRootCertificate)))
			a.NoError(err)

			actual := c.GetMeshRootCertificate(tc.mrcName)
			if tc.expected {
				a.Equal(tc.meshRootCertificate, actual)
			} else {
				a.Nil(actual)
			}
		})
	}
}

func TestListMeshRootCertificates(t *testing.T) {
	a := assert.New(t)

	mrc := &configv1alpha2.MeshRootCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "osm-mesh-root-certificate",
			Namespace: "osm-system",
		},
		Spec: configv1alpha2.MeshRootCertificateSpec{
			Role: configv1alpha2.ActiveRole,
			Provider: configv1alpha2.ProviderSpec{
				Tresor: &configv1alpha2.TresorProviderSpec{
					CA: configv1alpha2.TresorCASpec{
						SecretRef: v1.SecretReference{
							Name:      "osm-ca-bundle",
							Namespace: "osm-system",
						},
					},
				},
			},
		},
	}
	mrcClient := fakeConfigClient.NewSimpleClientset(mrc)
	stop := make(chan struct{})
	broker := messaging.NewBroker(stop)

	c, err := NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, broker, WithConfigClient(mrcClient))
	a.NoError(err)

	mrcList, err := c.ListMeshRootCertificates()
	a.NoError(err)
	a.Contains(mrcList, mrc)
	a.Len(mrcList, 1)
}

func TestListHTTPTrafficSpecs(t *testing.T) {
	nsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: testMeshName,
			},
		},
	}

	obj := &smiSpecs.HTTPRouteGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha4",
			Kind:       "HTTPRouteGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNs,
			Name:      "test-ListHTTPTrafficSpecs",
		},
		Spec: smiSpecs.HTTPRouteGroupSpec{
			Matches: []smiSpecs.HTTPMatch{
				{
					Name:      tests.BuyBooksMatchName,
					PathRegex: tests.BookstoreBuyPath,
					Methods:   []string{"GET"},
					Headers: map[string]string{
						"user-agent": tests.HTTPUserAgent,
					},
				},
				{
					Name:      tests.SellBooksMatchName,
					PathRegex: tests.BookstoreSellPath,
					Methods:   []string{"GET"},
				},
				{
					Name: tests.WildcardWithHeadersMatchName,
					Headers: map[string]string{
						"user-agent": tests.HTTPUserAgent,
					},
				},
			},
		},
	}
	a := assert.New(t)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset(obj)
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	stop := make(chan struct{})
	broker := messaging.NewBroker(stop)

	c, err := NewClient("osm", tests.OsmMeshConfigName, broker,
		WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
		WithKubeClient(fake.NewSimpleClientset(nsObj), testMeshName),
	)
	a.NoError(err)
	// Verify
	actual := c.ListHTTPTrafficSpecs()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])
}

func TestGetHTTPRouteGroup(t *testing.T) {
	nsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: testMeshName,
			},
		},
	}
	obj := &smiSpecs.HTTPRouteGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha4",
			Kind:       "HTTPRouteGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNs,
			Name:      "foo",
		},
		Spec: smiSpecs.HTTPRouteGroupSpec{
			Matches: []smiSpecs.HTTPMatch{
				{
					Name:      tests.BuyBooksMatchName,
					PathRegex: tests.BookstoreBuyPath,
					Methods:   []string{"GET"},
					Headers: map[string]string{
						"user-agent": tests.HTTPUserAgent,
					},
				},
				{
					Name:      tests.SellBooksMatchName,
					PathRegex: tests.BookstoreSellPath,
					Methods:   []string{"GET"},
				},
				{
					Name: tests.WildcardWithHeadersMatchName,
					Headers: map[string]string{
						"user-agent": tests.HTTPUserAgent,
					},
				},
			},
		},
	}
	a := assert.New(t)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset(obj)
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	stop := make(chan struct{})
	broker := messaging.NewBroker(stop)

	c, err := NewClient("osm", tests.OsmMeshConfigName, broker,
		WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
		WithKubeClient(fake.NewSimpleClientset(nsObj), testMeshName),
	)
	a.NoError(err)
	// Verify
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	actual := c.GetHTTPRouteGroup(key)
	a.Equal(obj, actual)

	invalid := c.GetHTTPRouteGroup("invalid")
	a.Nil(invalid)
}

func TestListTCPTrafficSpecs(t *testing.T) {
	nsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: testMeshName,
			},
		},
	}

	obj := &smiSpecs.TCPRoute{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha4",
			Kind:       "TCPRoute",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNs,
			Name:      "tcp-route",
		},
		Spec: smiSpecs.TCPRouteSpec{},
	}
	a := assert.New(t)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset(obj)
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	stop := make(chan struct{})
	broker := messaging.NewBroker(stop)

	c, err := NewClient("osm", tests.OsmMeshConfigName, broker,
		WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
		WithKubeClient(fake.NewSimpleClientset(nsObj), testMeshName),
	)
	a.NoError(err)
	// Verify
	actual := c.ListTCPTrafficSpecs()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])
}

func TestGetTCPRoute(t *testing.T) {
	nsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
			Labels: map[string]string{
				constants.OSMKubeResourceMonitorAnnotation: testMeshName,
			},
		},
	}

	obj := &smiSpecs.TCPRoute{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha4",
			Kind:       "TCPRoute",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNs,
			Name:      "tcp-route",
		},
		Spec: smiSpecs.TCPRouteSpec{},
	}
	a := assert.New(t)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset(obj)
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	stop := make(chan struct{})
	broker := messaging.NewBroker(stop)

	c, err := NewClient("osm", tests.OsmMeshConfigName, broker, WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
		WithKubeClient(fake.NewSimpleClientset(nsObj), testMeshName),
	)
	a.NoError(err)

	// Verify
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	actual := c.GetTCPRoute(key)
	a.Equal(obj, actual)

	invalid := c.GetTCPRoute("invalid")
	a.Nil(invalid)
}

func TestListServiceImports(t *testing.T) {
	a := assert.New(t)

	obj := &mcs.ServiceImport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNs,
			Name:      "test-service-import",
		},
	}

	mcsClientSet := mcsClientFake.NewSimpleClientset(obj)
	stop := make(chan struct{})
	broker := messaging.NewBroker(stop)

	c, err := NewClient("osm-system", "osm-mesh-config", broker,
		WithKubeClient(fake.NewSimpleClientset(monitoredNS(testNs)), testMeshName),
		WithMCSClient(mcsClientSet),
	)
	a.NoError(err)

	// Verify
	actual := c.ListServiceImports()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])
}

func TestListServiceExports(t *testing.T) {
	a := assert.New(t)
	obj := &mcs.ServiceExport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNs,
			Name:      "test-service-export",
		},
	}
	mcsClientSet := mcsClientFake.NewSimpleClientset(obj)

	stop := make(chan struct{})
	broker := messaging.NewBroker(stop)

	c, err := NewClient("osm-system", "osm-mesh-config", broker,
		WithKubeClient(fake.NewSimpleClientset(monitoredNS(testNs)), testMeshName),
		WithMCSClient(mcsClientSet),
	)
	a.NoError(err)

	// Verify
	actual := c.ListServiceExports()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])
}
