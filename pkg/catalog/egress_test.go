package catalog

import (
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	specs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/policy"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetEgressTrafficPolicy(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mockCfg := configurator.NewMockConfigurator(mockCtrl)

	defer mockCtrl.Finish()

	upstreamTrafficSetting := &policyv1alpha1.UpstreamTrafficSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u1",
			Namespace: "ns1",
		},
	}

	testCases := []struct {
		name                   string
		egressPolicies         []*policyv1alpha1.Egress
		egressPort             int
		httpRouteGroups        []*specs.HTTPRouteGroup
		upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting
		expectedEgressPolicy   *trafficpolicy.EgressTrafficPolicy
		expectError            bool
	}{
		{
			name: "multiple egress policies for HTTP ports",
			egressPolicies: []*policyv1alpha1.Egress{
				{
					Spec: policyv1alpha1.EgressSpec{
						Hosts: []string{
							"foo.com",
						},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   80,
								Protocol: "http",
							},
						},
					},
				},
				{
					Spec: policyv1alpha1.EgressSpec{
						Hosts: []string{
							"bar.com",
						},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   80,
								Protocol: "http",
							},
						},
					},
				},
				{
					Spec: policyv1alpha1.EgressSpec{
						Hosts: []string{
							"baz.com",
						},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   90,
								Protocol: "http",
							},
						},
					},
				},
			},
			httpRouteGroups: nil, // no SMI HTTP route matches
			expectedEgressPolicy: &trafficpolicy.EgressTrafficPolicy{
				TrafficMatches: []*trafficpolicy.TrafficMatch{
					{

						DestinationPort:     80, // Used by foo.com and bar.com
						DestinationProtocol: "http",
					},
					{
						DestinationPort:     90, // Used by baz.com
						DestinationProtocol: "http",
					},
				},
				HTTPRouteConfigsPerPort: map[int][]*trafficpolicy.EgressHTTPRouteConfig{
					80: {
						{
							Name: "foo.com",
							Hostnames: []string{
								"foo.com",
								"foo.com:80",
							},
							RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
								{
									Route: trafficpolicy.RouteWeightedClusters{
										HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
										WeightedClusters: mapset.NewSetFromSlice([]interface{}{
											service.WeightedCluster{ClusterName: service.ClusterName("foo.com:80"), Weight: 100},
										}),
									},
									AllowedDestinationIPRanges: nil,
								},
							},
						},
						{
							Name: "bar.com",
							Hostnames: []string{
								"bar.com",
								"bar.com:80",
							},
							RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
								{
									Route: trafficpolicy.RouteWeightedClusters{
										HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
										WeightedClusters: mapset.NewSetFromSlice([]interface{}{
											service.WeightedCluster{ClusterName: service.ClusterName("bar.com:80"), Weight: 100},
										}),
									},
									AllowedDestinationIPRanges: nil,
								},
							},
						},
					},
					90: {
						{
							Name: "baz.com",
							Hostnames: []string{
								"baz.com",
								"baz.com:90",
							},
							RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
								{
									Route: trafficpolicy.RouteWeightedClusters{
										HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
										WeightedClusters: mapset.NewSetFromSlice([]interface{}{
											service.WeightedCluster{ClusterName: service.ClusterName("baz.com:90"), Weight: 100},
										}),
									},
									AllowedDestinationIPRanges: nil,
								},
							},
						},
					},
				},
				ClustersConfigs: []*trafficpolicy.EgressClusterConfig{
					{
						Name: "foo.com:80",
						Host: "foo.com",
						Port: 80,
					},
					{
						Name: "bar.com:80",
						Host: "bar.com",
						Port: 80,
					},
					{
						Name: "baz.com:90",
						Host: "baz.com",
						Port: 90,
					},
				},
			},
			expectError: false,
		},
		{
			name: "multiple egress policies for HTTP and TCP ports",
			egressPolicies: []*policyv1alpha1.Egress{
				{
					Spec: policyv1alpha1.EgressSpec{
						Hosts: []string{
							"foo.com",
						},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   80,
								Protocol: "http",
							},
							{
								Number:   100,
								Protocol: "tcp", // This port should be ignored for HTTP routes
							},
						},
					},
				},
				{
					Spec: policyv1alpha1.EgressSpec{
						Hosts: []string{
							"bar.com",
						},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   80,
								Protocol: "http",
							},
						},
					},
				},
			},
			httpRouteGroups: nil, // no SMI HTTP route matches
			expectedEgressPolicy: &trafficpolicy.EgressTrafficPolicy{
				TrafficMatches: []*trafficpolicy.TrafficMatch{
					{
						DestinationPort:     80, // Used by foo.com and bar.com
						DestinationProtocol: "http",
					},
					{
						DestinationPort:     100, // Used by foo.com
						DestinationProtocol: "tcp",
						Cluster:             "100",
					},
				},
				HTTPRouteConfigsPerPort: map[int][]*trafficpolicy.EgressHTTPRouteConfig{
					80: {
						{
							Name: "foo.com",
							Hostnames: []string{
								"foo.com",
								"foo.com:80",
							},
							RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
								{
									Route: trafficpolicy.RouteWeightedClusters{
										HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
										WeightedClusters: mapset.NewSetFromSlice([]interface{}{
											service.WeightedCluster{ClusterName: service.ClusterName("foo.com:80"), Weight: 100},
										}),
									},
									AllowedDestinationIPRanges: nil,
								},
							},
						},
						{
							Name: "bar.com",
							Hostnames: []string{
								"bar.com",
								"bar.com:80",
							},
							RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
								{
									Route: trafficpolicy.RouteWeightedClusters{
										HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
										WeightedClusters: mapset.NewSetFromSlice([]interface{}{
											service.WeightedCluster{ClusterName: service.ClusterName("bar.com:80"), Weight: 100},
										}),
									},
									AllowedDestinationIPRanges: nil,
								},
							},
						},
					},
				},
				ClustersConfigs: []*trafficpolicy.EgressClusterConfig{
					{
						Name: "foo.com:80",
						Host: "foo.com",
						Port: 80,
					},
					{
						Name: "100",
						Port: 100,
					},
					{
						Name: "bar.com:80",
						Host: "bar.com",
						Port: 80,
					},
				},
			},
			expectError: false,
		},
		{
			name: "multiple egress policies for HTTPS and TCP ports",
			egressPolicies: []*policyv1alpha1.Egress{
				{
					Spec: policyv1alpha1.EgressSpec{
						Hosts: []string{
							"foo.com",
						},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   100,
								Protocol: "https",
							},
						},
					},
				},
				{
					Spec: policyv1alpha1.EgressSpec{
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   100,
								Protocol: "tcp",
							},
						},
					},
				},
			},
			httpRouteGroups: nil, // no SMI HTTP route matches
			expectedEgressPolicy: &trafficpolicy.EgressTrafficPolicy{
				TrafficMatches: []*trafficpolicy.TrafficMatch{
					{
						DestinationPort:     100,
						DestinationProtocol: "https",
						ServerNames:         []string{"foo.com"},
						Cluster:             "100",
					},
					{
						DestinationPort:     100,
						DestinationProtocol: "tcp",
						Cluster:             "100",
					},
				},
				HTTPRouteConfigsPerPort: map[int][]*trafficpolicy.EgressHTTPRouteConfig{},
				ClustersConfigs: []*trafficpolicy.EgressClusterConfig{
					{
						// Same cluster used for both HTTPS and TCP on port 100
						Name: "100",
						Port: 100,
					},
				},
			},
			expectError: false,
		},
		{
			name: "policy with valid UpstreamTrafficSetting match is processed",
			egressPolicies: []*policyv1alpha1.Egress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: upstreamTrafficSetting.Namespace,
					},
					Spec: policyv1alpha1.EgressSpec{
						Hosts: []string{
							"foo.com",
						},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   100,
								Protocol: "https",
							},
						},
						Matches: []corev1.TypedLocalObjectReference{
							{
								APIGroup: pointer.StringPtr("policy.openservicemesh.io/v1alpha1"),
								Kind:     "UpstreamTrafficSetting",
								Name:     upstreamTrafficSetting.Name,
							},
						},
					},
				},
			},
			httpRouteGroups:        nil, // no SMI HTTP route matches
			upstreamTrafficSetting: upstreamTrafficSetting,
			expectedEgressPolicy: &trafficpolicy.EgressTrafficPolicy{
				TrafficMatches: []*trafficpolicy.TrafficMatch{
					{
						DestinationPort:     100,
						DestinationProtocol: "https",
						ServerNames:         []string{"foo.com"},
						Cluster:             "100",
					},
				},
				HTTPRouteConfigsPerPort: map[int][]*trafficpolicy.EgressHTTPRouteConfig{},
				ClustersConfigs: []*trafficpolicy.EgressClusterConfig{
					{
						// Same cluster used for both HTTPS and TCP on port 100
						Name:                   "100",
						Port:                   100,
						UpstreamTrafficSetting: upstreamTrafficSetting,
					},
				},
			},
			expectError: false,
		},
		{
			name: "policy with invalid UpstreamTrafficSetting match is ignored",
			egressPolicies: []*policyv1alpha1.Egress{
				{
					Spec: policyv1alpha1.EgressSpec{
						Hosts: []string{
							"foo.com",
						},
						Ports: []policyv1alpha1.PortSpec{
							{
								Number:   100,
								Protocol: "https",
							},
						},
						Matches: []corev1.TypedLocalObjectReference{
							{
								APIGroup: pointer.StringPtr("policy.openservicemesh.io/v1alpha1"),
								Kind:     "UpstreamTrafficSetting",
								Name:     "invalid",
							},
						},
					},
				},
			},
			httpRouteGroups: nil, // no SMI HTTP route matches
			expectedEgressPolicy: &trafficpolicy.EgressTrafficPolicy{
				TrafficMatches:          nil,
				HTTPRouteConfigsPerPort: map[int][]*trafficpolicy.EgressHTTPRouteConfig{},
				ClustersConfigs:         nil,
			},
			expectError: false,
		},
	}

	testSourceIdentity := identity.ServiceIdentity("foo.bar.cluster.local")

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockPolicyController := policy.NewMockController(mockCtrl)

			for _, rg := range tc.httpRouteGroups {
				mockMeshSpec.EXPECT().GetHTTPRouteGroup(fmt.Sprintf("%s/%s", rg.Namespace, rg.Name)).Return(rg).AnyTimes()
			}
			mockPolicyController.EXPECT().ListEgressPoliciesForSourceIdentity(gomock.Any()).Return(tc.egressPolicies).Times(1)
			mockPolicyController.EXPECT().GetUpstreamTrafficSetting(gomock.Any()).Return(tc.upstreamTrafficSetting).AnyTimes()

			mc := &MeshCatalog{
				meshSpec:         mockMeshSpec,
				configurator:     mockCfg,
				policyController: mockPolicyController,
			}

			mockCfg.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableEgressPolicy: true}).Times(1)

			actual, err := mc.GetEgressTrafficPolicy(testSourceIdentity)
			assert.Equal(tc.expectError, err != nil)
			assert.ElementsMatch(tc.expectedEgressPolicy.TrafficMatches, actual.TrafficMatches)
			assert.ElementsMatch(tc.expectedEgressPolicy.ClustersConfigs, actual.ClustersConfigs)
			assert.Equal(tc.expectedEgressPolicy.HTTPRouteConfigsPerPort, actual.HTTPRouteConfigsPerPort)
		})
	}
}

func TestBuildHTTPRouteConfigs(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	upstreamTrafficSetting := &policyv1alpha1.UpstreamTrafficSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u1",
			Namespace: "ns1",
		},
	}

	testCases := []struct {
		name                   string
		egressPolicy           *policyv1alpha1.Egress
		egressPort             int
		httpRouteGroups        []*specs.HTTPRouteGroup
		upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting
		expectedRouteConfigs   []*trafficpolicy.EgressHTTPRouteConfig
		expectedClusterConfigs []*trafficpolicy.EgressClusterConfig
	}{
		{
			name: "egress policy with no SMI HTTP route matches specified",
			egressPolicy: &policyv1alpha1.Egress{
				Spec: policyv1alpha1.EgressSpec{
					Hosts: []string{
						"foo.com",
						"bar.com",
					},
					Ports: []policyv1alpha1.PortSpec{
						{
							Number:   80,
							Protocol: "http",
						},
					},
				},
			},
			egressPort:      80,
			httpRouteGroups: nil, // no matches specified in the egress policy via Spec.Matches
			expectedRouteConfigs: []*trafficpolicy.EgressHTTPRouteConfig{
				{
					Name: "foo.com",
					Hostnames: []string{
						"foo.com",
						"foo.com:80",
					},
					RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("foo.com:80"), Weight: 100},
								}),
							},
							AllowedDestinationIPRanges: nil,
						},
					},
				},
				{
					Name: "bar.com",
					Hostnames: []string{
						"bar.com",
						"bar.com:80",
					},
					RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("bar.com:80"), Weight: 100},
								}),
							},
							AllowedDestinationIPRanges: nil,
						},
					},
				},
			},
			expectedClusterConfigs: []*trafficpolicy.EgressClusterConfig{
				{
					Name: "foo.com:80",
					Host: "foo.com",
					Port: 80,
				},
				{
					Name: "bar.com:80",
					Host: "bar.com",
					Port: 80,
				},
			},
		},
		{
			name: "egress policy with SMI matching routes specified",
			egressPolicy: &policyv1alpha1.Egress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "egress-1",
					Namespace: "test",
				},
				Spec: policyv1alpha1.EgressSpec{
					Hosts: []string{
						"foo.com",
					},
					Ports: []policyv1alpha1.PortSpec{
						{
							Number:   80,
							Protocol: "http",
						},
					},
					Matches: []corev1.TypedLocalObjectReference{
						{
							APIGroup: pointer.StringPtr("specs.smi-spec.io/v1alpha4"),
							Kind:     "HTTPRouteGroup",
							Name:     "route-1",
						},
						{
							APIGroup: pointer.StringPtr("specs.smi-spec.io/v1alpha4"),
							Kind:     "HTTPRouteGroup",
							Name:     "route-2",
						},
					},
				},
			},
			egressPort: 80,
			httpRouteGroups: []*specs.HTTPRouteGroup{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "specs.smi-spec.io/v1alpha4",
						Kind:       "HTTPRouteGroup",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-1",
						Namespace: "test",
					},
					Spec: spec.HTTPRouteGroupSpec{
						Matches: []specs.HTTPMatch{
							{
								Name:      "match-1",
								PathRegex: "/foo",
								Methods:   []string{"GET"},
							},
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "specs.smi-spec.io/v1alpha4",
						Kind:       "HTTPRouteGroup",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-2",
						Namespace: "test",
					},
					Spec: spec.HTTPRouteGroupSpec{
						Matches: []specs.HTTPMatch{
							{
								Name:      "match-2",
								PathRegex: "/bar",
								Methods:   []string{"GET"},
							},
						},
					},
				},
			},
			expectedRouteConfigs: []*trafficpolicy.EgressHTTPRouteConfig{
				{
					Name: "foo.com",
					Hostnames: []string{
						"foo.com",
						"foo.com:80",
					},
					RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/foo",
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{"GET"},
								},
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("foo.com:80"), Weight: 100},
								}),
							},
							AllowedDestinationIPRanges: nil,
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/bar",
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{"GET"},
								},
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("foo.com:80"), Weight: 100},
								}),
							},
							AllowedDestinationIPRanges: nil,
						},
					},
				},
			},
			expectedClusterConfigs: []*trafficpolicy.EgressClusterConfig{
				{
					Name: "foo.com:80",
					Host: "foo.com",
					Port: 80,
				},
			},
		},
		{
			name: "egress policy with SMI matching routes and IP addresses specified",
			egressPolicy: &policyv1alpha1.Egress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "egress-1",
					Namespace: "test",
				},
				Spec: policyv1alpha1.EgressSpec{
					Hosts: []string{
						"foo.com",
					},
					Ports: []policyv1alpha1.PortSpec{
						{
							Number:   80,
							Protocol: "http",
						},
					},
					IPAddresses: []string{
						"1.1.1.1/32",
						"10.0.0.0/24",
					},
					Matches: []corev1.TypedLocalObjectReference{
						{
							APIGroup: pointer.StringPtr("specs.smi-spec.io/v1alpha4"),
							Kind:     "HTTPRouteGroup",
							Name:     "route-1",
						},
					},
				},
			},
			egressPort: 80,
			httpRouteGroups: []*specs.HTTPRouteGroup{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "specs.smi-spec.io/v1alpha4",
						Kind:       "HTTPRouteGroup",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "route-1",
						Namespace: "test",
					},
					Spec: spec.HTTPRouteGroupSpec{
						Matches: []specs.HTTPMatch{
							{
								Name:      "match-1",
								PathRegex: "/foo",
								Methods:   []string{"GET"},
							},
						},
					},
				},
			},
			expectedRouteConfigs: []*trafficpolicy.EgressHTTPRouteConfig{
				{
					Name: "foo.com",
					Hostnames: []string{
						"foo.com",
						"foo.com:80",
					},
					RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          "/foo",
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{"GET"},
								},
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("foo.com:80"), Weight: 100},
								}),
							},
							AllowedDestinationIPRanges: []string{"1.1.1.1/32", "10.0.0.0/24"},
						},
					},
				},
			},
			expectedClusterConfigs: []*trafficpolicy.EgressClusterConfig{
				{
					Name: "foo.com:80",
					Host: "foo.com",
					Port: 80,
				},
			},
		},
		{
			name: "egress policy with UpstreamTrafficSetting match specified",
			egressPolicy: &policyv1alpha1.Egress{
				Spec: policyv1alpha1.EgressSpec{
					Hosts: []string{
						"foo.com",
						"bar.com",
					},
					Ports: []policyv1alpha1.PortSpec{
						{
							Number:   80,
							Protocol: "http",
						},
					},
				},
			},
			egressPort:             80,
			httpRouteGroups:        nil, // no matches specified in the egress policy via Spec.Matches
			upstreamTrafficSetting: upstreamTrafficSetting,
			expectedRouteConfigs: []*trafficpolicy.EgressHTTPRouteConfig{
				{
					Name: "foo.com",
					Hostnames: []string{
						"foo.com",
						"foo.com:80",
					},
					RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("foo.com:80"), Weight: 100},
								}),
							},
							AllowedDestinationIPRanges: nil,
						},
					},
				},
				{
					Name: "bar.com",
					Hostnames: []string{
						"bar.com",
						"bar.com:80",
					},
					RoutingRules: []*trafficpolicy.EgressHTTPRoutingRule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.WildCardRouteMatch,
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: service.ClusterName("bar.com:80"), Weight: 100},
								}),
							},
							AllowedDestinationIPRanges: nil,
						},
					},
				},
			},
			expectedClusterConfigs: []*trafficpolicy.EgressClusterConfig{
				{
					Name:                   "foo.com:80",
					Host:                   "foo.com",
					Port:                   80,
					UpstreamTrafficSetting: upstreamTrafficSetting,
				},
				{
					Name:                   "bar.com:80",
					Host:                   "bar.com",
					Port:                   80,
					UpstreamTrafficSetting: upstreamTrafficSetting,
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)

			for _, rg := range tc.httpRouteGroups {
				mockMeshSpec.EXPECT().GetHTTPRouteGroup(fmt.Sprintf("%s/%s", rg.Namespace, rg.Name)).Return(rg).AnyTimes()
			}

			mc := &MeshCatalog{
				meshSpec: mockMeshSpec,
			}

			routeConfigs, clusterConfigs := mc.buildHTTPRouteConfigs(tc.egressPolicy, tc.egressPort, tc.upstreamTrafficSetting)
			assert.ElementsMatch(tc.expectedRouteConfigs, routeConfigs)
			assert.ElementsMatch(tc.expectedClusterConfigs, clusterConfigs)
		})
	}
}

func TestGetHTTPRouteMatchesFromHTTPRouteGroup(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name            string
		httpRouteGroup  *specs.HTTPRouteGroup
		expectedMatches []trafficpolicy.HTTPRouteMatch
	}{
		{
			name: "multiple HTTP route matches",
			httpRouteGroup: &specs.HTTPRouteGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "specs.smi-spec.io/v1alpha4",
					Kind:       "HTTPRouteGroup",
				},
				Spec: spec.HTTPRouteGroupSpec{
					Matches: []specs.HTTPMatch{
						{
							Name:      "match-1",
							PathRegex: "/foo",
							Methods:   []string{"GET"},
						},
						{
							Name:      "match-2",
							PathRegex: "/bar",
							Methods:   []string{"GET"},
						},
					},
				},
			},
			expectedMatches: []trafficpolicy.HTTPRouteMatch{
				{
					Path:          "/foo",
					PathMatchType: trafficpolicy.PathMatchRegex,
					Methods:       []string{"GET"},
				},
				{
					Path:          "/bar",
					PathMatchType: trafficpolicy.PathMatchRegex,
					Methods:       []string{"GET"},
				},
			},
		},
		{
			name:            "nil HTTPRouteGroup",
			httpRouteGroup:  nil,
			expectedMatches: nil,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			actual := getHTTPRouteMatchesFromHTTPRouteGroup(tc.httpRouteGroup)

			assert.ElementsMatch(tc.expectedMatches, actual)
		})
	}
}
