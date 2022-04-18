package catalog

import (
	"net"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/policy"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestGetOutboundMeshTrafficPolicy(t *testing.T) {
	//---
	// Create MeshServices used by the test. We test both explicit protocol specification and defaults here.
	// MeshService for k8s service ns1/s1 with 2 ports
	meshSvc1P1 := service.MeshService{Name: "s1", Namespace: "ns1", Port: 8080, TargetPort: 80, Protocol: "http"}
	meshSvc1P2 := service.MeshService{Name: "s1", Namespace: "ns1", Port: 9090, TargetPort: 90, Protocol: "http"}
	// MeshService for  k8s service ns2/s2 with 1 port
	meshSvc2 := service.MeshService{Name: "s2", Namespace: "ns2", Port: 8080, TargetPort: 80, Protocol: "http"}
	// MeshService for  k8s service ns3/s3 with 1 port, and 2 split backends
	meshSvc3 := service.MeshService{Name: "s3", Namespace: "ns3", Port: 8080, TargetPort: 80, Protocol: "http"}
	meshSvc3V1 := service.MeshService{Name: "s3-v1", Namespace: "ns3", Port: 8080, TargetPort: 80, Protocol: "http"}
	meshSvc3V2 := service.MeshService{Name: "s3-v2", Namespace: "ns3", Port: 8080, TargetPort: 80, Protocol: "http"}
	// MeshService for  k8s service ns3/s4 with 1 port
	meshSvc4 := service.MeshService{Name: "s4", Namespace: "ns3", Port: 9090, TargetPort: 90, Protocol: "tcp"}
	// MeshService for k8s service ns3/s5 with 1 port
	meshSvc5 := service.MeshService{Name: "s5", Namespace: "ns3", Port: 9091, TargetPort: 91, Protocol: "tcp-server-first"}

	allMeshServices := []service.MeshService{meshSvc1P1, meshSvc1P2, meshSvc2, meshSvc3, meshSvc3V1, meshSvc3V2, meshSvc4, meshSvc5}

	svcToEndpointsMap := map[string][]endpoint.Endpoint{
		meshSvc1P1.String(): {
			{IP: net.ParseIP("10.0.1.1")},
			{IP: net.ParseIP("10.0.1.2")},
		},
		meshSvc1P2.String(): {
			{IP: net.ParseIP("10.0.1.1")},
			{IP: net.ParseIP("10.0.1.2")},
		},
		meshSvc2.String(): {
			{IP: net.ParseIP("10.0.2.1")},
		},
		meshSvc3.String(): {
			{IP: net.ParseIP("10.0.3.1")},
		},
		meshSvc3V1.String(): {
			{IP: net.ParseIP("10.0.3.2")},
		},
		meshSvc3V2.String(): {
			{IP: net.ParseIP("10.0.3.3")},
		},
		meshSvc4.String(): {
			{IP: net.ParseIP("10.0.4.1")},
		},
		meshSvc5.String(): {
			{IP: net.ParseIP("10.0.5.1")},
		},
	}

	svcIdentityToSvcMapping := map[string][]service.MeshService{
		"sa1.ns1.cluster.local": {meshSvc1P1, meshSvc1P2},
		"sa2.ns2.cluster.local": {meshSvc2}, // Client `downstreamIdentity` cannot access this upstream
		"sa3.ns3.cluster.local": {meshSvc3, meshSvc3V1, meshSvc3V2, meshSvc4, meshSvc5},
	}

	downstreamIdentity := identity.ServiceIdentity("sa-x.ns1.cluster.local")

	// TrafficTargets that allow: sa-x.ns1 -> sa1.ns1, sa3.ns3
	// No TrafficTarget that allows sa-x.ns1 -> sa2.ns2 (this should be allowed in permissive mode)
	trafficTargets := []*access.TrafficTarget{
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "access.smi-spec.io/v1alpha3",
				Kind:       "TrafficTarget",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "t1",
				Namespace: "ns1",
			},
			Spec: access.TrafficTargetSpec{
				Destination: access.IdentityBindingSubject{
					Kind:      "ServiceAccount",
					Name:      "sa1",
					Namespace: "ns1",
				},
				Sources: []access.IdentityBindingSubject{{
					Kind:      "ServiceAccount",
					Name:      "sa-x", // matches downstreamIdentity
					Namespace: "ns1",
				}},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "access.smi-spec.io/v1alpha3",
				Kind:       "TrafficTarget",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "t3",
				Namespace: "ns3",
			},
			Spec: access.TrafficTargetSpec{
				Destination: access.IdentityBindingSubject{
					Kind:      "ServiceAccount",
					Name:      "sa3",
					Namespace: "ns3",
				},
				Sources: []access.IdentityBindingSubject{{
					Kind:      "ServiceAccount",
					Name:      "sa-x", // matches downstreamIdentity
					Namespace: "ns1",
				}},
			},
		},
	}

	// TrafficSplit
	// In this test, we create a TrafficSplit for service ns3/s3 to split
	// traffic to ns3/s3-v1 and ns3/s3-v2
	trafficSplitSvc3 := &split.TrafficSplit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "split3",
			Namespace: "ns3",
		},
		Spec: split.TrafficSplitSpec{
			Service: "s3.ns3.cluster.local",
			Backends: []split.TrafficSplitBackend{
				{
					Service: "s3-v1",
					Weight:  10,
				},
				{
					Service: "s3-v2",
					Weight:  90,
				},
			},
		},
	}
	trafficSplits := []*split.TrafficSplit{trafficSplitSvc3}

	// Add UpstreamTrafficSetting config for service meshSvc1P1, meshSvc1P1: ns1/s1
	// Both map to the same k8s service but different ports
	upstreamTrafficSettingSvc1 := policyv1alpha1.UpstreamTrafficSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "meshSvc1P1",
			Namespace: meshSvc1P1.Namespace,
		},
		Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
			Host: meshSvc1P1.FQDN(),
		},
	}

	testCases := []struct {
		name           string
		permissiveMode bool
		expected       *trafficpolicy.OutboundMeshTrafficPolicy
	}{
		{
			name:           "SMI mode with traffic target(deny ns2/sa2/s2) and split(ns3/s3)",
			permissiveMode: false,
			expected: &trafficpolicy.OutboundMeshTrafficPolicy{
				TrafficMatches: []*trafficpolicy.TrafficMatch{
					{
						// To match ns1/s1 on port 8080
						Name:                meshSvc1P1.OutboundTrafficMatchName(),
						DestinationPort:     8080,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.1.1/32", "10.0.1.2/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns1/s1|80",
								Weight:      100,
							},
						},
					},
					{
						// To match ns1/s1 on port 9090
						Name:                meshSvc1P2.OutboundTrafficMatchName(),
						DestinationPort:     9090,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.1.1/32", "10.0.1.2/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns1/s1|90",
								Weight:      100,
							},
						},
					},
					{
						// To match ns3/s3(s3 apex) on port 8080, split to s3-v1 and s3-v2
						Name:                meshSvc3.OutboundTrafficMatchName(),
						DestinationPort:     8080,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.3.1/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns3/s3-v1|80",
								Weight:      10,
							},
							{
								ClusterName: "ns3/s3-v2|80",
								Weight:      90,
							},
						},
					},
					{
						// To match ns3/s3(s3-v1) on port 8080
						Name:                meshSvc3V1.OutboundTrafficMatchName(),
						DestinationPort:     8080,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.3.2/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns3/s3-v1|80",
								Weight:      100,
							},
						},
					},
					{
						// To match ns3/s3(s3-v2) on port 8080
						Name:                meshSvc3V2.OutboundTrafficMatchName(),
						DestinationPort:     8080,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.3.3/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns3/s3-v2|80",
								Weight:      100,
							},
						},
					},
					{
						// To match ns3/s4 on port 9090
						Name:                meshSvc4.OutboundTrafficMatchName(),
						DestinationPort:     9090,
						DestinationProtocol: "tcp",
						DestinationIPRanges: []string{"10.0.4.1/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns3/s4|90",
								Weight:      100,
							},
						},
					},
					{
						// To match ns3/s5 on port 9091
						Name:                meshSvc5.OutboundTrafficMatchName(),
						DestinationPort:     9091,
						DestinationProtocol: "tcp-server-first",
						DestinationIPRanges: []string{"10.0.5.1/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns3/s5|91",
								Weight:      100,
							},
						},
					},
				},
				ClustersConfigs: []*trafficpolicy.MeshClusterConfig{
					{
						Name:                   "ns1/s1|80",
						Service:                meshSvc1P1,
						UpstreamTrafficSetting: &upstreamTrafficSettingSvc1,
					},
					{
						Name:                   "ns1/s1|90",
						Service:                meshSvc1P2,
						UpstreamTrafficSetting: &upstreamTrafficSettingSvc1,
					},
					{
						Name:    "ns3/s3|80",
						Service: meshSvc3,
					},
					{
						Name:    "ns3/s3-v1|80",
						Service: meshSvc3V1,
					},
					{
						Name:    "ns3/s3-v2|80",
						Service: meshSvc3V2,
					},
					{
						Name:    "ns3/s4|90",
						Service: meshSvc4,
					},
					{
						Name:    "ns3/s5|91",
						Service: meshSvc5,
					},
				},
				HTTPRouteConfigsPerPort: map[int][]*trafficpolicy.OutboundTrafficPolicy{
					8080: {
						{
							Name: "s1.ns1.svc.cluster.local",
							Hostnames: []string{
								"s1",
								"s1:8080",
								"s1.ns1",
								"s1.ns1:8080",
								"s1.ns1.svc",
								"s1.ns1.svc:8080",
								"s1.ns1.svc.cluster",
								"s1.ns1.svc.cluster:8080",
								"s1.ns1.svc.cluster.local",
								"s1.ns1.svc.cluster.local:8080",
							},
							Routes: []*trafficpolicy.RouteWeightedClusters{
								{
									HTTPRouteMatch: tests.WildCardRouteMatch,
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "ns1/s1|80",
										Weight:      100,
									}),
								},
							},
						},
						{
							Name: "s3.ns3.svc.cluster.local",
							Hostnames: []string{
								"s3.ns3",
								"s3.ns3:8080",
								"s3.ns3.svc",
								"s3.ns3.svc:8080",
								"s3.ns3.svc.cluster",
								"s3.ns3.svc.cluster:8080",
								"s3.ns3.svc.cluster.local",
								"s3.ns3.svc.cluster.local:8080",
							},
							Routes: []*trafficpolicy.RouteWeightedClusters{
								{
									HTTPRouteMatch: tests.WildCardRouteMatch,
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "ns3/s3-v1|80",
										Weight:      10,
									}, service.WeightedCluster{
										ClusterName: "ns3/s3-v2|80",
										Weight:      90,
									}),
								},
							},
						},
						{
							Name: "s3-v1.ns3.svc.cluster.local",
							Hostnames: []string{
								"s3-v1.ns3",
								"s3-v1.ns3:8080",
								"s3-v1.ns3.svc",
								"s3-v1.ns3.svc:8080",
								"s3-v1.ns3.svc.cluster",
								"s3-v1.ns3.svc.cluster:8080",
								"s3-v1.ns3.svc.cluster.local",
								"s3-v1.ns3.svc.cluster.local:8080",
							},
							Routes: []*trafficpolicy.RouteWeightedClusters{
								{
									HTTPRouteMatch: tests.WildCardRouteMatch,
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "ns3/s3-v1|80",
										Weight:      100,
									}),
								},
							},
						},
						{
							Name: "s3-v2.ns3.svc.cluster.local",
							Hostnames: []string{
								"s3-v2.ns3",
								"s3-v2.ns3:8080",
								"s3-v2.ns3.svc",
								"s3-v2.ns3.svc:8080",
								"s3-v2.ns3.svc.cluster",
								"s3-v2.ns3.svc.cluster:8080",
								"s3-v2.ns3.svc.cluster.local",
								"s3-v2.ns3.svc.cluster.local:8080",
							},
							Routes: []*trafficpolicy.RouteWeightedClusters{
								{
									HTTPRouteMatch: tests.WildCardRouteMatch,
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "ns3/s3-v2|80",
										Weight:      100,
									}),
								},
							},
						},
					},
					9090: {
						{
							Name: "s1.ns1.svc.cluster.local",
							Hostnames: []string{
								"s1",
								"s1:9090",
								"s1.ns1",
								"s1.ns1:9090",
								"s1.ns1.svc",
								"s1.ns1.svc:9090",
								"s1.ns1.svc.cluster",
								"s1.ns1.svc.cluster:9090",
								"s1.ns1.svc.cluster.local",
								"s1.ns1.svc.cluster.local:9090",
							},
							Routes: []*trafficpolicy.RouteWeightedClusters{
								{
									HTTPRouteMatch: tests.WildCardRouteMatch,
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "ns1/s1|90",
										Weight:      100,
									}),
								},
							},
						},
					},
				},
			},
		},
		{
			name:           "Permissive mode with traffic split(ns3/s3)",
			permissiveMode: true,
			expected: &trafficpolicy.OutboundMeshTrafficPolicy{
				TrafficMatches: []*trafficpolicy.TrafficMatch{
					{
						// To match ns1/s1 on port 8080
						Name:                meshSvc1P1.OutboundTrafficMatchName(),
						DestinationPort:     8080,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.1.1/32", "10.0.1.2/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns1/s1|80",
								Weight:      100,
							},
						},
					},
					{
						// To match ns1/s1 on port 9090
						Name:                meshSvc1P2.OutboundTrafficMatchName(),
						DestinationPort:     9090,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.1.1/32", "10.0.1.2/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns1/s1|90",
								Weight:      100,
							},
						},
					},
					{
						// To match ns2/s2 on port 8080
						Name:                meshSvc2.OutboundTrafficMatchName(),
						DestinationPort:     8080,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.2.1/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns2/s2|80",
								Weight:      100,
							},
						},
					},
					{
						// To match ns3/s3(s3 apex) on port 8080, split to s3-v1 and s3-v2
						Name:                meshSvc3.OutboundTrafficMatchName(),
						DestinationPort:     8080,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.3.1/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns3/s3-v1|80",
								Weight:      10,
							},
							{
								ClusterName: "ns3/s3-v2|80",
								Weight:      90,
							},
						},
					},
					{
						// To match ns3/s3(s3-v1) on port 8080
						Name:                meshSvc3V1.OutboundTrafficMatchName(),
						DestinationPort:     8080,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.3.2/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns3/s3-v1|80",
								Weight:      100,
							},
						},
					},
					{
						// To match ns3/s3(s3-v2) on port 8080
						Name:                meshSvc3V2.OutboundTrafficMatchName(),
						DestinationPort:     8080,
						DestinationProtocol: "http",
						DestinationIPRanges: []string{"10.0.3.3/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns3/s3-v2|80",
								Weight:      100,
							},
						},
					},
					{
						// To match ns3/s4 on port 9090
						Name:                meshSvc4.OutboundTrafficMatchName(),
						DestinationPort:     9090,
						DestinationProtocol: "tcp",
						DestinationIPRanges: []string{"10.0.4.1/32"},
						WeightedClusters: []service.WeightedCluster{
							{
								ClusterName: "ns3/s4|90",
								Weight:      100,
							},
						},
					},
				},
				ClustersConfigs: []*trafficpolicy.MeshClusterConfig{
					{
						Name:                   "ns1/s1|80",
						Service:                meshSvc1P1,
						UpstreamTrafficSetting: &upstreamTrafficSettingSvc1,
					},
					{
						Name:                   "ns1/s1|90",
						Service:                meshSvc1P2,
						UpstreamTrafficSetting: &upstreamTrafficSettingSvc1,
					},
					{
						Name:    "ns2/s2|80",
						Service: meshSvc2,
					},
					{
						Name:    "ns3/s3|80",
						Service: meshSvc3,
					},
					{
						Name:    "ns3/s3-v1|80",
						Service: meshSvc3V1,
					},
					{
						Name:    "ns3/s3-v2|80",
						Service: meshSvc3V2,
					},
					{
						Name:    "ns3/s4|90",
						Service: meshSvc4,
					},
					{
						Name:    "ns3/s5|91",
						Service: meshSvc5,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
			mockServiceProvider := service.NewMockProvider(mockCtrl)
			mockCfg := configurator.NewMockConfigurator(mockCtrl)
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockPolicyController := policy.NewMockController(mockCtrl)

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
				serviceProviders:   []service.Provider{mockServiceProvider},
				configurator:       mockCfg,
				meshSpec:           mockMeshSpec,
				policyController:   mockPolicyController,
			}

			// Mock calls to k8s client caches
			mockCfg.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).AnyTimes()
			mockCfg.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{}).AnyTimes()
			mockServiceProvider.EXPECT().ListServices().Return(allMeshServices).AnyTimes()
			mockMeshSpec.EXPECT().ListTrafficTargets().Return(trafficTargets).AnyTimes()
			mockServiceProvider.EXPECT().GetID().Return("test").AnyTimes()
			mockEndpointProvider.EXPECT().GetID().Return("test").AnyTimes()
			firstSplitCall := mockMeshSpec.EXPECT().ListTrafficSplits().Return(trafficSplits).Times(1)
			// Mock conditional traffic split for service
			mockMeshSpec.EXPECT().ListTrafficSplits(gomock.Any()).DoAndReturn(
				func(options ...smi.TrafficSplitListOption) []*split.TrafficSplit {
					o := &smi.TrafficSplitListOpt{}
					for _, opt := range options {
						opt(o)
					}
					// In this test, only service ns3/s3 has a split configured
					if o.ApexService.String() == "ns3/s3" {
						return []*split.TrafficSplit{trafficSplitSvc3}
					}
					return nil
				}).After(firstSplitCall).AnyTimes()

			// Mock ServiceIdentity -> Service lookups executed when TrafficTargets are evaluated
			if !tc.permissiveMode {
				for _, target := range trafficTargets {
					dstSvcIdentity := identity.K8sServiceAccount{Namespace: target.Spec.Destination.Namespace, Name: target.Spec.Destination.Name}.ToServiceIdentity()
					mockServiceProvider.EXPECT().GetServicesForServiceIdentity(dstSvcIdentity).Return(svcIdentityToSvcMapping[dstSvcIdentity.String()]).AnyTimes()
				}
			} else {
				for svcIdentity, services := range svcIdentityToSvcMapping {
					mockServiceProvider.EXPECT().GetServicesForServiceIdentity(svcIdentity).Return(services).AnyTimes()
				}
			}

			// Mock service -> endpoint lookups
			mockEndpointProvider.EXPECT().GetResolvableEndpointsForService(gomock.Any()).DoAndReturn(
				func(svc service.MeshService) ([]endpoint.Endpoint, error) {
					return svcToEndpointsMap[svc.String()], nil
				}).AnyTimes()

			// Mock calls to UpstreamTrafficSetting lookups
			mockPolicyController.EXPECT().GetUpstreamTrafficSetting(gomock.Any()).DoAndReturn(
				func(opt policy.UpstreamTrafficSettingGetOpt) *policyv1alpha1.UpstreamTrafficSetting {
					// In this test, only service ns1/<p1|p2> has UpstreamTrafficSetting configured
					if opt.MeshService != nil &&
						(*opt.MeshService == meshSvc1P1 || *opt.MeshService == meshSvc1P2) {
						return &upstreamTrafficSettingSvc1
					}
					return nil
				}).AnyTimes()

			actual := mc.GetOutboundMeshTrafficPolicy(downstreamIdentity)
			assert.NotNil(actual)

			// Verify expected fields
			assert.ElementsMatch(tc.expected.TrafficMatches, actual.TrafficMatches)
			assert.ElementsMatch(tc.expected.ClustersConfigs, actual.ClustersConfigs)
			for expectedKey, expectedVal := range tc.expected.HTTPRouteConfigsPerPort {
				assert.ElementsMatch(expectedVal, actual.HTTPRouteConfigsPerPort[expectedKey])
			}
		})
		break
	}
}

func TestListOutboundServicesForIdentity(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name           string
		svcIdentity    identity.ServiceIdentity
		expectedList   []service.MeshService
		permissiveMode bool
	}{
		{
			name:           "traffic targets configured for service account",
			svcIdentity:    tests.BookbuyerServiceIdentity,
			expectedList:   []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService},
			permissiveMode: false,
		},
		{
			name: "traffic targets not configured for service account",
			svcIdentity: identity.K8sServiceAccount{
				Name:      "some-name",
				Namespace: "some-ns",
			}.ToServiceIdentity(),
			expectedList:   nil,
			permissiveMode: false,
		},
		{
			name:           "permissive mode enabled",
			svcIdentity:    tests.BookstoreServiceIdentity,
			expectedList:   []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService, tests.BookbuyerService},
			permissiveMode: true,
		},
		{
			name:           "gateway",
			svcIdentity:    "gateway.osm-system.cluster.local",
			expectedList:   []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService, tests.BookbuyerService},
			permissiveMode: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mc := newFakeMeshCatalogForRoutes(t, testParams{
				permissiveMode: tc.permissiveMode,
			})
			actualList := mc.ListOutboundServicesForIdentity(tc.svcIdentity)
			assert.ElementsMatch(actualList, tc.expectedList)
		})
	}
}

func TestGetDestinationServicesFromTrafficTarget(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
	mockServiceProvider := service.NewMockProvider(mockCtrl)

	mc := MeshCatalog{
		kubeController:     mockKubeController,
		endpointsProviders: []endpoint.Provider{mockEndpointProvider},
		serviceProviders:   []service.Provider{mockServiceProvider},
	}

	destSA := identity.K8sServiceAccount{
		Name:      "bookstore",
		Namespace: "bookstore-ns",
	}

	destMeshService := service.MeshService{
		Name:      "bookstore",
		Namespace: "bookstore-ns",
	}

	destK8sService := tests.NewServiceFixture(destMeshService.Name, destMeshService.Namespace, map[string]string{})
	mockServiceProvider.EXPECT().GetServicesForServiceIdentity(destSA.ToServiceIdentity()).Return([]service.MeshService{destMeshService}).AnyTimes()
	mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
	mockServiceProvider.EXPECT().GetID().Return("fake").AnyTimes()
	mockKubeController.EXPECT().GetService(destMeshService).Return(destK8sService).AnyTimes()

	trafficTarget := &access.TrafficTarget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "access.smi-spec.io/v1alpha3",
			Kind:       "TrafficTarget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "target",
			Namespace: "bookstore-ns",
		},
		Spec: access.TrafficTargetSpec{
			Destination: access.IdentityBindingSubject{
				Kind:      "Name",
				Name:      "bookstore",
				Namespace: "bookstore-ns",
			},
			Sources: []access.IdentityBindingSubject{{
				Kind:      "Name",
				Name:      "bookbuyer",
				Namespace: "default",
			}},
		},
	}

	actual := mc.getDestinationServicesFromTrafficTarget(trafficTarget)
	assert.Equal([]service.MeshService{destMeshService}, actual)
}

func TestListAllowedUpstreamServicesIncludeApex(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockController := k8s.NewMockController(mockCtrl)
	mockServiceProvider := service.NewMockProvider(mockCtrl)
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()
	mockConfigurator.EXPECT().GetOSMNamespace().Return("osm-system").AnyTimes()

	mc := MeshCatalog{
		meshSpec:         mockMeshSpec,
		kubeController:   mockController,
		configurator:     mockConfigurator,
		serviceProviders: []service.Provider{mockServiceProvider},
	}

	testCases := []struct {
		name          string
		id            identity.ServiceIdentity
		services      []*corev1.Service
		trafficSplits []*split.TrafficSplit
		expected      []service.MeshService
	}{
		{
			name:     "no allowed outbound services",
			id:       "foo.bar",
			expected: nil,
		},
		{
			name: "some allowed service",
			id:   "my-src-ns.my-src-name",
			services: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-dst-ns",
						Name:      "split-backend-1",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "servicePort",
								Protocol:   corev1.ProtocolTCP,
								Port:       tests.ServicePort,
								TargetPort: intstr.FromInt(tests.ServicePort),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-dst-ns",
						Name:      "split-backend-2",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "servicePort",
								Protocol:   corev1.ProtocolTCP,
								Port:       tests.ServicePort,
								TargetPort: intstr.FromInt(tests.ServicePort),
							},
						},
					},
				},
			},
			trafficSplits: []*split.TrafficSplit{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "wrong-ns",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "split",
						Namespace: "my-dst-ns",
					},
					Spec: split.TrafficSplitSpec{
						Service: "split-svc.my-dst-ns",
						Backends: []split.TrafficSplitBackend{
							{
								Service: "split-backend-1",
							},
							{
								Service: "split-backend-2",
							},
						},
					},
				},
			},
			expected: []service.MeshService{
				{
					Name:       "split-svc",
					Namespace:  "my-dst-ns",
					Port:       tests.ServicePort,
					TargetPort: tests.ServicePort,
					Protocol:   "http",
				},
				{
					Name:       "split-backend-1",
					Namespace:  "my-dst-ns",
					Port:       tests.ServicePort,
					TargetPort: tests.ServicePort,
					Protocol:   "http",
				},
				{
					Name:       "split-backend-2",
					Namespace:  "my-dst-ns",
					Port:       tests.ServicePort,
					TargetPort: tests.ServicePort,
					Protocol:   "http",
				},
			},
		},
		{
			name: "TrafficSplit apex service should not have duplicate when it does not have endpoints",
			id:   "my-src-ns.my-src-name",
			services: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-dst-ns",
						Name:      "split-backend-1",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "servicePort",
								Protocol:   corev1.ProtocolTCP,
								Port:       tests.ServicePort,
								TargetPort: intstr.FromInt(tests.ServicePort),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-dst-ns",
						Name:      "split-backend-2",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "servicePort",
								Protocol:   corev1.ProtocolTCP,
								Port:       tests.ServicePort,
								TargetPort: intstr.FromInt(tests.ServicePort),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-dst-ns",
						Name:      "split-svc",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:     "servicePort",
								Protocol: corev1.ProtocolTCP,
								Port:     tests.ServicePort,
								// No TargetPort: apex service without endpoint
							},
						},
					},
				},
			},
			trafficSplits: []*split.TrafficSplit{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "split",
						Namespace: "my-dst-ns",
					},
					Spec: split.TrafficSplitSpec{
						Service: "split-svc.my-dst-ns",
						Backends: []split.TrafficSplitBackend{
							{
								Service: "split-backend-1",
							},
							{
								Service: "split-backend-2",
							},
						},
					},
				},
			},
			expected: []service.MeshService{
				{
					Name:       "split-svc",
					Namespace:  "my-dst-ns",
					Port:       tests.ServicePort,
					TargetPort: tests.ServicePort,
					Protocol:   "http",
				},
				{
					Name:       "split-backend-1",
					Namespace:  "my-dst-ns",
					Port:       tests.ServicePort,
					TargetPort: tests.ServicePort,
					Protocol:   "http",
				},
				{
					Name:       "split-backend-2",
					Namespace:  "my-dst-ns",
					Port:       tests.ServicePort,
					TargetPort: tests.ServicePort,
					Protocol:   "http",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var meshServices []service.MeshService

			for _, k8sSvc := range tc.services {
				meshServices = append(meshServices, service.MeshService{Namespace: k8sSvc.Namespace, Name: k8sSvc.Name, Port: tests.ServicePort, TargetPort: uint16(k8sSvc.Spec.Ports[0].TargetPort.IntValue()), Protocol: "http"})
			}

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(true).Times(1)
			mockServiceProvider.EXPECT().ListServices().Return(meshServices).Times(1)
			if len(tc.trafficSplits) > 0 {
				mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficSplits).Times(1)
			}

			tassert.ElementsMatch(t, tc.expected, mc.listAllowedUpstreamServicesIncludeApex(tc.id))
		})
	}
}
