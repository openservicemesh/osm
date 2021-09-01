package catalog

import (
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/duration"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

var expectedBookbuyerOutbound []*trafficpolicy.OutboundTrafficPolicy = []*trafficpolicy.OutboundTrafficPolicy{
	{
		Name:      "bookstore-v1.default.svc.cluster.local",
		Hostnames: tests.BookstoreV1Hostnames,
		Routes: []*trafficpolicy.RouteWeightedClusters{
			{
				HTTPRouteMatch:   tests.WildCardRouteMatch,
				WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
			},
		},
	},
	{
		Name:      "bookstore-v2.default.svc.cluster.local",
		Hostnames: tests.BookstoreV2Hostnames,
		Routes: []*trafficpolicy.RouteWeightedClusters{
			{
				HTTPRouteMatch:   tests.WildCardRouteMatch,
				WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
			},
		},
	},
	{
		Name:      "bookstore-apex.default.svc.cluster.local",
		Hostnames: tests.BookstoreApexHostnames,
		Routes: []*trafficpolicy.RouteWeightedClusters{
			{
				HTTPRouteMatch:   tests.WildCardRouteMatch,
				WeightedClusters: mapset.NewSet(tests.BookstoreApexDefaultWeightedCluster),
			},
		},
	},
}

// BookstoreApexHostnamesSorted are the hostnames for the bookstore-apex service
var BookstoreApexHostnamesSorted = []string{
	"bookstore-apex",
	"bookstore-apex.default",
	"bookstore-apex.default.svc",
	"bookstore-apex.default.svc.cluster",
	"bookstore-apex.default.svc.cluster.local",
	"bookstore-apex:8888",
	"bookstore-apex.default:8888",
	"bookstore-apex.default.svc:8888",
	"bookstore-apex.default.svc.cluster:8888",
	"bookstore-apex.default.svc.cluster.local:8888",
}

func TestListOutboundTrafficPolicies(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                string
		downstreamSA        identity.ServiceIdentity
		apexMeshServices    []service.MeshService
		meshServices        []service.MeshService
		meshServiceAccounts []identity.K8sServiceAccount
		trafficsplits       []*split.TrafficSplit
		traffictargets      []*access.TrafficTarget
		trafficspecs        []*spec.HTTPRouteGroup
		expectedOutbound    []*trafficpolicy.OutboundTrafficPolicy
		permissiveMode      bool
	}{
		{
			name:                "only traffic targets",
			downstreamSA:        tests.BookbuyerServiceIdentity,
			apexMeshServices:    []service.MeshService{},
			meshServices:        []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			meshServiceAccounts: []identity.K8sServiceAccount{},
			trafficsplits:       []*split.TrafficSplit{},
			traffictargets:      []*access.TrafficTarget{&tests.TrafficTarget},
			trafficspecs:        []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
			expectedOutbound:    expectedBookbuyerOutbound,
			permissiveMode:      false,
		}, {
			name:         "traffic targets and traffic splits",
			downstreamSA: tests.BookbuyerServiceIdentity,
			apexMeshServices: []service.MeshService{
				{
					Name:      tests.BookstoreApexServiceName,
					Namespace: tests.Namespace,
				},
			},
			meshServices:        []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			meshServiceAccounts: []identity.K8sServiceAccount{},
			trafficsplits:       []*split.TrafficSplit{&tests.TrafficSplit},
			traffictargets:      []*access.TrafficTarget{&tests.TrafficTarget},
			trafficspecs:        []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-v1.default.svc.cluster.local",
					Hostnames: tests.BookstoreV1Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-v2.default.svc.cluster.local",
					Hostnames: tests.BookstoreV2Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: tests.BookstoreApexHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "default/bookstore-v1", Weight: 90},
								service.WeightedCluster{ClusterName: "default/bookstore-v2", Weight: 10},
							}),
						},
					},
				},
			},
			permissiveMode: false,
		},
		{
			name:         "traffic targets, traffic splits and host header",
			downstreamSA: tests.BookbuyerServiceIdentity,
			apexMeshServices: []service.MeshService{
				{
					Name:      tests.BookstoreApexServiceName,
					Namespace: tests.Namespace,
				},
			},
			meshServices:        []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			meshServiceAccounts: []identity.K8sServiceAccount{},
			trafficsplits:       []*split.TrafficSplit{&tests.TrafficSplit},
			traffictargets:      []*access.TrafficTarget{&tests.TrafficTarget},
			trafficspecs:        []*spec.HTTPRouteGroup{&tests.HTTPRouteGroupWithHost},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-v1.default.svc.cluster.local",
					Hostnames: tests.BookstoreV1Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-v2.default.svc.cluster.local",
					Hostnames: tests.BookstoreV2Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: BookstoreApexHostnamesSorted,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "default/bookstore-v1", Weight: 90},
								service.WeightedCluster{ClusterName: "default/bookstore-v2", Weight: 10},
							}),
						},
					},
				},
				{
					Name:      tests.HTTPHostHeader,
					Hostnames: []string{tests.HTTPHostHeader},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster, tests.BookstoreV2DefaultWeightedCluster),
						},
					},
				},
			},
			permissiveMode: false,
		},
		{
			name:         "only traffic splits, no traffic targets",
			downstreamSA: tests.BookbuyerServiceIdentity,
			apexMeshServices: []service.MeshService{
				{
					Name:      tests.BookstoreApexServiceName,
					Namespace: tests.Namespace,
				},
			},
			meshServices:        []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			meshServiceAccounts: []identity.K8sServiceAccount{},
			trafficsplits:       []*split.TrafficSplit{&tests.TrafficSplit},
			traffictargets:      []*access.TrafficTarget{},
			trafficspecs:        []*spec.HTTPRouteGroup{},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: tests.BookstoreApexHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "default/bookstore-v1", Weight: 90},
								service.WeightedCluster{ClusterName: "default/bookstore-v2", Weight: 10},
							}),
						},
					},
				},
			},
			permissiveMode: false,
		},
		{
			name:                "no traffic targets and no traffic splits",
			downstreamSA:        tests.BookbuyerServiceIdentity,
			apexMeshServices:    []service.MeshService{},
			meshServices:        []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			meshServiceAccounts: []identity.K8sServiceAccount{},
			trafficsplits:       []*split.TrafficSplit{},
			traffictargets:      []*access.TrafficTarget{},
			trafficspecs:        []*spec.HTTPRouteGroup{},
			expectedOutbound:    []*trafficpolicy.OutboundTrafficPolicy{},
			permissiveMode:      false,
		},
		{
			name:                "permissive mode",
			downstreamSA:        tests.BookbuyerServiceIdentity,
			apexMeshServices:    []service.MeshService{},
			meshServices:        []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookbuyerService},
			meshServiceAccounts: []identity.K8sServiceAccount{tests.BookbuyerServiceAccount, tests.BookstoreServiceAccount},
			trafficsplits:       []*split.TrafficSplit{},
			traffictargets:      []*access.TrafficTarget{},
			trafficspecs:        []*spec.HTTPRouteGroup{},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name: "bookstore-v1.default.svc.cluster.local",
					Hostnames: []string{
						"bookstore-v1",
						"bookstore-v1.default",
						"bookstore-v1.default.svc",
						"bookstore-v1.default.svc.cluster",
						"bookstore-v1.default.svc.cluster.local",
						"bookstore-v1:8888",
						"bookstore-v1.default:8888",
						"bookstore-v1.default.svc:8888",
						"bookstore-v1.default.svc.cluster:8888",
						"bookstore-v1.default.svc.cluster.local:8888",
					},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
				{
					Name: "bookstore-v2.default.svc.cluster.local",
					Hostnames: []string{
						"bookstore-v2",
						"bookstore-v2.default",
						"bookstore-v2.default.svc",
						"bookstore-v2.default.svc.cluster",
						"bookstore-v2.default.svc.cluster.local",
						"bookstore-v2:8888",
						"bookstore-v2.default:8888",
						"bookstore-v2.default.svc:8888",
						"bookstore-v2.default.svc.cluster:8888",
						"bookstore-v2.default.svc.cluster.local:8888",
					},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
						},
					},
				},
				{
					Name: "bookbuyer.default.svc.cluster.local",
					Hostnames: []string{
						"bookbuyer",
						"bookbuyer.default",
						"bookbuyer.default.svc",
						"bookbuyer.default.svc.cluster",
						"bookbuyer.default.svc.cluster.local",
						"bookbuyer:8888",
						"bookbuyer.default:8888",
						"bookbuyer.default.svc:8888",
						"bookbuyer.default.svc.cluster:8888",
						"bookbuyer.default.svc.cluster.local:8888",
					},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookbuyerDefaultWeightedCluster),
						},
					},
				},
			},
			permissiveMode: true,
		},
		{
			name:                "permissive mode with traffic splits",
			downstreamSA:        tests.BookbuyerServiceIdentity,
			apexMeshServices:    []service.MeshService{tests.BookstoreApexService},
			meshServices:        []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookbuyerService},
			meshServiceAccounts: []identity.K8sServiceAccount{tests.BookbuyerServiceAccount, tests.BookstoreServiceAccount},
			trafficsplits:       []*split.TrafficSplit{&tests.TrafficSplit},
			traffictargets:      []*access.TrafficTarget{},
			trafficspecs:        []*spec.HTTPRouteGroup{},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: tests.BookstoreApexHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "default/bookstore-v1", Weight: 90},
								service.WeightedCluster{ClusterName: "default/bookstore-v2", Weight: 10},
							}),
						},
					},
				},
				{
					Name: "bookstore-v1.default.svc.cluster.local",
					Hostnames: []string{
						"bookstore-v1",
						"bookstore-v1.default",
						"bookstore-v1.default.svc",
						"bookstore-v1.default.svc.cluster",
						"bookstore-v1.default.svc.cluster.local",
						"bookstore-v1:8888",
						"bookstore-v1.default:8888",
						"bookstore-v1.default.svc:8888",
						"bookstore-v1.default.svc.cluster:8888",
						"bookstore-v1.default.svc.cluster.local:8888",
					},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
				{
					Name: "bookstore-v2.default.svc.cluster.local",
					Hostnames: []string{
						"bookstore-v2",
						"bookstore-v2.default",
						"bookstore-v2.default.svc",
						"bookstore-v2.default.svc.cluster",
						"bookstore-v2.default.svc.cluster.local",
						"bookstore-v2:8888",
						"bookstore-v2.default:8888",
						"bookstore-v2.default.svc:8888",
						"bookstore-v2.default.svc.cluster:8888",
						"bookstore-v2.default.svc.cluster.local:8888",
					},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
						},
					},
				},
				{
					Name: "bookbuyer.default.svc.cluster.local",
					Hostnames: []string{
						"bookbuyer",
						"bookbuyer.default",
						"bookbuyer.default.svc",
						"bookbuyer.default.svc.cluster",
						"bookbuyer.default.svc.cluster.local",
						"bookbuyer:8888",
						"bookbuyer.default:8888",
						"bookbuyer.default.svc:8888",
						"bookbuyer.default.svc.cluster:8888",
						"bookbuyer.default.svc.cluster.local:8888",
					},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookbuyerDefaultWeightedCluster),
						},
					},
				},
			},
			permissiveMode: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
			mockServiceProvider := service.NewMockProvider(mockCtrl)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockServiceProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()
			for _, ms := range tc.apexMeshServices {
				apexK8sService := tests.NewServiceFixture(ms.Name, ms.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(ms).Return(apexK8sService).AnyTimes()
			}

			var services []*corev1.Service
			for _, ms := range tc.meshServices {
				k8sService := tests.NewServiceFixture(ms.Name, ms.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(ms).Return(k8sService).AnyTimes()
				services = append(services, k8sService)
			}

			if tc.permissiveMode {
				var serviceAccounts []*corev1.ServiceAccount
				for _, sa := range tc.meshServiceAccounts {
					k8sSvcAccount := tests.NewServiceAccountFixture(sa.Name, sa.Namespace)
					serviceAccounts = append(serviceAccounts, k8sSvcAccount)
				}
				mockKubeController.EXPECT().ListServices().Return(services).AnyTimes()
				mockKubeController.EXPECT().ListServiceAccounts().Return(serviceAccounts).AnyTimes()
				mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficsplits).AnyTimes()
			} else {
				mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficsplits).AnyTimes()
				mockMeshSpec.EXPECT().ListTrafficTargets().Return(tc.traffictargets).AnyTimes()
				mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return(tc.trafficspecs).AnyTimes()
				mockServiceProvider.EXPECT().GetServicesForServiceIdentity(tests.BookstoreServiceAccount.ToServiceIdentity()).Return([]service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService}, nil).AnyTimes()
				mockKubeController.EXPECT().GetService(tests.BookstoreApexService).Return(tests.NewServiceFixture(tests.BookstoreApexService.Name, tests.BookstoreApexService.Namespace, map[string]string{})).AnyTimes()
			}

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
				serviceProviders:   []service.Provider{mockServiceProvider},
				configurator:       mockConfigurator,
			}

			expectedServices := tc.meshServices
			expectedServices = append(expectedServices, tests.BookstoreApexService)
			for _, ms := range expectedServices {
				if ms.Namespace == tc.downstreamSA.ToK8sServiceAccount().Namespace {
					locality := service.LocalNS
					mockServiceProvider.EXPECT().GetHostnamesForService(ms, locality).Return(tests.ExpectedHostnames[ms.Name], nil).AnyTimes()
				}
			}

			mockServiceProvider.EXPECT().ListServices().Return(tc.meshServices, nil).AnyTimes()

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).AnyTimes()
			outbound := mc.ListOutboundTrafficPolicies(tc.downstreamSA)
			assert.ElementsMatch(tc.expectedOutbound, outbound)
		})
	}
}

func TestListOutboundTrafficPoliciesForTrafficSplits(t *testing.T) {
	assert := tassert.New(t)

	testSplit1 := split.TrafficSplit{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "bar",
		},
		Spec: split.TrafficSplitSpec{
			Service: "apex-split-1",
			Backends: []split.TrafficSplitBackend{
				{
					Service: tests.BookstoreV1ServiceName,
					Weight:  tests.Weight10,
				},
				{
					Service: tests.BookstoreV2ServiceName,
					Weight:  tests.Weight90,
				},
			},
		},
	}

	testSplit1NamespacedHostnames := []string{
		"apex-split-1.bar",
		"apex-split-1.bar.svc",
		"apex-split-1.bar.svc.cluster",
		"apex-split-1.bar.svc.cluster.local",
		"apex-split-1.bar:8888",
		"apex-split-1.bar.svc:8888",
		"apex-split-1.bar.svc.cluster:8888",
		"apex-split-1.bar.svc.cluster.local:8888",
	}

	testSplit2 := split.TrafficSplit{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "bar",
		},
		Spec: split.TrafficSplitSpec{
			Service: "apex-split-1",
			Backends: []split.TrafficSplitBackend{
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

	testSplit3 := split.TrafficSplit{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "baz",
		},
		Spec: split.TrafficSplitSpec{
			Service: "apex-split-1",
			Backends: []split.TrafficSplitBackend{
				{
					Service: tests.BookstoreV1ServiceName,
					Weight:  tests.Weight10,
				},
				{
					Service: tests.BookstoreV2ServiceName,
					Weight:  tests.Weight90,
				},
			},
		},
	}

	testSplit4 := split.TrafficSplit{
		ObjectMeta: v1.ObjectMeta{
			Namespace: tests.Namespace,
		},
		Spec: split.TrafficSplitSpec{
			Service: tests.BookstoreApexServiceName + "." + tests.Namespace,
			Backends: []split.TrafficSplitBackend{
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

	testSplit3NamespacedHostnames := []string{
		"apex-split-1.baz",
		"apex-split-1.baz.svc",
		"apex-split-1.baz.svc.cluster",
		"apex-split-1.baz.svc.cluster.local",
		"apex-split-1.baz:8888",
		"apex-split-1.baz.svc:8888",
		"apex-split-1.baz.svc.cluster:8888",
		"apex-split-1.baz.svc.cluster.local:8888",
	}

	testCases := []struct {
		name             string
		sourceNamespace  string
		trafficsplits    []*split.TrafficSplit
		expectedPolicies []*trafficpolicy.OutboundTrafficPolicy
		expectedRoutes   []*trafficpolicy.RouteWeightedClusters
		apexMeshServices []service.MeshService
	}{
		{
			name:            "single traffic split policy in different namespace",
			sourceNamespace: "foo",
			trafficsplits:   []*split.TrafficSplit{&tests.TrafficSplit},
			apexMeshServices: []service.MeshService{
				{
					Name:      tests.BookstoreApexServiceName,
					Namespace: tests.Namespace,
				},
			},
			expectedPolicies: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: tests.BookstoreApexNamespacedHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "default/bookstore-v1", Weight: 90},
								service.WeightedCluster{ClusterName: "default/bookstore-v2", Weight: 10},
							}),
						},
					},
				},
			},
		},
		{
			name:            "single traffic split policy in different namespace with namespaced root service",
			sourceNamespace: "foo",
			trafficsplits:   []*split.TrafficSplit{&testSplit4},
			apexMeshServices: []service.MeshService{
				{
					Name:      tests.BookstoreApexServiceName,
					Namespace: tests.Namespace,
				},
			},
			expectedPolicies: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: tests.BookstoreApexNamespacedHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "default/bookstore-v1", Weight: 90},
								service.WeightedCluster{ClusterName: "default/bookstore-v2", Weight: 10},
							}),
						},
					},
				},
			},
		},
		{
			name:            "single traffic split policy in same namespace",
			sourceNamespace: tests.Namespace,
			trafficsplits:   []*split.TrafficSplit{&tests.TrafficSplit},
			apexMeshServices: []service.MeshService{
				{
					Name:      tests.BookstoreApexServiceName,
					Namespace: tests.Namespace,
				},
			},
			expectedPolicies: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: tests.BookstoreApexHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "default/bookstore-v1", Weight: 90},
								service.WeightedCluster{ClusterName: "default/bookstore-v2", Weight: 10},
							}),
						},
					},
				},
			},
		},
		{
			name:            "multiple traffic splits",
			sourceNamespace: "foo",
			trafficsplits:   []*split.TrafficSplit{&tests.TrafficSplit, &testSplit1},
			apexMeshServices: []service.MeshService{
				{
					Name:      tests.BookstoreApexServiceName,
					Namespace: tests.Namespace,
				},
				{
					Name:      "apex-split-1",
					Namespace: "bar",
				},
			},
			expectedPolicies: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: tests.BookstoreApexNamespacedHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "default/bookstore-v1", Weight: 90},
								service.WeightedCluster{ClusterName: "default/bookstore-v2", Weight: 10},
							}),
						},
					},
				},
				{
					Name:      "apex-split-1.bar.svc.cluster.local",
					Hostnames: testSplit1NamespacedHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "bar/bookstore-v1", Weight: 10},
								service.WeightedCluster{ClusterName: "bar/bookstore-v2", Weight: 90},
							}),
						},
					},
				},
			},
		},
		{
			name:            "duplicate traffic splits",
			sourceNamespace: "foo",
			trafficsplits:   []*split.TrafficSplit{&testSplit1, &testSplit2},
			apexMeshServices: []service.MeshService{
				{
					Name:      "apex-split-1",
					Namespace: "bar",
				},
			},
			expectedPolicies: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "apex-split-1.bar.svc.cluster.local",
					Hostnames: testSplit1NamespacedHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "bar/bookstore-v1", Weight: 10},
								service.WeightedCluster{ClusterName: "bar/bookstore-v2", Weight: 90},
							}),
						},
					},
				},
			},
		},
		{
			name:            "duplicate traffic splits different namespaces",
			sourceNamespace: "foo",
			trafficsplits:   []*split.TrafficSplit{&testSplit1, &testSplit3},
			apexMeshServices: []service.MeshService{
				{
					Name:      "apex-split-1",
					Namespace: "bar",
				},
				{
					Name:      "apex-split-1",
					Namespace: "baz",
				},
			},
			expectedPolicies: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "apex-split-1.bar.svc.cluster.local",
					Hostnames: testSplit1NamespacedHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "bar/bookstore-v1", Weight: 10},
								service.WeightedCluster{ClusterName: "bar/bookstore-v2", Weight: 90},
							}),
						},
					},
				},
				{
					Name:      "apex-split-1.baz.svc.cluster.local",
					Hostnames: testSplit3NamespacedHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "baz/bookstore-v1", Weight: 10},
								service.WeightedCluster{ClusterName: "baz/bookstore-v2", Weight: 90},
							}),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
			mockServiceProvider := service.NewMockProvider(mockCtrl)

			for _, ms := range tc.apexMeshServices {
				apexK8sService := tests.NewServiceFixture(ms.Name, ms.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(ms).Return(apexK8sService).AnyTimes()
			}
			mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficsplits).AnyTimes()

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
				serviceProviders:   []service.Provider{mockServiceProvider},
			}

			for _, ms := range tc.apexMeshServices {
				locality := service.LocalCluster
				if ms.Namespace == tc.sourceNamespace && ms.Namespace == "default" {
					locality = service.LocalNS
					mockServiceProvider.EXPECT().GetHostnamesForService(ms, locality).Return(tests.ExpectedHostnames[ms.Name], nil).AnyTimes()
				} else if ms.Namespace == "baz" {
					mockServiceProvider.EXPECT().GetHostnamesForService(ms, locality).Return(tests.ExpectedHostnames[ms.Name+"-baz-namespaced"], nil).AnyTimes()
				} else {
					mockServiceProvider.EXPECT().GetHostnamesForService(ms, locality).Return(tests.ExpectedHostnames[ms.Name+"-namespaced"], nil).AnyTimes()
				}
			}

			actual := mc.listOutboundTrafficPoliciesForTrafficSplits(tc.sourceNamespace)

			assert.ElementsMatch(tc.expectedPolicies, actual)
		})
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

func TestBuildOutboundPermissiveModePolicies(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
	mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
	mockServiceProvider := service.NewMockProvider(mockCtrl)

	mc := MeshCatalog{
		kubeController:     mockKubeController,
		meshSpec:           mockMeshSpec,
		endpointsProviders: []endpoint.Provider{mockEndpointProvider},
		serviceProviders:   []service.Provider{mockServiceProvider},
	}

	testCases := []struct {
		name                     string
		sourceNamespace          string
		services                 map[string]string
		expectedOutboundPolicies []*trafficpolicy.OutboundTrafficPolicy
	}{
		{
			name:            "outbound traffic policies for permissive mode",
			sourceNamespace: "test",
			services:        map[string]string{"bookstore-v1": "default", "bookstore-apex": "default", "bookbuyer": "test"},
			expectedOutboundPolicies: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name: "bookstore-apex.default.svc.cluster.local",
					Hostnames: []string{
						"bookstore-apex.default",
						"bookstore-apex.default.svc",
						"bookstore-apex.default.svc.cluster",
						"bookstore-apex.default.svc.cluster.local",
						"bookstore-apex.default:8888",
						"bookstore-apex.default.svc:8888",
						"bookstore-apex.default.svc.cluster:8888",
						"bookstore-apex.default.svc.cluster.local:8888",
					},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreApexDefaultWeightedCluster),
						},
					},
				},
				{
					Name: "bookstore-v1.default.svc.cluster.local",
					Hostnames: []string{
						"bookstore-v1.default",
						"bookstore-v1.default.svc",
						"bookstore-v1.default.svc.cluster",
						"bookstore-v1.default.svc.cluster.local",
						"bookstore-v1.default:8888",
						"bookstore-v1.default.svc:8888",
						"bookstore-v1.default.svc.cluster:8888",
						"bookstore-v1.default.svc.cluster.local:8888",
					},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
				{
					Name: "bookbuyer.test.svc.cluster.local",
					Hostnames: []string{
						"bookbuyer",
						"bookbuyer.test",
						"bookbuyer.test.svc",
						"bookbuyer.test.svc.cluster",
						"bookbuyer.test.svc.cluster.local",
						"bookbuyer:8888",
						"bookbuyer.test:8888",
						"bookbuyer.test.svc:8888",
						"bookbuyer.test.svc.cluster:8888",
						"bookbuyer.test.svc.cluster.local:8888",
					},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "test/bookbuyer",
								Weight:      100,
							}),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var meshServices []service.MeshService

			for name, namespace := range tc.services {
				svcFixture := tests.NewServiceFixture(name, namespace, map[string]string{})
				meshSvc := tests.NewMeshServiceFixture(name, namespace)
				meshServices = append(meshServices, meshSvc)
				mockKubeController.EXPECT().GetService(meshSvc).Return(svcFixture)
			}

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockServiceProvider.EXPECT().ListServices().Return(meshServices, nil)

			for _, ms := range meshServices {
				locality := service.LocalCluster
				if ms.Namespace == tc.sourceNamespace && ms.Namespace == "default" {
					locality = service.LocalNS
					mockServiceProvider.EXPECT().GetHostnamesForService(ms, locality).Return(tests.ExpectedHostnames[ms.Name], nil).AnyTimes()
				} else if ms.Namespace == tc.sourceNamespace && ms.Namespace == "test" {
					locality = service.LocalNS
					mockServiceProvider.EXPECT().GetHostnamesForService(ms, locality).Return(tests.ExpectedHostnames[ms.Name+"-test"], nil).AnyTimes()
				} else {
					mockServiceProvider.EXPECT().GetHostnamesForService(ms, locality).Return(tests.ExpectedHostnames[ms.Name+"-namespaced"], nil).AnyTimes()
				}
			}
			actual := mc.buildOutboundPermissiveModePolicies(tc.sourceNamespace)
			assert.Len(actual, len(tc.expectedOutboundPolicies))
			assert.ElementsMatch(tc.expectedOutboundPolicies, actual)
		})
	}
}

func TestBuildOutboundPolicies(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name             string
		sourceSA         identity.K8sServiceAccount
		destSA           identity.K8sServiceAccount
		destMeshService  service.MeshService
		trafficSpec      spec.HTTPRouteGroup
		trafficSplit     split.TrafficSplit
		expectedOutbound []*trafficpolicy.OutboundTrafficPolicy
	}{
		{
			name:            "outbound policy without host header",
			sourceSA:        tests.BookbuyerServiceAccount,
			destSA:          tests.BookstoreServiceAccount,
			destMeshService: tests.BookstoreV1Service,
			trafficSpec:     tests.HTTPRouteGroup,
			trafficSplit:    split.TrafficSplit{},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      tests.BookstoreV1Service.FQDN(),
					Hostnames: tests.BookstoreV1Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
			},
		},
		{
			name:            "outbound policy with host header",
			sourceSA:        tests.BookbuyerServiceAccount,
			destSA:          tests.BookstoreServiceAccount,
			destMeshService: tests.BookstoreV1Service,
			trafficSpec:     tests.HTTPRouteGroupWithHost,
			trafficSplit:    split.TrafficSplit{},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      tests.BookstoreV1Service.FQDN(),
					Hostnames: tests.BookstoreV1Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
				// a new outbound policy with the specified host
				{
					Name:      tests.HTTPHostHeader,
					Hostnames: []string{tests.HTTPHostHeader},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
			mockServiceProvider := service.NewMockProvider(mockCtrl)

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
				serviceProviders:   []service.Provider{mockServiceProvider},
			}

			destK8sService := tests.NewServiceFixture(tc.destMeshService.Name, tc.destMeshService.Namespace, map[string]string{})

			mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*spec.HTTPRouteGroup{&tc.trafficSpec}).AnyTimes()
			mockMeshSpec.EXPECT().ListTrafficSplits().Return([]*split.TrafficSplit{&tc.trafficSplit}).AnyTimes()
			mockServiceProvider.EXPECT().GetServicesForServiceIdentity(tc.destSA.ToServiceIdentity()).Return([]service.MeshService{tc.destMeshService}, nil).AnyTimes()
			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockServiceProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockKubeController.EXPECT().GetService(tc.destMeshService).Return(destK8sService).AnyTimes()

			trafficTarget := tests.NewSMITrafficTarget(tc.sourceSA.ToServiceIdentity(), tc.destSA.ToServiceIdentity())

			mockServiceProvider.EXPECT().GetHostnamesForService(tc.destMeshService, service.LocalNS).Return(tests.ExpectedHostnames[tc.destMeshService.Name], nil).AnyTimes()

			actual := mc.buildOutboundPolicies(tc.sourceSA.ToServiceIdentity(), &trafficTarget)
			assert.ElementsMatch(tc.expectedOutbound, actual)
		})
	}
}

func TestListOutboundPoliciesForTrafficTargets(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name              string
		serviceIdentity   identity.ServiceIdentity
		apexMeshServices  []service.MeshService
		traffictargets    []*access.TrafficTarget
		trafficsplits     []*split.TrafficSplit
		trafficspecs      []*spec.HTTPRouteGroup
		enableRetryPolicy v1alpha1.FeatureFlags
		expectedOutbound  []*trafficpolicy.OutboundTrafficPolicy
	}{
		{
			name:              "only traffic targets",
			serviceIdentity:   tests.BookbuyerServiceIdentity,
			apexMeshServices:  []service.MeshService{},
			traffictargets:    []*access.TrafficTarget{&tests.TrafficTarget},
			trafficsplits:     []*split.TrafficSplit{},
			trafficspecs:      []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
			enableRetryPolicy: v1alpha1.FeatureFlags{EnableRetryPolicy: false},
			expectedOutbound:  expectedBookbuyerOutbound,
		},
		{
			name:              "no traffic targets and no traffic splits",
			serviceIdentity:   tests.BookbuyerServiceIdentity,
			apexMeshServices:  []service.MeshService{},
			traffictargets:    []*access.TrafficTarget{},
			trafficsplits:     []*split.TrafficSplit{},
			trafficspecs:      []*spec.HTTPRouteGroup{},
			enableRetryPolicy: v1alpha1.FeatureFlags{EnableRetryPolicy: false},
			expectedOutbound:  []*trafficpolicy.OutboundTrafficPolicy{},
		},
		{
			name:              "only traffic targets with a route having host",
			serviceIdentity:   tests.BookbuyerServiceIdentity,
			apexMeshServices:  []service.MeshService{},
			traffictargets:    []*access.TrafficTarget{&tests.TrafficTarget},
			trafficsplits:     []*split.TrafficSplit{},
			trafficspecs:      []*spec.HTTPRouteGroup{&tests.HTTPRouteGroupWithHost},
			enableRetryPolicy: v1alpha1.FeatureFlags{EnableRetryPolicy: false},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-v1.default.svc.cluster.local",
					Hostnames: tests.BookstoreV1Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-v2.default.svc.cluster.local",
					Hostnames: tests.BookstoreV2Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: tests.BookstoreApexHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreApexDefaultWeightedCluster),
						},
					},
				},
				{
					Name:      tests.HTTPHostHeader,
					Hostnames: []string{tests.HTTPHostHeader},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster, tests.BookstoreV2DefaultWeightedCluster, tests.BookstoreApexDefaultWeightedCluster),
						},
					},
				},
			},
		},
		{
			name:              "traffic targets with a route having host and retry policy",
			serviceIdentity:   tests.BookbuyerServiceIdentity,
			apexMeshServices:  []service.MeshService{},
			traffictargets:    []*access.TrafficTarget{&tests.TrafficTarget},
			trafficsplits:     []*split.TrafficSplit{},
			trafficspecs:      []*spec.HTTPRouteGroup{&tests.HTTPRouteGroupWithHost},
			enableRetryPolicy: v1alpha1.FeatureFlags{EnableRetryPolicy: true},

			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      "bookstore-v1.default.svc.cluster.local",
					Hostnames: tests.BookstoreV1Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-v2.default.svc.cluster.local",
					Hostnames: tests.BookstoreV2Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-apex.default.svc.cluster.local",
					Hostnames: tests.BookstoreApexHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreApexDefaultWeightedCluster),
						},
					},
				},
				{
					Name:      tests.HTTPHostHeader,
					Hostnames: []string{tests.HTTPHostHeader},
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster, tests.BookstoreV2DefaultWeightedCluster, tests.BookstoreApexDefaultWeightedCluster),
							RetryPolicy:      trafficpolicy.RetryPolicy{},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
			mockServiceProvider := service.NewMockProvider(mockCtrl)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

			for _, ms := range tc.apexMeshServices {
				apexK8sService := tests.NewServiceFixture(ms.Name, ms.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(ms).Return(apexK8sService).AnyTimes()
			}
			mockMeshSpec.EXPECT().ListTrafficTargets().Return(tc.traffictargets).AnyTimes()
			mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return(tc.trafficspecs).AnyTimes()
			mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficsplits).AnyTimes()
			mockServiceProvider.EXPECT().GetServicesForServiceIdentity(tests.BookstoreServiceAccount.ToServiceIdentity()).Return([]service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService}, nil).AnyTimes()
			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockServiceProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockKubeController.EXPECT().GetService(tests.BookstoreV1Service).Return(tests.NewServiceFixture(tests.BookstoreV1Service.Name, tests.BookstoreV1Service.Namespace, map[string]string{})).AnyTimes()
			mockKubeController.EXPECT().GetService(tests.BookstoreV2Service).Return(tests.NewServiceFixture(tests.BookstoreV2Service.Name, tests.BookstoreV2Service.Namespace, map[string]string{})).AnyTimes()
			mockKubeController.EXPECT().GetService(tests.BookstoreApexService).Return(tests.NewServiceFixture(tests.BookstoreApexService.Name, tests.BookstoreApexService.Namespace, map[string]string{})).AnyTimes()
			mockConfigurator.EXPECT().GetFeatureFlags().Return(tc.enableRetryPolicy).AnyTimes()
			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
				serviceProviders:   []service.Provider{mockServiceProvider},
				configurator:       mockConfigurator,
			}

			meshServices := []service.MeshService{
				tests.BookstoreV1Service,
				tests.BookstoreV2Service,
				tests.BookstoreApexService,
			}

			for _, ms := range meshServices {
				if ms.Namespace == tc.serviceIdentity.ToK8sServiceAccount().Namespace {
					locality := service.LocalNS
					mockServiceProvider.EXPECT().GetHostnamesForService(ms, locality).Return(tests.ExpectedHostnames[ms.Name], nil).AnyTimes()
				}
			}
			outbound := mc.listOutboundPoliciesForTrafficTargets(tc.serviceIdentity)
			assert.ElementsMatch(tc.expectedOutbound, outbound)
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
	mockServiceProvider.EXPECT().GetServicesForServiceIdentity(destSA.ToServiceIdentity()).Return([]service.MeshService{destMeshService}, nil).AnyTimes()
	mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
	mockServiceProvider.EXPECT().GetID().Return("fake").AnyTimes()
	mockKubeController.EXPECT().GetService(destMeshService).Return(destK8sService).AnyTimes()

	trafficTarget := &access.TrafficTarget{
		TypeMeta: v1.TypeMeta{
			APIVersion: "access.smi-spec.io/v1alpha3",
			Kind:       "TrafficTarget",
		},
		ObjectMeta: v1.ObjectMeta{
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

	actual, err := mc.getDestinationServicesFromTrafficTarget(trafficTarget)
	assert.Nil(err)
	assert.Equal([]service.MeshService{destMeshService}, actual)
}

func TestGetOutboundTCPFilter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type testCase struct {
		name          string
		upstream      service.MeshService
		trafficSplits []*split.TrafficSplit
		expected      []service.WeightedCluster
	}

	testCases := []testCase{
		{
			name: "TCP filter for upstream without any traffic split policies",
			upstream: service.MeshService{
				Name:      "foo",
				Namespace: "bar",
			},
			trafficSplits: nil,
			expected:      nil,
		},
		{
			name: "TCP filter for upstream with matching traffic split policy",
			upstream: service.MeshService{
				Name:      "foo",
				Namespace: "bar",
			},
			trafficSplits: []*split.TrafficSplit{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Spec: split.TrafficSplitSpec{
						Service: "foo.bar.svc.cluster.local",
						Backends: []split.TrafficSplitBackend{
							{
								Service: "foo-v1",
								Weight:  10,
							},
							{
								Service: "foo-v2",
								Weight:  90,
							},
						},
					},
				},
			},
			expected: []service.WeightedCluster{
				{
					ClusterName: "bar/foo-v1",
					Weight:      10,
				},
				{
					ClusterName: "bar/foo-v2",
					Weight:      90,
				},
			},
		},
		{
			name: "TCP filter for upstream without matching traffic split policy",
			upstream: service.MeshService{
				Name:      "foo",
				Namespace: "bar",
			},
			trafficSplits: []*split.TrafficSplit{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Spec: split.TrafficSplitSpec{
						Service: "not-upstream.bar.svc.cluster.local", // Root service is not the upstream the filter is being built for
						Backends: []split.TrafficSplitBackend{
							{
								Service: "foo-v1",
								Weight:  10,
							},
							{
								Service: "foo-v2",
								Weight:  90,
							},
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "TCP filter for upstream with multiple matching policies, pick first",
			upstream: service.MeshService{
				Name:      "foo",
				Namespace: "bar",
			},
			trafficSplits: []*split.TrafficSplit{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Spec: split.TrafficSplitSpec{
						Service: "foo.bar.svc.cluster.local",
						Backends: []split.TrafficSplitBackend{
							{
								Service: "foo-v1",
								Weight:  10,
							},
							{
								Service: "foo-v2",
								Weight:  90,
							},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Spec: split.TrafficSplitSpec{
						Service: "foo.bar.svc.cluster.local",
						Backends: []split.TrafficSplitBackend{
							{
								Service: "foo-v3",
								Weight:  10,
							},
							{
								Service: "foo-v4",
								Weight:  90,
							},
						},
					},
				},
			},
			expected: []service.WeightedCluster{
				{
					ClusterName: "bar/foo-v1",
					Weight:      10,
				},
				{
					ClusterName: "bar/foo-v2",
					Weight:      90,
				},
			},
		},
		{
			name: "TCP filter for upstream with matching traffic split policy including a backend with 0 weight",
			upstream: service.MeshService{
				Name:      "foo",
				Namespace: "bar",
			},
			trafficSplits: []*split.TrafficSplit{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Spec: split.TrafficSplitSpec{
						Service: "foo.bar.svc.cluster.local",
						Backends: []split.TrafficSplitBackend{
							{
								Service: "foo-v1",
								Weight:  100,
							},
							{
								Service: "foo-v2",
								Weight:  0,
							},
						},
					},
				},
			},
			expected: []service.WeightedCluster{
				{
					ClusterName: "bar/foo-v1",
					Weight:      100,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficSplits).Times(1)

			mc := MeshCatalog{
				meshSpec: mockMeshSpec,
			}

			clusterWeights := mc.GetWeightedClustersForUpstream(tc.upstream)

			assert := tassert.New(t)
			assert.Equal(tc.expected, clusterWeights)
		})
	}
}

func TestListMeshServicesForIdentity(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockController := k8s.NewMockController(mockCtrl)
	mockServiceProvider := service.NewMockProvider(mockCtrl)
	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()
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
					ObjectMeta: v1.ObjectMeta{
						Namespace: "my-dst-ns",
						Name:      "split-backend-1",
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "my-dst-ns",
						Name:      "split-backend-2",
					},
				},
			},
			trafficSplits: []*split.TrafficSplit{
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "wrong-ns",
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
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
					Name:      "split-svc",
					Namespace: "my-dst-ns",
				},
				{
					Name:      "split-backend-1",
					Namespace: "my-dst-ns",
				},
				{
					Name:      "split-backend-2",
					Namespace: "my-dst-ns",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var meshServices []service.MeshService

			for _, k8Svc := range tc.services {
				meshServices = append(meshServices, utils.K8sSvcToMeshSvc(k8Svc))
			}

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(true).Times(1)
			mockServiceProvider.EXPECT().ListServices().Return(meshServices, nil).Times(1)
			if len(tc.trafficSplits) > 0 {
				mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficSplits).Times(1)
			}

			tassert.ElementsMatch(t, tc.expected, mc.ListMeshServicesForIdentity(tc.id))
		})
	}
}

func TestGetRetryPolicy(t *testing.T) {
	testCases := []struct {
		name           string
		trafficTarget  access.TrafficTarget
		expRetryPolicy *trafficpolicy.RetryPolicy
	}{
		{
			name:           "no retry policy",
			trafficTarget:  access.TrafficTarget{},
			expRetryPolicy: nil,
		},
		{
			name: "retry policy but invalid fields",
			trafficTarget: access.TrafficTarget{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						RetryOnAnnotation:       "5xx",
						NumRetriesAnnotation:    "b",
						PerTryTimeoutAnnotation: "c",
					},
				},
			},
			expRetryPolicy: &trafficpolicy.RetryPolicy{
				RetryOn: "5xx",
			},
		},
		{
			name: "retry policy missing required field",
			trafficTarget: access.TrafficTarget{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						NumRetriesAnnotation:    "b",
						PerTryTimeoutAnnotation: "c",
					},
				},
			},
			expRetryPolicy: nil,
		},
		{
			name: "valid retry policy",
			trafficTarget: access.TrafficTarget{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						RetryOnAnnotation:       "5xx",
						NumRetriesAnnotation:    "3",
						PerTryTimeoutAnnotation: "10s",
					},
				},
			},
			expRetryPolicy: &trafficpolicy.RetryPolicy{
				RetryOn:       "5xx",
				NumRetries:    &wrapperspb.UInt32Value{Value: 3},
				PerTryTimeout: &duration.Duration{Seconds: 10},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			retryPolicy := getRetryPolicy(&tc.trafficTarget)
			assert.Equal(tc.expRetryPolicy, retryPolicy)
		})
	}
}
