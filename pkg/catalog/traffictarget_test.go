package catalog

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
)

func TestListAllowedInboundServiceAccounts(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
	meshCatalog := MeshCatalog{
		meshSpec: mockMeshSpec,
	}

	testCases := []struct {
		trafficTargets             []*smiAccess.TrafficTarget
		svcAccount                 service.K8sServiceAccount
		expectedInboundSvcAccounts []service.K8sServiceAccount
		expectError                bool
	}{
		// Test case 1 begin ------------------------------------
		// There is a valid inbound service account
		{
			[]*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha2",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-2",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-2",
							Namespace: "ns-2",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						}},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha2",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-2",
						Namespace: "ns-3",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-3",
							Namespace: "ns-3",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						}},
					},
				},
			},

			// given service account to test
			service.K8sServiceAccount{
				Name:      "sa-2",
				Namespace: "ns-2",
			},

			// allowed inbound service accounts: 1 match
			[]service.K8sServiceAccount{
				{
					Name:      "sa-1",
					Namespace: "ns-1",
				},
			},

			false, // no errors expected
		},
		// Test case 1 end ------------------------------------

		// Test case 2 begin ------------------------------------
		// There are no inbound service accounts
		{
			[]*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha2",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-2",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-2",
							Namespace: "ns-2",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						}},
					},
				},
			},

			// given service account to test
			service.K8sServiceAccount{
				Name:      "sa-1",
				Namespace: "ns-1",
			},

			// allowed inbound service accounts: no match
			nil,

			false, // no errors expected
		},
		// Test case 2 end ------------------------------------

		// Test case 3 begin ------------------------------------
		// Error due to invalid kind for Destination
		{
			[]*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha2",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-2",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "foo", // Invalid kind
							Name:      "sa-2",
							Namespace: "ns-2",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						}},
					},
				},
			},

			// given service account to test
			service.K8sServiceAccount{
				Name:      "sa-1",
				Namespace: "ns-1",
			},

			// allowed inbound service accounts: no match
			nil,

			false, // will log an error but function will ignore policy with error
		},
		// Test case 3 end ------------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			// Mock TrafficTargets returned by MeshSpec, should return all TrafficTargets relevant for this test
			mockMeshSpec.EXPECT().ListTrafficTargets().Return(tc.trafficTargets).Times(1)

			actual, err := meshCatalog.ListAllowedInboundServiceAccounts(tc.svcAccount)
			assert.Equal(err != nil, tc.expectError)
			assert.ElementsMatch(actual, tc.expectedInboundSvcAccounts)
		})
	}
}

func TestListAllowedOutboundServiceAccounts(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
	meshCatalog := MeshCatalog{
		meshSpec: mockMeshSpec,
	}

	testCases := []struct {
		trafficTargets              []*smiAccess.TrafficTarget
		svcAccount                  service.K8sServiceAccount
		expectedOutboundSvcAccounts []service.K8sServiceAccount
		expectError                 bool
	}{
		// Test case 1 begin ------------------------------------
		// There is a valid outbound service account
		{
			[]*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha2",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-2",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-2",
							Namespace: "ns-2",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						}},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha2",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-2",
						Namespace: "ns-3",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-3",
							Namespace: "ns-3",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						}},
					},
				},
			},

			// given service account to test
			service.K8sServiceAccount{
				Name:      "sa-1",
				Namespace: "ns-1",
			},

			// allowed inbound service accounts: 2 matches
			[]service.K8sServiceAccount{
				{
					Name:      "sa-2",
					Namespace: "ns-2",
				},
				{
					Name:      "sa-3",
					Namespace: "ns-3",
				},
			},

			false, // no errors expected
		},
		// Test case 1 end ------------------------------------

		// Test case 2 begin ------------------------------------
		// There are no outbound service accounts
		{
			[]*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha2",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-2",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-2",
							Namespace: "ns-2",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						}},
					},
				},
			},

			// given service account to test
			service.K8sServiceAccount{
				Name:      "sa-2",
				Namespace: "ns-2",
			},

			// allowed inbound service accounts: no match
			nil,

			false, // no errors expected
		},
		// Test case 2 end ------------------------------------

		// Test case 3 begin ------------------------------------
		// Error due to invalid kind for Source
		{
			[]*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha2",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-2",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-2",
							Namespace: "ns-2",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "foo", // Invalid kind
							Name:      "sa-1",
							Namespace: "ns-1",
						}},
					},
				},
			},

			// given service account to test
			service.K8sServiceAccount{
				Name:      "sa-1",
				Namespace: "ns-1",
			},

			// allowed inbound service accounts: no match
			nil,

			false, // will log an error but function will ignore policy with error
		},
		// Test case 3 end ------------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			// Mock TrafficTargets returned by MeshSpec, should return all TrafficTargets relevant for this test
			mockMeshSpec.EXPECT().ListTrafficTargets().Return(tc.trafficTargets).Times(1)

			actual, err := meshCatalog.ListAllowedOutboundServiceAccounts(tc.svcAccount)
			assert.Equal(err != nil, tc.expectError)
			assert.ElementsMatch(actual, tc.expectedOutboundSvcAccounts)
		})
	}
}

func TestTrafficTargetIdentityToSvcAccount(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		identity               smiAccess.IdentityBindingSubject
		expectedServiceAccount service.K8sServiceAccount
	}{
		{
			smiAccess.IdentityBindingSubject{
				Kind:      "ServiceAccount",
				Name:      "sa-2",
				Namespace: "ns-2",
			},
			service.K8sServiceAccount{
				Name:      "sa-2",
				Namespace: "ns-2",
			},
		},
		{
			smiAccess.IdentityBindingSubject{
				Kind:      "ServiceAccount",
				Name:      "sa-1",
				Namespace: "ns-1",
			},
			service.K8sServiceAccount{
				Name:      "sa-1",
				Namespace: "ns-1",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			svcAccount := trafficTargetIdentityToSvcAccount(tc.identity)
			assert.Equal(svcAccount, tc.expectedServiceAccount)
		})
	}
}

func TestTrafficTargetIdentitiesToSvcAccounts(t *testing.T) {
	assert := assert.New(t)
	input := []target.IdentityBindingSubject{
		{
			Kind:      "ServiceAccount",
			Name:      "example1",
			Namespace: "default1",
		},
		{
			Kind:      "Name",
			Name:      "example2",
			Namespace: "default2",
		},
	}

	expected := []service.K8sServiceAccount{
		{
			Name:      "example1",
			Namespace: "default1",
		},
		{
			Name:      "example2",
			Namespace: "default2",
		},
	}

	actual := trafficTargetIdentitiesToSvcAccounts(input)
	assert.ElementsMatch(expected, actual)
}
