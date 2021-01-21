package catalog

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestListAllowedInboundServiceAccounts(t *testing.T) {
	assert := tassert.New(t)
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
						APIVersion: "access.smi-spec.io/v1alpha3",
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
						APIVersion: "access.smi-spec.io/v1alpha3",
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
						APIVersion: "access.smi-spec.io/v1alpha3",
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
						APIVersion: "access.smi-spec.io/v1alpha3",
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
	assert := tassert.New(t)
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
						APIVersion: "access.smi-spec.io/v1alpha3",
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
						APIVersion: "access.smi-spec.io/v1alpha3",
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
						APIVersion: "access.smi-spec.io/v1alpha3",
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
						APIVersion: "access.smi-spec.io/v1alpha3",
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
	assert := tassert.New(t)

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
	assert := tassert.New(t)
	input := []smiAccess.IdentityBindingSubject{
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

func TestListInboundTrafficTargetsWithRoutes(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testCases := []struct {
		name               string
		trafficTargets     []*smiAccess.TrafficTarget
		tcpRoutes          map[string]*smiSpecs.TCPRoute
		upstreamSvcAccount service.K8sServiceAccount

		expectedTrafficTargets []trafficpolicy.TrafficTargetWithRoutes
		expectError            bool
	}{
		// Test case 1 begin ------------------------------------
		{
			name: "Single traffic target with single TCP route rule",
			trafficTargets: []*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha3",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-1",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-2",
							Namespace: "ns-2",
						}},
						Rules: []smiAccess.TrafficTargetRule{
							{
								Kind: "TCPRoute",
								Name: "route-1",
							},
						},
					},
				},
			},

			// Each route in this list corresponds to a TCPRoute
			tcpRoutes: map[string]*smiSpecs.TCPRoute{
				"ns-1/route-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-1",
						Namespace: "ns-1",
					},
					Spec: smiSpecs.TCPRouteSpec{
						Matches: smiSpecs.TCPMatch{
							Name:  "8000,9000",
							Ports: []int{8000, 9000},
						},
					},
				},
			},

			upstreamSvcAccount: service.K8sServiceAccount{Namespace: "ns-1", Name: "sa-1"},

			expectedTrafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
					},
					TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
						{
							Ports: []int{8000, 9000},
						},
					},
				},
			},

			expectError: false, // no errors expected
		},
		// Test case 1 end ------------------------------------

		// Test case 2 begin ------------------------------------
		{
			name: "Single traffic target with multiple TCP route rules",
			trafficTargets: []*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha3",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-1",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-2",
							Namespace: "ns-2",
						}},
						Rules: []smiAccess.TrafficTargetRule{
							{
								Kind: "TCPRoute",
								Name: "route-1",
							},
							{
								Kind: "TCPRoute",
								Name: "route-2",
							},
						},
					},
				},
			},

			// Each route in this list corresponds to a TCPRoute
			tcpRoutes: map[string]*smiSpecs.TCPRoute{
				"ns-1/route-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-1",
						Namespace: "ns-1",
					},
					Spec: smiSpecs.TCPRouteSpec{
						Matches: smiSpecs.TCPMatch{
							Name:  "8000",
							Ports: []int{8000},
						},
					},
				},
				"ns-1/route-2": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-2",
						Namespace: "ns-1",
					},
					Spec: smiSpecs.TCPRouteSpec{
						Matches: smiSpecs.TCPMatch{
							Name:  "9000",
							Ports: []int{9000},
						},
					},
				},
			},

			upstreamSvcAccount: service.K8sServiceAccount{Namespace: "ns-1", Name: "sa-1"},

			expectedTrafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
					},
					TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
						{
							// route-1
							Ports: []int{8000},
						},
						{
							// route-2
							Ports: []int{9000},
						},
					},
				},
			},

			expectError: false, // no errors expected
		},
		// Test case 2 end ------------------------------------

		// Test case 3 begin ------------------------------------
		{
			name: "Multiple traffic target with multiple TCP route rules",
			trafficTargets: []*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha3",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-1",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-2",
							Namespace: "ns-2",
						}},
						Rules: []smiAccess.TrafficTargetRule{
							{
								Kind: "TCPRoute",
								Name: "route-1",
							},
							{
								Kind: "TCPRoute",
								Name: "route-2",
							},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha3",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-2",
						Namespace: "ns-1",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-3",
							Namespace: "ns-3",
						}},
						Rules: []smiAccess.TrafficTargetRule{
							{
								Kind: "TCPRoute",
								Name: "route-3",
							},
							{
								Kind: "TCPRoute",
								Name: "route-4",
							},
						},
					},
				},
			},

			// Each route in this list corresponds to a TCPRoute
			tcpRoutes: map[string]*smiSpecs.TCPRoute{
				"ns-1/route-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-1",
						Namespace: "ns-1",
					},
					Spec: smiSpecs.TCPRouteSpec{
						Matches: smiSpecs.TCPMatch{
							Name:  "1000",
							Ports: []int{1000},
						},
					},
				},
				"ns-1/route-2": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-2",
						Namespace: "ns-1",
					},
					Spec: smiSpecs.TCPRouteSpec{
						Matches: smiSpecs.TCPMatch{
							Name:  "2000",
							Ports: []int{2000},
						},
					},
				},
				"ns-1/route-3": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-3",
						Namespace: "ns-1",
					},
					Spec: smiSpecs.TCPRouteSpec{
						Matches: smiSpecs.TCPMatch{
							Name:  "3000",
							Ports: []int{3000},
						},
					},
				},
				"ns-1/route-4": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-4",
						Namespace: "ns-1",
					},
					Spec: smiSpecs.TCPRouteSpec{
						Matches: smiSpecs.TCPMatch{
							Name:  "4000",
							Ports: []int{4000},
						},
					},
				},
			},

			upstreamSvcAccount: service.K8sServiceAccount{Namespace: "ns-1", Name: "sa-1"},

			expectedTrafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
					},
					TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
						{
							// route-1
							Ports: []int{1000},
						},
						{
							// route-2
							Ports: []int{2000},
						},
					},
				},
				{
					Name:        "ns-1/test-2",
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-3.ns-3.cluster.local"),
					},
					TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
						{
							// route-3
							Ports: []int{3000},
						},
						{
							// route-4
							Ports: []int{4000},
						},
					},
				},
			},

			expectError: false, // no errors expected
		},
		// Test case 3 end ------------------------------------

		// Test case 4 begin ------------------------------------
		{
			name: "Single traffic target with single TCP route rule without ports specified",
			trafficTargets: []*smiAccess.TrafficTarget{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "access.smi-spec.io/v1alpha3",
						Kind:       "TrafficTarget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "ns-1",
					},
					Spec: smiAccess.TrafficTargetSpec{
						Destination: smiAccess.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      "sa-1",
							Namespace: "ns-1",
						},
						Sources: []smiAccess.IdentityBindingSubject{{
							Kind:      "ServiceAccount",
							Name:      "sa-2",
							Namespace: "ns-2",
						}},
						Rules: []smiAccess.TrafficTargetRule{
							{
								Kind: "TCPRoute",
								Name: "route-1",
							},
						},
					},
				},
			},

			// Each route in this list corresponds to a TCPRoute
			tcpRoutes: map[string]*smiSpecs.TCPRoute{
				"ns-1/route-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-1",
						Namespace: "ns-1",
					},
				},
			},

			upstreamSvcAccount: service.K8sServiceAccount{Namespace: "ns-1", Name: "sa-1"},

			expectedTrafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1.cluster.local"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2.cluster.local"),
					},
					TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
						{
							Ports: nil,
						},
					},
				},
			},

			expectError: false, // no errors expected
		},
		// Test case 4 end ------------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			// Initialize test objects
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockCfg := configurator.NewMockConfigurator(mockCtrl)
			meshCatalog := MeshCatalog{
				meshSpec:     mockMeshSpec,
				configurator: mockCfg,
			}

			mockCfg.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

			// Mock TrafficTargets returned by MeshSpec, should return all TrafficTargets relevant for this test
			mockMeshSpec.EXPECT().ListTrafficTargets().Return(tc.trafficTargets).AnyTimes()
			for _, trafficTarget := range tc.trafficTargets {
				for _, rule := range trafficTarget.Spec.Rules {
					if rule.Kind != tcpRouteKind {
						continue
					}
					routeName := fmt.Sprintf("%s/%s", trafficTarget.Spec.Destination.Namespace, rule.Name)
					mockMeshSpec.EXPECT().GetTCPRoute(routeName).Return(tc.tcpRoutes[routeName]).AnyTimes()
				}
			}

			actual, err := meshCatalog.ListInboundTrafficTargetsWithRoutes(tc.upstreamSvcAccount)
			assert.Equal(err != nil, tc.expectError)
			assert.ElementsMatch(tc.expectedTrafficTargets, actual)
		})
	}
}
