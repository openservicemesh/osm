package policy

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	fakePolicyClient "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
)

func TestListEgressPoliciesForSourceIdentity(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockKubeController.EXPECT().IsMonitoredNamespace("test").Return(true).AnyTimes()

	testCases := []struct {
		name             string
		allEgresses      []*policyV1alpha1.Egress
		source           identity.K8sServiceAccount
		expectedEgresses []*policyV1alpha1.Egress
	}{
		{
			name: "matching egress policy not found for source identity test/sa-3",
			allEgresses: []*policyV1alpha1.Egress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "egress-1",
						Namespace: "test",
					},
					Spec: policyV1alpha1.EgressSpec{
						Sources: []policyV1alpha1.EgressSourceSpec{
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
						Ports: []policyV1alpha1.PortSpec{
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
			allEgresses: []*policyV1alpha1.Egress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "egress-1",
						Namespace: "test",
					},
					Spec: policyV1alpha1.EgressSpec{
						Sources: []policyV1alpha1.EgressSourceSpec{
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
						Ports: []policyV1alpha1.PortSpec{
							{
								Number:   80,
								Protocol: "http",
							},
						},
					},
				},
			},
			source: identity.K8sServiceAccount{Name: "sa-1", Namespace: "test"},
			expectedEgresses: []*policyV1alpha1.Egress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "egress-1",
						Namespace: "test",
					},
					Spec: policyV1alpha1.EgressSpec{
						Sources: []policyV1alpha1.EgressSourceSpec{
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
						Ports: []policyV1alpha1.PortSpec{
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

			c, err := newClient(mockKubeController, fakePolicyClient.NewSimpleClientset(), nil, nil)
			a.Nil(err)
			a.NotNil(c)

			// Create fake egress policies
			for _, egressPolicy := range tc.allEgresses {
				_ = c.caches.egress.Add(egressPolicy)
			}

			actual := c.ListEgressPoliciesForSourceIdentity(tc.source)
			a.ElementsMatch(tc.expectedEgresses, actual)
		})
	}
}

func TestGetIngressBackendPolicy(t *testing.T) {
	testCases := []struct {
		name                   string
		allResources           []*policyV1alpha1.IngressBackend
		backend                service.MeshService
		expectedIngressBackend *policyV1alpha1.IngressBackend
	}{
		{
			name:                   "IngressBackend policy not found",
			allResources:           nil,
			backend:                service.MeshService{Name: "backend1", Namespace: "test"},
			expectedIngressBackend: nil,
		},
		{
			name: "IngressBackend policy found",
			allResources: []*policyV1alpha1.IngressBackend{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-1",
						Namespace: "test",
					},
					Spec: policyV1alpha1.IngressBackendSpec{
						Backends: []policyV1alpha1.BackendSpec{
							{
								Name: "backend1", // matches the backend specified in the test case
								Port: policyV1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
							{
								Name: "backend2",
								Port: policyV1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
						Sources: []policyV1alpha1.IngressSourceSpec{
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
					Spec: policyV1alpha1.IngressBackendSpec{
						Backends: []policyV1alpha1.BackendSpec{
							{
								Name: "backend2", // does not match the backend specified in the test case
								Port: policyV1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
						Sources: []policyV1alpha1.IngressSourceSpec{
							{
								Kind:      "Service",
								Name:      "client",
								Namespace: "foo",
							},
						},
					},
				},
			},
			backend: service.MeshService{Name: "backend1", Namespace: "test"},
			expectedIngressBackend: &policyV1alpha1.IngressBackend{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-backend-1",
					Namespace: "test",
				},
				Spec: policyV1alpha1.IngressBackendSpec{
					Backends: []policyV1alpha1.BackendSpec{
						{
							Name: "backend1",
							Port: policyV1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
						{
							Name: "backend2",
							Port: policyV1alpha1.PortSpec{
								Number:   80,
								Protocol: "http",
							},
						},
					},
					Sources: []policyV1alpha1.IngressSourceSpec{
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
			allResources: []*policyV1alpha1.IngressBackend{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-backend-1",
						Namespace: "test",
					},
					Spec: policyV1alpha1.IngressBackendSpec{
						Backends: []policyV1alpha1.BackendSpec{
							{
								Name: "backend1", // matches the backend specified in the test case
								Port: policyV1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
							{
								Name: "backend2",
								Port: policyV1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
						Sources: []policyV1alpha1.IngressSourceSpec{
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
					Spec: policyV1alpha1.IngressBackendSpec{
						Backends: []policyV1alpha1.BackendSpec{
							{
								Name: "backend2", // does not match the backend specified in the test case
								Port: policyV1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
						Sources: []policyV1alpha1.IngressSourceSpec{
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

			c, err := newClient(nil, fakePolicyClient.NewSimpleClientset(), nil, nil)
			a.Nil(err)
			a.NotNil(c)

			// Create fake egress policies
			for _, ingressBackend := range tc.allResources {
				_ = c.caches.ingressBackend.Add(ingressBackend)
			}

			actual := c.GetIngressBackendPolicy(tc.backend)
			a.Equal(tc.expectedIngressBackend, actual)
		})
	}
}
