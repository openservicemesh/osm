package policy

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	fakePolicyClient "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
)

func TestNewPolicyClient(t *testing.T) {
	assert := tassert.New(t)

	client, err := newPolicyClient(fakePolicyClient.NewSimpleClientset(), nil, nil)
	assert.Nil(err)
	assert.NotNil(client)
	assert.NotNil(client.informers.egress)
	assert.NotNil(client.caches.egress)
}

func TestListEgressPoliciesForSourceIdentity(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockKubeController.EXPECT().IsMonitoredNamespace("test").Return(true).AnyTimes()

	stop := make(chan struct{})

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
						Sources: []policyV1alpha1.SourceSpec{
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
						Sources: []policyV1alpha1.SourceSpec{
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
						Sources: []policyV1alpha1.SourceSpec{
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
			assert := tassert.New(t)

			fakepolicyClientSet := fakePolicyClient.NewSimpleClientset()

			// Create fake egress policies
			for _, egressPolicy := range tc.allEgresses {
				_, err := fakepolicyClientSet.PolicyV1alpha1().Egresses(egressPolicy.Namespace).Create(context.TODO(), egressPolicy, metav1.CreateOptions{})
				assert.Nil(err)
			}

			policyClient, err := newPolicyClient(fakepolicyClientSet, mockKubeController, stop)
			assert.Nil(err)
			assert.NotNil(policyClient)

			actual := policyClient.ListEgressPoliciesForSourceIdentity(tc.source)
			assert.ElementsMatch(tc.expectedEgresses, actual)
		})
	}
}
