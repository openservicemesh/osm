package catalog

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	smiAccessClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	smiSpecClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned/fake"
	smiSplitClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fakePolicyClient "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"

	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestListInboundServiceIdentities(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCompute := compute.NewMockInterface(mockCtrl)
	meshCatalog := MeshCatalog{
		Interface: mockCompute,
	}

	testCases := []struct {
		trafficTargets                   []*smiAccess.TrafficTarget
		serviceIdentity                  identity.ServiceIdentity
		expectedInboundServiceIdentities []identity.ServiceIdentity
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
						Rules: []smiAccess.TrafficTargetRule{{
							Kind: "HTTPRouteGroup",
							Name: "rule",
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
						Rules: []smiAccess.TrafficTargetRule{{
							Kind: "HTTPRouteGroup",
							Name: "rule",
						}},
					},
				},
			},

			// given service account to test
			identity.K8sServiceAccount{
				Name:      "sa-2",
				Namespace: "ns-2",
			}.ToServiceIdentity(),

			// allowed inbound service accounts: 1 match
			[]identity.ServiceIdentity{
				identity.K8sServiceAccount{
					Name:      "sa-1",
					Namespace: "ns-1",
				}.ToServiceIdentity(),
			},
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
						Rules: []smiAccess.TrafficTargetRule{{
							Kind: "HTTPRouteGroup",
							Name: "rule",
						}},
					},
				},
			},

			// given service account to test
			identity.K8sServiceAccount{
				Name:      "sa-1",
				Namespace: "ns-1",
			}.ToServiceIdentity(),

			// allowed inbound service accounts: no match
			nil,
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
						Rules: []smiAccess.TrafficTargetRule{{
							Kind: "HTTPRouteGroup",
							Name: "rule",
						}},
					},
				},
			},

			// given service account to test
			identity.K8sServiceAccount{
				Name:      "sa-1",
				Namespace: "ns-1",
			}.ToServiceIdentity(),

			// allowed inbound service accounts: no match
			nil,
		},
		// Test case 3 end ------------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			// Mock TrafficTargets returned by MeshSpec, should return all TrafficTargets relevant for this test
			mockCompute.EXPECT().ListTrafficTargets().Return(tc.trafficTargets).Times(1)

			actual := meshCatalog.ListInboundServiceIdentities(tc.serviceIdentity)
			assert.ElementsMatch(actual, tc.expectedInboundServiceIdentities)
		})
	}
}

func TestListOutboundServiceIdentities(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCompute := compute.NewMockInterface(mockCtrl)
	meshCatalog := MeshCatalog{
		Interface: mockCompute,
	}

	testCases := []struct {
		trafficTargets                    []*smiAccess.TrafficTarget
		serviceIdentity                   identity.ServiceIdentity
		expectedOutboundServiceIdentities []identity.ServiceIdentity
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
						Rules: []smiAccess.TrafficTargetRule{{
							Kind: "HTTPRouteGroup",
							Name: "rule",
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
						Rules: []smiAccess.TrafficTargetRule{{
							Kind: "HTTPRouteGroup",
							Name: "rule",
						}},
					},
				},
			},

			// given service account to test
			identity.K8sServiceAccount{
				Name:      "sa-1",
				Namespace: "ns-1",
			}.ToServiceIdentity(),

			// allowed inbound service accounts: 2 matches
			[]identity.ServiceIdentity{
				identity.K8sServiceAccount{
					Name:      "sa-2",
					Namespace: "ns-2",
				}.ToServiceIdentity(),
				identity.K8sServiceAccount{
					Name:      "sa-3",
					Namespace: "ns-3",
				}.ToServiceIdentity(),
			},
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
						Rules: []smiAccess.TrafficTargetRule{{
							Kind: "HTTPRouteGroup",
							Name: "rule",
						}},
					},
				},
			},

			// given service account to test
			identity.K8sServiceAccount{
				Name:      "sa-2",
				Namespace: "ns-2",
			}.ToServiceIdentity(),

			// allowed inbound service accounts: no match
			nil,
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
						Rules: []smiAccess.TrafficTargetRule{{
							Kind: "HTTPRouteGroup",
							Name: "rule",
						}},
					},
				},
			},

			// given service account to test
			identity.K8sServiceAccount{
				Name:      "sa-1",
				Namespace: "ns-1",
			}.ToServiceIdentity(),

			// allowed inbound service accounts: no match
			nil,
		},
		// Test case 3 end ------------------------------------
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d", i), func(t *testing.T) {
			// Mock TrafficTargets returned by MeshSpec, should return all TrafficTargets relevant for this test
			mockCompute.EXPECT().ListTrafficTargets().Return(tc.trafficTargets).Times(1)

			actual := meshCatalog.ListOutboundServiceIdentities(tc.serviceIdentity)
			assert.ElementsMatch(actual, tc.expectedOutboundServiceIdentities)
		})
	}
}

func TestTrafficTargetIdentityToSvcAccount(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		identity               smiAccess.IdentityBindingSubject
		expectedServiceAccount identity.K8sServiceAccount
	}{
		{
			smiAccess.IdentityBindingSubject{
				Kind:      "ServiceAccount",
				Name:      "sa-2",
				Namespace: "ns-2",
			},
			identity.K8sServiceAccount{
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
			identity.K8sServiceAccount{
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

	expected := []identity.K8sServiceAccount{
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
		name                    string
		trafficTargets          []*smiAccess.TrafficTarget
		tcpRoutes               map[string]*smiSpecs.TCPRoute
		upstreamServiceIdentity identity.ServiceIdentity

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

			upstreamServiceIdentity: identity.K8sServiceAccount{Namespace: "ns-1", Name: "sa-1"}.ToServiceIdentity(),

			expectedTrafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2"),
					},
					TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
						{
							Ports: []uint16{8000, 9000},
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

			upstreamServiceIdentity: identity.K8sServiceAccount{Namespace: "ns-1", Name: "sa-1"}.ToServiceIdentity(),

			expectedTrafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2"),
					},
					TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
						{
							// route-1
							Ports: []uint16{8000},
						},
						{
							// route-2
							Ports: []uint16{9000},
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

			upstreamServiceIdentity: identity.K8sServiceAccount{Namespace: "ns-1", Name: "sa-1"}.ToServiceIdentity(),

			expectedTrafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2"),
					},
					TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
						{
							// route-1
							Ports: []uint16{1000},
						},
						{
							// route-2
							Ports: []uint16{2000},
						},
					},
				},
				{
					Name:        "ns-1/test-2",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-3.ns-3"),
					},
					TCPRouteMatches: []trafficpolicy.TCPRouteMatch{
						{
							// route-3
							Ports: []uint16{3000},
						},
						{
							// route-4
							Ports: []uint16{4000},
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

			upstreamServiceIdentity: identity.K8sServiceAccount{Namespace: "ns-1", Name: "sa-1"}.ToServiceIdentity(),

			expectedTrafficTargets: []trafficpolicy.TrafficTargetWithRoutes{
				{
					Name:        "ns-1/test-1",
					Destination: identity.ServiceIdentity("sa-1.ns-1"),
					Sources: []identity.ServiceIdentity{
						identity.ServiceIdentity("sa-2.ns-2"),
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
			mockCompute := compute.NewMockInterface(mockCtrl)
			meshCatalog := MeshCatalog{
				Interface: mockCompute,
			}

			mockCompute.EXPECT().GetMeshConfig().AnyTimes()

			// Mock TrafficTargets returned by MeshSpec, should return all TrafficTargets relevant for this test
			mockCompute.EXPECT().ListTrafficTargets().Return(tc.trafficTargets).AnyTimes()
			for _, trafficTarget := range tc.trafficTargets {
				for _, rule := range trafficTarget.Spec.Rules {
					if rule.Kind != smi.TCPRouteKind {
						continue
					}
					routeName := fmt.Sprintf("%s/%s", trafficTarget.Spec.Destination.Namespace, rule.Name)
					mockCompute.EXPECT().GetTCPRoute(routeName).Return(tc.tcpRoutes[routeName]).AnyTimes()
				}
			}

			actual, err := meshCatalog.ListInboundTrafficTargetsWithRoutes(tc.upstreamServiceIdentity)
			assert.Equal(err != nil, tc.expectError)
			assert.ElementsMatch(tc.expectedTrafficTargets, actual)
		})
	}
}

func TestListTrafficSplitsByOptions(t *testing.T) {
	a := assert.New(t)

	mockCtrl := gomock.NewController(t)
	mockCompute := compute.NewMockInterface(mockCtrl)
	mockCompute.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()

	testNamespaceName := "test"
	fakeClient := fakePolicyClient.NewSimpleClientset()
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset()
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	ic, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient), informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet))
	a.Nil(err)
	meshCatalog := MeshCatalog{
		Interface: mockCompute,
	}

	obj := &smiSplit.TrafficSplit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ListTrafficSplits",
			Namespace: testNamespaceName,
		},
		Spec: smiSplit.TrafficSplitSpec{
			Service: tests.BookstoreApexServiceName,
			Backends: []smiSplit.TrafficSplitBackend{
				{
					Service: tests.BookstoreV1ServiceName,
					Weight:  tests.Weight90,
				},
				{
					Service: tests.BookstoreV2ServiceName,
					Weight:  tests.Weight10,
				},
			},
		},
	}
	err = ic.Add(informers.InformerKeyTrafficSplit, obj, t)
	a.Nil(err)
	mockCompute.EXPECT().ListTrafficSplits().Return([]*smiSplit.TrafficSplit{obj}).AnyTimes()

	// Verify
	actual := meshCatalog.ListTrafficSplits()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])

	// Verify filter for apex service
	filteredApexAvailable := meshCatalog.ListTrafficSplitsByOptions(smi.WithTrafficSplitApexService(service.MeshService{Name: tests.BookstoreApexServiceName, Namespace: testNamespaceName}))
	a.Len(filteredApexAvailable, 1)
	a.Equal(obj, filteredApexAvailable[0])
	filteredApexUnavailable := meshCatalog.ListTrafficSplitsByOptions(smi.WithTrafficSplitApexService(tests.BookstoreV1Service))
	a.Len(filteredApexUnavailable, 0)

	// Verify filter for backend service
	filteredBackendAvailable := meshCatalog.ListTrafficSplitsByOptions(smi.WithTrafficSplitBackendService(service.MeshService{Name: tests.BookstoreV1ServiceName, Namespace: testNamespaceName}))
	a.Len(filteredBackendAvailable, 1)
	a.Equal(obj, filteredBackendAvailable[0])
	filteredBackendNameMismatch := meshCatalog.ListTrafficSplitsByOptions(smi.WithTrafficSplitBackendService(service.MeshService{Namespace: testNamespaceName, Name: "invalid"}))
	a.Len(filteredBackendNameMismatch, 0)
	filteredBackendNamespaceMismatch := meshCatalog.ListTrafficSplitsByOptions(smi.WithTrafficSplitBackendService(service.MeshService{Namespace: "invalid", Name: tests.BookstoreV1ServiceName}))
	a.Len(filteredBackendNamespaceMismatch, 0)
}

func TestListTrafficTargetsByOptions(t *testing.T) {
	a := assert.New(t)

	mockCtrl := gomock.NewController(t)
	mockCompute := compute.NewMockInterface(mockCtrl)
	mockCompute.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()

	testNamespaceName := "test"
	fakeClient := fakePolicyClient.NewSimpleClientset()
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset()
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	ic, err := informers.NewInformerCollection("osm", nil, informers.WithPolicyClient(fakeClient), informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet))
	a.Nil(err)
	meshCatalog := MeshCatalog{
		Interface: mockCompute,
	}

	obj := &smiAccess.TrafficTarget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "access.smi-spec.io/v1alpha3",
			Kind:       "TrafficTarget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ListTrafficTargets",
			Namespace: testNamespaceName,
		},
		Spec: smiAccess.TrafficTargetSpec{
			Destination: smiAccess.IdentityBindingSubject{
				Kind:      "ServiceAccount",
				Name:      tests.BookstoreServiceAccountName,
				Namespace: testNamespaceName,
			},
			Sources: []smiAccess.IdentityBindingSubject{{
				Kind:      "ServiceAccount",
				Name:      tests.BookbuyerServiceAccountName,
				Namespace: testNamespaceName,
			}},
			Rules: []smiAccess.TrafficTargetRule{{
				Kind:    "HTTPRouteGroup",
				Name:    tests.RouteGroupName,
				Matches: []string{tests.BuyBooksMatchName},
			}},
		},
	}
	err = ic.Add(informers.InformerKeyTrafficTarget, obj, t)
	a.Nil(err)
	mockCompute.EXPECT().ListTrafficTargets().Return([]*smiAccess.TrafficTarget{obj}).AnyTimes()

	// Verify
	actual := meshCatalog.ListTrafficTargets()
	a.Len(actual, 1)
	a.Equal(obj, actual[0])

	// Verify destination based filtering
	filteredAvailable := meshCatalog.ListTrafficTargetsByOptions(smi.WithTrafficTargetDestination(identity.K8sServiceAccount{Namespace: testNamespaceName, Name: tests.BookstoreServiceAccountName}))
	a.Len(filteredAvailable, 1)
	filteredUnavailable := meshCatalog.ListTrafficTargetsByOptions(smi.WithTrafficTargetDestination(identity.K8sServiceAccount{Namespace: testNamespaceName, Name: "unavailable"}))
	a.Len(filteredUnavailable, 0)
}

func TestListSMIPolicies(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCompute := compute.NewMockInterface(mockCtrl)

	splits := []*smiSplit.TrafficSplit{&tests.TrafficSplit}
	targets := []*smiAccess.TrafficTarget{&tests.TrafficTarget}
	httpRoutes := []*smiSpecs.HTTPRouteGroup{&tests.HTTPRouteGroup}
	svcAccounts := []identity.K8sServiceAccount{tests.BookbuyerServiceAccount, tests.BookstoreServiceAccount}

	mockCompute.EXPECT().IsMonitoredNamespace(gomock.Any()).Return(true).AnyTimes()
	mockCompute.EXPECT().ListTrafficSplits().Return(splits).AnyTimes()
	mockCompute.EXPECT().ListTrafficTargets().Return(targets).AnyTimes()
	mockCompute.EXPECT().ListHTTPTrafficSpecs().Return(httpRoutes).AnyTimes()

	mc := &MeshCatalog{
		Interface: mockCompute,
	}

	a := assert.New(t)

	trafficSplits := mc.ListTrafficSplits()
	serviceAccounts := mc.ListServiceAccountsFromTrafficTargets()
	routeGroups := mc.ListHTTPTrafficSpecs()
	trafficTargets := mc.ListTrafficTargets()

	a.ElementsMatch(trafficSplits, splits)
	a.ElementsMatch(targets, trafficTargets)
	a.ElementsMatch(httpRoutes, routeGroups)
	a.ElementsMatch(svcAccounts, serviceAccounts)
}
