package catalog

import (
	"testing"

	"github.com/openservicemesh/osm/pkg/identity"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	v1alpha2 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

var expectedBookbuyerOutbound []*trafficpolicy.OutboundTrafficPolicy = []*trafficpolicy.OutboundTrafficPolicy{
	{
		Name:      "bookstore-v1",
		Hostnames: tests.BookstoreV1Hostnames,
		Routes: []*trafficpolicy.RouteWeightedClusters{
			{
				HTTPRouteMatch:   tests.WildCardRouteMatch,
				WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
			},
		},
	},
	{
		Name:      "bookstore-v2",
		Hostnames: tests.BookstoreV2Hostnames,
		Routes: []*trafficpolicy.RouteWeightedClusters{
			{
				HTTPRouteMatch:   tests.WildCardRouteMatch,
				WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
			},
		},
	},
	{
		Name:      "bookstore-apex",
		Hostnames: tests.BookstoreApexHostnames,
		Routes: []*trafficpolicy.RouteWeightedClusters{
			{
				HTTPRouteMatch:   tests.WildCardRouteMatch,
				WeightedClusters: mapset.NewSet(tests.BookstoreApexDefaultWeightedCluster),
			},
		},
	},
}

func TestListOutboundTrafficPolicies(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                string
		downstreamSA        identity.K8sServiceAccount
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
			downstreamSA:        tests.BookbuyerServiceAccount,
			apexMeshServices:    []service.MeshService{},
			meshServices:        []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			meshServiceAccounts: []identity.K8sServiceAccount{},
			trafficsplits:       []*split.TrafficSplit{},
			traffictargets:      []*access.TrafficTarget{&tests.TrafficTarget},
			trafficspecs:        []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
			expectedOutbound:    expectedBookbuyerOutbound,
			permissiveMode:      false,
		},
		{
			name:         "traffic targets and traffic splits",
			downstreamSA: tests.BookbuyerServiceAccount,
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
					Name:      "bookstore-v1",
					Hostnames: tests.BookstoreV1Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-v2",
					Hostnames: tests.BookstoreV2Hostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch:   tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
						},
					},
				},
				{
					Name:      "bookstore-apex",
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
			name:         "only traffic splits, no traffic targets",
			downstreamSA: tests.BookbuyerServiceAccount,
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
					Name:      "bookstore-apex",
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
			downstreamSA:        tests.BookbuyerServiceAccount,
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
			downstreamSA:        tests.BookbuyerServiceAccount,
			apexMeshServices:    []service.MeshService{},
			meshServices:        []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookbuyerService},
			meshServiceAccounts: []identity.K8sServiceAccount{tests.BookbuyerServiceAccount, tests.BookstoreServiceAccount},
			trafficsplits:       []*split.TrafficSplit{},
			traffictargets:      []*access.TrafficTarget{},
			trafficspecs:        []*spec.HTTPRouteGroup{},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name: "bookstore-v1.default",
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
					Name: "bookstore-v2.default",
					Hostnames: []string{
						"bookstore-v2.default",
						"bookstore-v2.default.svc",
						"bookstore-v2.default.svc.cluster",
						"bookstore-v2.default.svc.cluster.local",
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
					Name: "bookbuyer.default",
					Hostnames: []string{
						"bookbuyer.default",
						"bookbuyer.default.svc",
						"bookbuyer.default.svc.cluster",
						"bookbuyer.default.svc.cluster.local",
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
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()

			for _, ms := range tc.apexMeshServices {
				apexK8sService := tests.NewServiceFixture(ms.Name, ms.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(ms).Return(apexK8sService).AnyTimes()
			}

			services := []*corev1.Service{}
			for _, ms := range tc.meshServices {
				k8sService := tests.NewServiceFixture(ms.Name, ms.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(ms).Return(k8sService).AnyTimes()
				services = append(services, k8sService)
			}

			if tc.permissiveMode {
				serviceAccounts := []*corev1.ServiceAccount{}
				for _, sa := range tc.meshServiceAccounts {
					k8sSvcAccount := tests.NewServiceAccountFixture(sa.Name, sa.Namespace)
					serviceAccounts = append(serviceAccounts, k8sSvcAccount)
				}
				mockKubeController.EXPECT().ListServices().Return(services).AnyTimes()
				mockKubeController.EXPECT().ListServiceAccounts().Return(serviceAccounts).AnyTimes()
			} else {
				mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficsplits).AnyTimes()
				mockMeshSpec.EXPECT().ListTrafficTargets().Return(tc.traffictargets).AnyTimes()
				mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return(tc.trafficspecs).AnyTimes()
				mockEndpointProvider.EXPECT().GetServicesForServiceAccount(tests.BookstoreServiceAccount).Return([]service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService}, nil).AnyTimes()
				mockKubeController.EXPECT().GetService(tests.BookstoreApexService).Return(tests.NewServiceFixture(tests.BookstoreApexService.Name, tests.BookstoreApexService.Namespace, map[string]string{})).AnyTimes()
			}

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
				configurator:       mockConfigurator,
			}

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
		Spec: v1alpha2.TrafficSplitSpec{
			Service: tests.BookstoreApexServiceName + "." + tests.Namespace,
			Backends: []v1alpha2.TrafficSplitBackend{
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
					Name:      "bookstore-apex.default",
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
					Name:      "bookstore-apex.default",
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
					Name:      "bookstore-apex",
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
					Name:      "bookstore-apex.default",
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
					Name:      "apex-split-1.bar",
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
					Name:      "apex-split-1.bar",
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
					Name:      "apex-split-1.bar",
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
					Name:      "apex-split-1.baz",
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

			for _, ms := range tc.apexMeshServices {
				apexK8sService := tests.NewServiceFixture(ms.Name, ms.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(ms).Return(apexK8sService).AnyTimes()
			}
			mockMeshSpec.EXPECT().ListTrafficSplits().Return(tc.trafficsplits).AnyTimes()

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
			}

			actual := mc.listOutboundTrafficPoliciesForTrafficSplits(tc.sourceNamespace)

			assert.ElementsMatch(tc.expectedPolicies, actual)
		})
	}
}

func TestListAllowedOutboundServicesForIdentity(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name           string
		serviceAccount identity.K8sServiceAccount
		expectedList   []service.MeshService
		permissiveMode bool
	}{
		{
			name:           "traffic targets configured for service account",
			serviceAccount: tests.BookbuyerServiceAccount,
			expectedList:   []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService},
			permissiveMode: false,
		},
		{
			name: "traffic targets not configured for service account",
			serviceAccount: identity.K8sServiceAccount{
				Name:      "some-name",
				Namespace: "some-ns",
			},
			expectedList:   nil,
			permissiveMode: false,
		},
		{
			name:           "permissive mode enabled",
			serviceAccount: tests.BookstoreServiceAccount,
			expectedList:   []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService, tests.BookbuyerService},
			permissiveMode: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mc := newFakeMeshCatalogForRoutes(t, testParams{
				permissiveMode: tc.permissiveMode,
			})
			actualList := mc.ListAllowedOutboundServicesForIdentity(tc.serviceAccount)
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

	mc := MeshCatalog{
		kubeController:     mockKubeController,
		meshSpec:           mockMeshSpec,
		endpointsProviders: []endpoint.Provider{mockEndpointProvider},
	}

	testCases := []struct {
		name                     string
		services                 map[string]string
		expectedOutboundPolicies []*trafficpolicy.OutboundTrafficPolicy
	}{
		{
			name:     "outbound traffic policies for permissive mode",
			services: map[string]string{"bookstore-v1": "default", "bookstore-apex": "default", "bookbuyer": "default"},
			expectedOutboundPolicies: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name: "bookstore-apex.default",
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
					Name: "bookstore-v1.default",
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
					Name: "bookbuyer.default",
					Hostnames: []string{
						"bookbuyer.default",
						"bookbuyer.default.svc",
						"bookbuyer.default.svc.cluster",
						"bookbuyer.default.svc.cluster.local",
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
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k8sServices := []*corev1.Service{}

			for name, namespace := range tc.services {
				svcFixture := tests.NewServiceFixture(name, namespace, map[string]string{})
				k8sServices = append(k8sServices, svcFixture)
				meshSvc := tests.NewMeshServiceFixture(name, namespace)
				mockKubeController.EXPECT().GetService(meshSvc).Return(svcFixture)
			}

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockKubeController.EXPECT().ListServices().Return(k8sServices)

			actual := mc.buildOutboundPermissiveModePolicies()
			assert.Len(actual, len(tc.expectedOutboundPolicies))
			assert.ElementsMatch(tc.expectedOutboundPolicies, actual)
		})
	}
}

func TestBuildOutboundPolicies(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
	mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)

	mc := MeshCatalog{
		kubeController:     mockKubeController,
		meshSpec:           mockMeshSpec,
		endpointsProviders: []endpoint.Provider{mockEndpointProvider},
	}
	sourceSA := identity.K8sServiceAccount{
		Name:      "bookbuyer",
		Namespace: "bookbuyer-ns",
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

	trafficSpec := spec.HTTPRouteGroup{
		TypeMeta: v1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha4",
			Kind:       "HTTPRouteGroup",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: "bookstore-ns",
			Name:      tests.RouteGroupName,
		},

		Spec: spec.HTTPRouteGroupSpec{
			Matches: []spec.HTTPMatch{
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
					Headers: map[string]string{
						"user-agent": tests.HTTPUserAgent,
					},
				},
			},
		},
	}
	mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*spec.HTTPRouteGroup{&trafficSpec}).AnyTimes()
	mockEndpointProvider.EXPECT().GetServicesForServiceAccount(destSA).Return([]service.MeshService{destMeshService}, nil).AnyTimes()
	mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
	mockKubeController.EXPECT().GetService(destMeshService).Return(destK8sService).AnyTimes()

	trafficTarget := tests.NewSMITrafficTarget(sourceSA.Name, sourceSA.Namespace, destSA.Name, destSA.Namespace)
	hostnames := []string{
		"bookstore.bookstore-ns",
		"bookstore.bookstore-ns.svc",
		"bookstore.bookstore-ns.svc.cluster",
		"bookstore.bookstore-ns.svc.cluster.local",
		"bookstore.bookstore-ns:8888",
		"bookstore.bookstore-ns.svc:8888",
		"bookstore.bookstore-ns.svc.cluster:8888",
		"bookstore.bookstore-ns.svc.cluster.local:8888",
	}
	bookstoreWeightedCluster := service.WeightedCluster{
		ClusterName: "bookstore-ns/bookstore",
		Weight:      100,
	}
	expected := []*trafficpolicy.OutboundTrafficPolicy{
		{
			Name:      destMeshService.Name + "." + destMeshService.Namespace,
			Hostnames: hostnames,
			Routes: []*trafficpolicy.RouteWeightedClusters{
				{
					HTTPRouteMatch:   tests.WildCardRouteMatch,
					WeightedClusters: mapset.NewSet(bookstoreWeightedCluster),
				},
			},
		},
	}
	actual := mc.buildOutboundPolicies(sourceSA, &trafficTarget)
	assert.ElementsMatch(expected, actual)
}

func TestListOutboundPoliciesForTrafficTargets(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name             string
		serviceAccount   identity.K8sServiceAccount
		apexMeshServices []service.MeshService
		traffictargets   []*access.TrafficTarget
		trafficspecs     []*spec.HTTPRouteGroup
		expectedOutbound []*trafficpolicy.OutboundTrafficPolicy
	}{
		{
			name:             "only traffic targets",
			serviceAccount:   tests.BookbuyerServiceAccount,
			apexMeshServices: []service.MeshService{},
			traffictargets:   []*access.TrafficTarget{&tests.TrafficTarget},
			trafficspecs:     []*spec.HTTPRouteGroup{&tests.HTTPRouteGroup},
			expectedOutbound: expectedBookbuyerOutbound,
		},
		{
			name:             "no traffic targets and no traffic splits",
			serviceAccount:   tests.BookbuyerServiceAccount,
			apexMeshServices: []service.MeshService{},
			traffictargets:   []*access.TrafficTarget{},
			trafficspecs:     []*spec.HTTPRouteGroup{},
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockKubeController := k8s.NewMockController(mockCtrl)
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)

			for _, ms := range tc.apexMeshServices {
				apexK8sService := tests.NewServiceFixture(ms.Name, ms.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(ms).Return(apexK8sService).AnyTimes()
			}
			mockMeshSpec.EXPECT().ListTrafficTargets().Return(tc.traffictargets).AnyTimes()
			mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return(tc.trafficspecs).AnyTimes()
			mockEndpointProvider.EXPECT().GetServicesForServiceAccount(tests.BookstoreServiceAccount).Return([]service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService}, nil).AnyTimes()
			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockKubeController.EXPECT().GetService(tests.BookstoreV1Service).Return(tests.NewServiceFixture(tests.BookstoreV1Service.Name, tests.BookstoreV1Service.Namespace, map[string]string{})).AnyTimes()
			mockKubeController.EXPECT().GetService(tests.BookstoreV2Service).Return(tests.NewServiceFixture(tests.BookstoreV2Service.Name, tests.BookstoreV2Service.Namespace, map[string]string{})).AnyTimes()
			mockKubeController.EXPECT().GetService(tests.BookstoreApexService).Return(tests.NewServiceFixture(tests.BookstoreApexService.Name, tests.BookstoreApexService.Namespace, map[string]string{})).AnyTimes()

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
			}

			outbound := mc.listOutboundPoliciesForTrafficTargets(tc.serviceAccount)
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

	mc := MeshCatalog{
		kubeController:     mockKubeController,
		endpointsProviders: []endpoint.Provider{mockEndpointProvider},
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

	mockEndpointProvider.EXPECT().GetServicesForServiceAccount(destSA).Return([]service.MeshService{destMeshService}, nil).AnyTimes()
	mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
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
