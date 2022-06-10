package policy

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	fakePolicyClient "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s/informers"

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

			fakeClient := fakePolicyClient.NewSimpleClientset()
			informerCollection, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)
			c := NewPolicyController(informerCollection, mockKubeController, nil)
			a.Nil(err)
			a.NotNil(c)

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
								Name: "backend3", // does not match the backend specified in the test case
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
			backend: service.MeshService{Name: "backend1", Namespace: "test", TargetPort: 80, Protocol: "http"},
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

			fakeClient := fakePolicyClient.NewSimpleClientset()
			informerCollection, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)
			c := NewPolicyController(informerCollection, nil, nil)
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

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockKubeController.EXPECT().IsMonitoredNamespace("test").Return(true).AnyTimes()

	testCases := []struct {
		name            string
		allRetries      []*policyV1alpha1.Retry
		source          identity.K8sServiceAccount
		expectedRetries []*policyV1alpha1.Retry
	}{
		{
			name: "matching retry policy not found for source identity test/sa-3",
			allRetries: []*policyV1alpha1.Retry{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "retry-1",
						Namespace: "test",
					},
					Spec: policyV1alpha1.RetrySpec{
						Source: policyV1alpha1.RetrySrcDstSpec{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "test",
						},
						Destinations: []policyV1alpha1.RetrySrcDstSpec{
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
						RetryPolicy: policyV1alpha1.RetryPolicySpec{
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
			allRetries: []*policyV1alpha1.Retry{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "retry-1",
						Namespace: "test",
					},
					Spec: policyV1alpha1.RetrySpec{
						Source: policyV1alpha1.RetrySrcDstSpec{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "test",
						},
						Destinations: []policyV1alpha1.RetrySrcDstSpec{
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
						RetryPolicy: policyV1alpha1.RetryPolicySpec{
							RetryOn:                  "",
							NumRetries:               &thresholdUintVal,
							PerTryTimeout:            &thresholdTimeoutDuration,
							RetryBackoffBaseInterval: &thresholdBackoffDuration,
						},
					},
				},
			},
			source: identity.K8sServiceAccount{Name: "sa-1", Namespace: "test"},
			expectedRetries: []*policyV1alpha1.Retry{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "retry-1",
						Namespace: "test",
					},
					Spec: policyV1alpha1.RetrySpec{
						Source: policyV1alpha1.RetrySrcDstSpec{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "test",
						},
						Destinations: []policyV1alpha1.RetrySrcDstSpec{
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
						RetryPolicy: policyV1alpha1.RetryPolicySpec{
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
			c := NewPolicyController(informerCollection, mockKubeController, nil)
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

func TestGetUpstreamTrafficSetting(t *testing.T) {
	testCases := []struct {
		name         string
		allResources []*policyV1alpha1.UpstreamTrafficSetting
		opt          UpstreamTrafficSettingGetOpt
		expected     *policyV1alpha1.UpstreamTrafficSetting
	}{
		{
			name: "MeshService has matching UpstreamTrafficSetting",
			allResources: []*policyV1alpha1.UpstreamTrafficSetting{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u1",
						Namespace: "ns1",
					},
					Spec: policyV1alpha1.UpstreamTrafficSettingSpec{
						Host: "s1.ns1.svc.cluster.local",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u2",
						Namespace: "ns1",
					},
					Spec: policyV1alpha1.UpstreamTrafficSettingSpec{
						Host: "s2.ns1.svc.cluster.local",
					},
				},
			},
			opt: UpstreamTrafficSettingGetOpt{MeshService: &service.MeshService{Name: "s1", Namespace: "ns1"}},
			expected: &policyV1alpha1.UpstreamTrafficSetting{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "u1",
					Namespace: "ns1",
				},
				Spec: policyV1alpha1.UpstreamTrafficSettingSpec{
					Host: "s1.ns1.svc.cluster.local",
				},
			},
		},
		{
			name: "MeshService that does not match any UpstreamTrafficSetting",
			allResources: []*policyV1alpha1.UpstreamTrafficSetting{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u1",
						Namespace: "ns1",
					},
					Spec: policyV1alpha1.UpstreamTrafficSettingSpec{
						Host: "s1.ns1.svc.cluster.local",
					},
				},
			},
			opt:      UpstreamTrafficSettingGetOpt{MeshService: &service.MeshService{Name: "s3", Namespace: "ns1"}},
			expected: nil,
		},
		{
			name: "UpstreamTrafficSetting namespaced name found",
			allResources: []*policyV1alpha1.UpstreamTrafficSetting{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u1",
						Namespace: "ns1",
					},
					Spec: policyV1alpha1.UpstreamTrafficSettingSpec{
						Host: "s1.ns1.svc.cluster.local",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u2",
						Namespace: "ns1",
					},
					Spec: policyV1alpha1.UpstreamTrafficSettingSpec{
						Host: "s2.ns1.svc.cluster.local",
					},
				},
			},
			opt: UpstreamTrafficSettingGetOpt{NamespacedName: &types.NamespacedName{Namespace: "ns1", Name: "u1"}},
			expected: &policyV1alpha1.UpstreamTrafficSetting{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "u1",
					Namespace: "ns1",
				},
				Spec: policyV1alpha1.UpstreamTrafficSettingSpec{
					Host: "s1.ns1.svc.cluster.local",
				},
			},
		},
		{
			name: "UpstreamTrafficSetting namespaced name not found",
			allResources: []*policyV1alpha1.UpstreamTrafficSetting{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u1",
						Namespace: "ns1",
					},
					Spec: policyV1alpha1.UpstreamTrafficSettingSpec{
						Host: "s1.ns1.svc.cluster.local",
					},
				},
			},
			opt:      UpstreamTrafficSettingGetOpt{NamespacedName: &types.NamespacedName{Namespace: "ns1", Name: "u3"}},
			expected: nil,
		},
		{
			name: "no filter option specified",
			allResources: []*policyV1alpha1.UpstreamTrafficSetting{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "u1",
						Namespace: "ns1",
					},
					Spec: policyV1alpha1.UpstreamTrafficSettingSpec{
						Host: "s1.ns1.svc.cluster.local",
					},
				},
			},
			opt:      UpstreamTrafficSettingGetOpt{},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fakePolicyClient.NewSimpleClientset()
			informerCollection, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient))
			a.Nil(err)
			c := NewPolicyController(informerCollection, nil, nil)
			a.Nil(err)
			a.NotNil(c)

			// Create fake egress policies
			for _, resource := range tc.allResources {
				_ = c.informers.Add(informers.InformerKeyUpstreamTrafficSetting, resource, t)
			}

			actual := c.GetUpstreamTrafficSetting(tc.opt)
			a.Equal(tc.expected, actual)
		})
	}
}
