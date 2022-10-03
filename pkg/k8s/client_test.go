package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiAccessClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	smiSpecClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned/fake"
	smiSplitClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	fakeConfigClient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	fakePolicyClient "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
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
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
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
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
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

func TestListNamespaces(t *testing.T) {
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
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
			for _, ns := range tc.namespaces {
				_ = ic.Add(informers.InformerKeyNamespace, ns, t)
			}

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
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
			_ = ic.Add(informers.InformerKeyService, tc.service, t)

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
		secrets  []*corev1.Secret
		expected []*models.Secret
	}{
		{
			name: "get multiple k8s secrets",
			secrets: []*corev1.Secret{
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
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "s1",
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
			name:     "no k8s secret",
			secrets:  []*corev1.Secret{},
			expected: []*models.Secret{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)

			for _, s := range tc.secrets {
				_ = ic.Add(informers.InformerKeySecret, s, t)
			}

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
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
			err = ic.Add(informers.InformerKeySecret, tc.secret, t)
			a.Nil(err)

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
					Namespace: "ns1"},
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
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)

			err = ic.Add(informers.InformerKeySecret, tc.corev1Secret, t)
			a.Nil(err)

			fakeK8sClient := fake.NewSimpleClientset()
			c := NewClient(testNs, tests.OsmMeshConfigName, ic, fakeK8sClient, nil, nil, nil)

			_, err = c.kubeClient.CoreV1().Secrets("ns1").Create(context.Background(), tc.corev1Secret, metav1.CreateOptions{})
			a.Nil(err)

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
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
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
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
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
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
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
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(testclient.NewSimpleClientset()))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
			_ = ic.Add(informers.InformerKeyEndpoints, tc.endpoints, t)

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
			kubeClient := testclient.NewSimpleClientset()
			policyClient := fakePolicyClient.NewSimpleClientset(tc.existingResource.(runtime.Object))
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithKubeClient(kubeClient), informers.WithPolicyClient(policyClient))
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, ic, kubeClient, policyClient, nil, nil)
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
					State: constants.MRCStateActive,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			kubeClient := testclient.NewSimpleClientset()
			configClient := fakeConfigClient.NewSimpleClientset(tc.existingResource.(runtime.Object))
			ic, err := informers.NewInformerCollection(tests.MeshName, nil, informers.WithKubeClient(kubeClient))
			a.Nil(err)
			c := NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, ic, nil, nil, configClient, nil)
			switch v := tc.updatedResource.(type) {
			case *configv1alpha2.MeshRootCertificate:
				_, err = c.UpdateMeshRootCertificateStatus(v)
				a.NoError(err)
			}
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

	kubeController := NewClient("osm", tests.OsmMeshConfigName, ic, nil, nil, nil, messaging.NewBroker(nil))

	testCases := []struct {
		name  string
		pod   *corev1.Pod
		proxy *models.Proxy
		err   error
	}{
		{
			name:  "fails when UUID does not match",
			proxy: models.NewProxy(models.KindSidecar, uuid.New(), tests.BookstoreServiceIdentity, nil, 1),
			err:   errDidNotFindPodForUUID,
		},
		{
			name:  "fails when service account does not match certificate",
			proxy: &models.Proxy{UUID: proxyUUID, Identity: identity.New("bad-name", namespace)},
			err:   errServiceAccountDoesNotMatchProxy,
		},
		{
			name:  "2 pods with same uuid",
			proxy: models.NewProxy(models.KindSidecar, someOtherEnvoyUID, tests.BookstoreServiceIdentity, nil, 1),
			err:   errMoreThanOnePodForUUID,
		},
		{
			name:  "fails when namespace does not match certificate",
			proxy: models.NewProxy(models.KindSidecar, proxyUUID, identity.New(tests.BookstoreServiceAccountName, "bad-namespace"), nil, 1),
			err:   errNamespaceDoesNotMatchProxy,
		},
		{
			name:  "works as expected",
			pod:   pod,
			proxy: models.NewProxy(models.KindSidecar, proxyUUID, tests.BookstoreServiceIdentity, nil, 1),
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

func TestGetMeshConfig(t *testing.T) {
	a := assert.New(t)

	meshConfigClient := fakeConfigClient.NewSimpleClientset()
	stop := make(chan struct{})
	osmNamespace := "osm"
	osmMeshConfigName := "osm-mesh-config"

	ic, err := informers.NewInformerCollection("osm", stop, informers.WithConfigClient(meshConfigClient, osmMeshConfigName, osmNamespace))
	a.Nil(err)

	c := NewClient(osmNamespace, tests.OsmMeshConfigName, ic, nil, nil, nil, nil)

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

func TestListEgressPolicies(t *testing.T) {
	egressNsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
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
		allEgresses      []*policyv1alpha1.Egress
		expectedEgresses []*policyv1alpha1.Egress
	}{
		{
			name:             "Only return egress resources for monitored namespaces",
			allEgresses:      []*policyv1alpha1.Egress{inMeshResource, outMeshResource},
			expectedEgresses: []*policyv1alpha1.Egress{inMeshResource},
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

			c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, nil, fakeClient, nil, nil)
			a.Nil(err)
			a.NotNil(c)

			// monitor namespaces
			err = c.informers.Add(informers.InformerKeyNamespace, egressNsObj, t)
			a.Nil(err)

			// Create fake egress policies
			for _, egressPolicy := range tc.allEgresses {
				_ = c.informers.Add(informers.InformerKeyEgress, egressPolicy, t)
			}

			policies := c.ListEgressPolicies()
			a.ElementsMatch(tc.expectedEgresses, policies)
		})
	}
}

func TestListRetryPolicy(t *testing.T) {
	policyNsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
		},
	}

	var thresholdUintVal uint32 = 3
	thresholdTimeoutDuration := metav1.Duration{Duration: time.Duration(5 * time.Second)}
	thresholdBackoffDuration := metav1.Duration{Duration: time.Duration(1 * time.Second)}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := NewMockController(mockCtrl)
	mockKubeController.EXPECT().IsMonitoredNamespace(testNs).Return(true).AnyTimes()

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
		allRetries      []*policyv1alpha1.Retry
		expectedRetries []*policyv1alpha1.Retry
	}{
		{
			name:            "Only return retry resources for monitored namespaces",
			allRetries:      []*policyv1alpha1.Retry{inMeshResource, outMeshResource},
			expectedRetries: []*policyv1alpha1.Retry{inMeshResource},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			informerCollection, err := informers.NewInformerCollection("osm", nil,
				informers.WithPolicyClient(fakeClient),
				informers.WithKubeClient(testclient.NewSimpleClientset()),
			)
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, nil, fakeClient, nil, nil)
			a.Nil(err)
			a.NotNil(c)

			// monitor namespaces
			err = c.informers.Add(informers.InformerKeyNamespace, policyNsObj, t)
			a.Nil(err)

			// Create fake retry policies
			for _, retryPolicy := range tc.allRetries {
				err := c.informers.Add(informers.InformerKeyRetry, retryPolicy, t)
				a.Nil(err)
			}

			policies := c.ListRetryPolicies()
			a.ElementsMatch(tc.expectedRetries, policies)
		})
	}
}

func TestListUpstreamTrafficSetting(t *testing.T) {
	settingNsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
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
		allResources []*policyv1alpha1.UpstreamTrafficSetting
		expected     []*policyv1alpha1.UpstreamTrafficSetting
	}{
		{
			name:         "Only return upstream traffic settings for monitored namespaces",
			allResources: []*policyv1alpha1.UpstreamTrafficSetting{inMeshResource, outMeshResource},
			expected:     []*policyv1alpha1.UpstreamTrafficSetting{inMeshResource},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			informerCollection, err := informers.NewInformerCollection("osm", nil,
				informers.WithPolicyClient(fakeClient),
				informers.WithKubeClient(testclient.NewSimpleClientset()),
			)
			a.Nil(err)
			c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, nil, fakeClient, nil, nil)
			a.Nil(err)
			a.NotNil(c)

			// monitor namespaces
			err = c.informers.Add(informers.InformerKeyNamespace, settingNsObj, t)
			a.Nil(err)

			// Create fake upstream traffic settings
			for _, resource := range tc.allResources {
				_ = c.informers.Add(informers.InformerKeyUpstreamTrafficSetting, resource, t)
			}

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
			ic, err := informers.NewInformerCollection(testMeshName, nil, informers.WithConfigClient(fakeConfigClient.NewSimpleClientset(), tests.OsmMeshConfigName, tests.OsmNamespace))
			a.Nil(err)
			c := NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
			_ = ic.Add(informers.InformerKeyMeshRootCertificate, tc.meshRootCertificate, t)

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

	mrcClient := fakeConfigClient.NewSimpleClientset()
	stop := make(chan struct{})

	ic, err := informers.NewInformerCollection(tests.MeshName, stop, informers.WithConfigClient(mrcClient, tests.OsmMeshConfigName, tests.OsmNamespace))
	a.Nil(err)

	c := NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, ic, nil, nil, nil, nil)

	mrcList, err := c.ListMeshRootCertificates()
	a.NoError(err)
	a.Empty(mrcList)

	newList := []*configv1alpha2.MeshRootCertificate{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "osm-mesh-root-certificate",
				Namespace: "osm-system",
			},
			Spec: configv1alpha2.MeshRootCertificateSpec{
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
				State: constants.MRCStateActive,
			},
		},
	}
	err = c.informers.Add(informers.InformerKeyMeshRootCertificate, newList[0], t)
	a.Nil(err)

	mrcList, err = c.ListMeshRootCertificates()
	a.NoError(err)
	a.ElementsMatch(newList, mrcList)
}

func TestListHTTPTrafficSpecs(t *testing.T) {
	nsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
		},
	}

	a := assert.New(t)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset()
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()

	informerCollection, err := informers.NewInformerCollection("osm", nil,
		informers.WithKubeClient(testclient.NewSimpleClientset()),
		informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
	)
	a.Nil(err)
	c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, nil, nil, nil, nil)
	a.Nil(err)
	a.NotNil(c)

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
	err = c.informers.Add(informers.InformerKeyNamespace, nsObj, t)
	a.Nil(err)
	err = c.informers.Add(informers.InformerKeyHTTPRouteGroup, obj, t)
	a.Nil(err)

	// Verify
	actual := c.ListHTTPTrafficSpecs()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])
}

func TestGetHTTPRouteGroup(t *testing.T) {
	nsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
		},
	}

	a := assert.New(t)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset()
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	informerCollection, err := informers.NewInformerCollection("osm", nil,
		informers.WithKubeClient(testclient.NewSimpleClientset()),
		informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
	)
	a.Nil(err)
	c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, nil, nil, nil, nil)
	a.Nil(err)
	a.NotNil(c)

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
	err = c.informers.Add(informers.InformerKeyNamespace, nsObj, t)
	a.Nil(err)
	err = c.informers.Add(informers.InformerKeyHTTPRouteGroup, obj, t)
	a.Nil(err)

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
		},
	}

	a := assert.New(t)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset()
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	informerCollection, err := informers.NewInformerCollection("osm", nil,
		informers.WithKubeClient(testclient.NewSimpleClientset()),
		informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
	)

	a.Nil(err)
	c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, nil, nil, nil, nil)
	a.Nil(err)
	a.NotNil(c)

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
	err = c.informers.Add(informers.InformerKeyNamespace, nsObj, t)
	a.Nil(err)
	err = c.informers.Add(informers.InformerKeyTCPRoute, obj, t)
	a.Nil(err)

	// Verify
	actual := c.ListTCPTrafficSpecs()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])
}

func TestGetTCPRoute(t *testing.T) {
	nsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNs,
		},
	}
	a := assert.New(t)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset()
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	informerCollection, err := informers.NewInformerCollection("osm", nil,
		informers.WithKubeClient(testclient.NewSimpleClientset()),
		informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
	)
	a.Nil(err)
	c := NewClient("osm", tests.OsmMeshConfigName, informerCollection, nil, nil, nil, nil)
	a.Nil(err)
	a.NotNil(c)

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
	err = c.informers.Add(informers.InformerKeyNamespace, nsObj, t)
	a.Nil(err)
	err = c.informers.Add(informers.InformerKeyTCPRoute, obj, t)
	a.Nil(err)

	// Verify
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	actual := c.GetTCPRoute(key)
	a.Equal(obj, actual)

	invalid := c.GetTCPRoute("invalid")
	a.Nil(invalid)
}

func TestGetTelemetryPolicy(t *testing.T) {
	proxyUUID := uuid.New()
	appNamespace := "test"
	osmNamespace := "global"

	globalPolicy := &policyv1alpha1.Telemetry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      "t1",
		},
	}

	namespacePolicy := &policyv1alpha1.Telemetry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: appNamespace,
			Name:      "t2",
		},
	}

	selectorPolicy := &policyv1alpha1.Telemetry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: appNamespace,
			Name:      "t2",
		},
		Spec: policyv1alpha1.TelemetrySpec{
			Selector: map[string]string{"app": "foo"},
		},
	}

	testCases := []struct {
		name              string
		proxy             *models.Proxy
		pod               *corev1.Pod
		telemetryPolicies []*policyv1alpha1.Telemetry
		expected          *policyv1alpha1.Telemetry
	}{
		{
			name:  "matches global scope policy",
			proxy: models.NewProxy(models.KindSidecar, proxyUUID, "sa-1.test", nil, 1),
			pod: tests.NewPodFixture(appNamespace, "pod-1", "sa-1",
				map[string]string{
					constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
					"app":                            "foo",
				}),
			telemetryPolicies: []*policyv1alpha1.Telemetry{
				globalPolicy,
			},
			expected: globalPolicy,
		},
		{
			name:  "matches namespace scope policy",
			proxy: models.NewProxy(models.KindSidecar, proxyUUID, "sa-1.test", nil, 1),
			pod: tests.NewPodFixture(appNamespace, "pod-1", "sa-1",
				map[string]string{
					constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
					"app":                            "foo",
				}),
			telemetryPolicies: []*policyv1alpha1.Telemetry{
				globalPolicy,
				namespacePolicy,
			},
			expected: namespacePolicy,
		},
		{
			name:  "matches selector scope policy",
			proxy: models.NewProxy(models.KindSidecar, proxyUUID, "sa-1.test", nil, 1),
			pod: tests.NewPodFixture(appNamespace, "pod-1", "sa-1",
				map[string]string{
					constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
					"app":                            "foo",
				}),
			telemetryPolicies: []*policyv1alpha1.Telemetry{
				globalPolicy,
				namespacePolicy,
				selectorPolicy,
			},
			expected: selectorPolicy,
		},
		{
			name:  "no policy when proxy does not match pod",
			proxy: models.NewProxy(models.KindSidecar, uuid.New(), "sa-1.test", nil, 1), // new UUID to avoid matching proxyUUID
			pod: tests.NewPodFixture(appNamespace, "pod-1", "sa-1",
				map[string]string{
					constants.EnvoyUniqueIDLabelName: proxyUUID.String(),
					"app":                            "foo",
				}),
			telemetryPolicies: []*policyv1alpha1.Telemetry{
				globalPolicy,
				namespacePolicy,
				selectorPolicy,
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fake.NewSimpleClientset()
			fakePolicyClient := fakePolicyClient.NewSimpleClientset()
			ic, err := informers.NewInformerCollection(testMeshName, nil,
				informers.WithKubeClient(fakeClient), informers.WithPolicyClient(fakePolicyClient))
			a.Nil(err)

			c := NewClient(osmNamespace, tests.OsmMeshConfigName, ic, nil, nil, nil, nil)
			a.NotNil(c)

			_ = ic.Add(informers.InformerKeyNamespace, monitoredNS(appNamespace), t)
			_ = ic.Add(informers.InformerKeyPod, tc.pod, t)

			for _, policy := range tc.telemetryPolicies {
				_ = ic.Add(informers.InformerKeyTelemetry, policy, t)
			}

			actual := c.GetTelemetryPolicy(tc.proxy)
			a.Equal(tc.expected, actual)
		})
	}
}
