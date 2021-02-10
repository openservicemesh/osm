package catalog

import (
	"fmt"
	"reflect"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	specs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

func TestIsValidTrafficTarget(t *testing.T) {
	assert := tassert.New(t)

	getTrafficTarget := func(rules []access.TrafficTargetRule) *access.TrafficTarget {
		return &access.TrafficTarget{
			TypeMeta: v1.TypeMeta{
				APIVersion: "access.smi-spec.io/v1alpha3",
				Kind:       "TrafficTarget",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "target",
				Namespace: "default",
			},
			Spec: access.TrafficTargetSpec{
				Destination: access.IdentityBindingSubject{
					Kind:      "Name",
					Name:      "dest-id",
					Namespace: "default",
				},
				Sources: []access.IdentityBindingSubject{{
					Kind:      "Name",
					Name:      "source-id",
					Namespace: "default",
				}},
				Rules: rules,
			},
		}
	}

	testCases := []struct {
		name     string
		input    *access.TrafficTarget
		expected bool
	}{
		{
			name:     "is valid",
			input:    &tests.TrafficTarget,
			expected: true,
		},
		{
			name:     "is not valid because TrafficTarget.Spec.Rules is nil",
			input:    getTrafficTarget(nil),
			expected: false,
		},
		{
			name:     "is not valid because TrafficTarget.Spec.Rules is not nil but is empty",
			input:    getTrafficTarget([]access.TrafficTargetRule{}),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing isValidTrafficTarget when input %s ", tc.name), func(t *testing.T) {
			actual := isValidTrafficTarget(tc.input)
			assert.Equal(tc.expected, actual)
		})
	}
}

func TestRoutesFromRules(t *testing.T) {
	assert := tassert.New(t)
	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	testCases := []struct {
		name           string
		rules          []access.TrafficTargetRule
		namespace      string
		expectedRoutes []trafficpolicy.HTTPRouteMatch
	}{
		{
			name: "http route group and match name exist",
			rules: []access.TrafficTargetRule{
				{
					Kind:    "HTTPRouteGroup",
					Name:    tests.RouteGroupName,
					Matches: []string{tests.BuyBooksMatchName},
				},
			},
			namespace:      tests.Namespace,
			expectedRoutes: []trafficpolicy.HTTPRouteMatch{tests.BookstoreBuyHTTPRoute},
		},
		{
			name: "http route group and match name do not exist",
			rules: []access.TrafficTargetRule{
				{
					Kind:    "HTTPRouteGroup",
					Name:    "DoesNotExist",
					Matches: []string{"hello"},
				},
			},
			namespace:      tests.Namespace,
			expectedRoutes: []trafficpolicy.HTTPRouteMatch{},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing routesFromRules where %s", tc.name), func(t *testing.T) {
			routes, err := mc.routesFromRules(tc.rules, tc.namespace)
			assert.Nil(err)
			assert.EqualValues(tc.expectedRoutes, routes)
		})
	}
}

func TestListTrafficPolicies(t *testing.T) {
	assert := tassert.New(t)

	type listTrafficPoliciesTest struct {
		input  service.MeshService
		output []trafficpolicy.TrafficTarget
	}

	listTrafficPoliciesTests := []listTrafficPoliciesTest{
		{
			input:  tests.BookstoreV1Service,
			output: []trafficpolicy.TrafficTarget{tests.BookstoreV1TrafficPolicy},
		},
		{
			input:  tests.BookbuyerService,
			output: []trafficpolicy.TrafficTarget{tests.BookstoreV1TrafficPolicy, tests.BookstoreV2TrafficPolicy, tests.BookstoreApexTrafficPolicy},
		},
	}

	mc := newFakeMeshCatalog()

	for _, test := range listTrafficPoliciesTests {
		trafficTargets, err := mc.ListTrafficPolicies(test.input)
		assert.Nil(err)
		assert.ElementsMatch(trafficTargets, test.output)
	}
}

func TestGetTrafficPoliciesForService(t *testing.T) {
	assert := tassert.New(t)

	type getTrafficPoliciesForServiceTest struct {
		input  service.MeshService
		output []trafficpolicy.TrafficTarget
	}

	getTrafficPoliciesForServiceTests := []getTrafficPoliciesForServiceTest{
		{
			input: tests.BookbuyerService,
			output: []trafficpolicy.TrafficTarget{
				{
					Name:             utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreV1Service),
					Destination:      tests.BookstoreV1Service,
					Source:           tests.BookbuyerService,
					HTTPRouteMatches: tests.BookstoreV1TrafficPolicy.HTTPRouteMatches,
				},
				{
					Name:             utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreV2Service),
					Destination:      tests.BookstoreV2Service,
					Source:           tests.BookbuyerService,
					HTTPRouteMatches: tests.BookstoreV2TrafficPolicy.HTTPRouteMatches,
				},
				{
					Name:             utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreApexService),
					Destination:      tests.BookstoreApexService,
					Source:           tests.BookbuyerService,
					HTTPRouteMatches: tests.BookstoreApexTrafficPolicy.HTTPRouteMatches,
				},
			},
		},
	}

	mc := newFakeMeshCatalog()

	for _, test := range getTrafficPoliciesForServiceTests {
		allTrafficPolicies, err := getTrafficPoliciesForService(mc, tests.RoutePolicyMap, test.input)
		assert.Nil(err)
		assert.ElementsMatch(allTrafficPolicies, test.output)
	}
}

func TestGetHTTPPathsPerRoute(t *testing.T) {
	assert := tassert.New(t)

	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}
	actual, err := mc.getHTTPPathsPerRoute()
	assert.Nil(err)

	specKey := mc.getTrafficSpecName("HTTPRouteGroup", tests.Namespace, tests.RouteGroupName)
	expected := map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch{
		specKey: {
			trafficpolicy.TrafficSpecMatchName(tests.BuyBooksMatchName): {
				PathRegex: tests.BookstoreBuyPath,
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": tests.HTTPUserAgent,
				},
			},
			trafficpolicy.TrafficSpecMatchName(tests.SellBooksMatchName): {
				PathRegex: tests.BookstoreSellPath,
				Methods:   []string{"GET"},
				Headers: map[string]string{
					"user-agent": tests.HTTPUserAgent,
				},
			},
			trafficpolicy.TrafficSpecMatchName(tests.WildcardWithHeadersMatchName): {
				PathRegex: ".*",
				Methods:   []string{"*"},
				Headers: map[string]string{
					"user-agent": tests.HTTPUserAgent,
				},
			},
		},
	}

	assert.True(reflect.DeepEqual(actual, expected))
}

func TestGetTrafficSpecName(t *testing.T) {
	assert := tassert.New(t)

	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	actual := mc.getTrafficSpecName("HTTPRouteGroup", tests.Namespace, tests.RouteGroupName)
	expected := trafficpolicy.TrafficSpecName(fmt.Sprintf("HTTPRouteGroup/%s/%s", tests.Namespace, tests.RouteGroupName))
	assert.Equal(actual, expected)
}

func TestListAllowedInboundServices(t *testing.T) {
	assert := tassert.New(t)

	mc := newFakeMeshCatalog()

	actualList, err := mc.ListAllowedInboundServices(tests.BookstoreV1Service)
	assert.Nil(err)
	expectedList := []service.MeshService{tests.BookbuyerService}
	assert.ElementsMatch(actualList, expectedList)
}

func TestBuildAllowPolicyForSourceToDest(t *testing.T) {
	assert := tassert.New(t)

	mc := newFakeMeshCatalog()

	selectors := map[string]string{
		tests.SelectorKey: tests.SelectorValue,
	}
	source := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selectors)
	expectedSourceTrafficResource := utils.K8sSvcToMeshSvc(source)
	destination := tests.NewServiceFixture(tests.BookstoreV1ServiceName, tests.Namespace, selectors)
	expectedDestinationTrafficResource := utils.K8sSvcToMeshSvc(destination)

	expectedHostHeaders := map[string]string{"user-agent": tests.HTTPUserAgent}
	expectedRoute := trafficpolicy.HTTPRouteMatch{
		PathRegex: constants.RegexMatchAll,
		Methods:   []string{constants.WildcardHTTPMethod},
		Headers:   expectedHostHeaders,
	}

	trafficTarget := mc.buildAllowPolicyForSourceToDest(source, destination)
	assert.Equal(trafficTarget.Source, expectedSourceTrafficResource)
	assert.Equal(trafficTarget.Destination, expectedDestinationTrafficResource)
	assert.Equal(trafficTarget.HTTPRouteMatches[0].PathRegex, expectedRoute.PathRegex)
	assert.ElementsMatch(trafficTarget.HTTPRouteMatches[0].Methods, expectedRoute.Methods)
}

func TestListAllowedOutboundServicesForIdentity(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name           string
		serviceAccount service.K8sServiceAccount
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
			serviceAccount: service.K8sServiceAccount{
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

func TestListMeshServices(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockKubeController := k8s.NewMockController(mockCtrl)
	mc := MeshCatalog{
		kubeController: mockKubeController,
	}

	testCases := []struct {
		name     string
		services map[string]string // name: namespace
	}{
		{
			name:     "services exist in mesh",
			services: map[string]string{"bookstore": "bookstore-ns", "bookbuyer": "bookbuyer-ns", "bookwarehouse": "bookwarehouse"},
		},
		{
			name:     "no services in mesh",
			services: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k8sServices := []*corev1.Service{}
			expectedMeshServices := []service.MeshService{}

			for name, namespace := range tc.services {
				k8sServices = append(k8sServices, tests.NewServiceFixture(name, namespace, map[string]string{}))
				expectedMeshServices = append(expectedMeshServices, tests.NewMeshServiceFixture(name, namespace))
			}

			mockKubeController.EXPECT().ListServices().Return(k8sServices)
			actual := mc.listMeshServices()
			assert.Equal(expectedMeshServices, actual)
		})
	}
}

func TestGetWeightedClusterForService(t *testing.T) {
	assert := tassert.New(t)

	mc := newFakeMeshCatalog()
	weightedCluster, err := mc.GetWeightedClusterForService(tests.BookstoreV1Service)
	assert.Nil(err)

	expected := service.WeightedCluster{
		ClusterName: "default/bookstore-v1",
		Weight:      tests.Weight90,
	}
	assert.Equal(weightedCluster, expected)
}

func TestGetServiceHostnames(t *testing.T) {
	assert := tassert.New(t)

	mc := newFakeMeshCatalog()

	testCases := []struct {
		svc           service.MeshService
		sameNamespace bool
		expected      []string
	}{
		{
			tests.BookstoreV1Service,
			true,
			[]string{
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
		},
		{
			tests.BookstoreV1Service,
			false,
			[]string{
				"bookstore-v1.default",
				"bookstore-v1.default.svc",
				"bookstore-v1.default.svc.cluster",
				"bookstore-v1.default.svc.cluster.local",
				"bookstore-v1.default:8888",
				"bookstore-v1.default.svc:8888",
				"bookstore-v1.default.svc.cluster:8888",
				"bookstore-v1.default.svc.cluster.local:8888",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing hostnames for svc %s with sameNamespace=%t", tc.svc, tc.sameNamespace), func(t *testing.T) {
			actual, err := mc.getServiceHostnames(tc.svc, tc.sameNamespace)
			assert.Nil(err)
			assert.ElementsMatch(actual, tc.expected)
		})
	}
}

func TestGetDefaultWeightedClusterForService(t *testing.T) {
	assert := tassert.New(t)

	actual := getDefaultWeightedClusterForService(tests.BookstoreV1Service)
	expected := service.WeightedCluster{
		ClusterName: "default/bookstore-v1",
		Weight:      100,
	}
	assert.Equal(actual, expected)
}

// TODO : remove as a part of routes refactor (#2397)
func TestGetResolvableHostnamesForUpstreamService(t *testing.T) {
	assert := tassert.New(t)

	mc := newFakeMeshCatalog()

	testCases := []struct {
		downstream        service.MeshService
		expectedHostnames []string
	}{
		{
			downstream: service.MeshService{
				Namespace: "default",
				Name:      "foo",
			},
			expectedHostnames: []string{
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
		},
		{
			downstream: service.MeshService{
				Namespace: "bar",
				Name:      "foo",
			},
			expectedHostnames: []string{
				"bookstore-apex.default",
				"bookstore-apex.default.svc",
				"bookstore-apex.default.svc.cluster",
				"bookstore-apex.default.svc.cluster.local",
				"bookstore-apex.default:8888",
				"bookstore-apex.default.svc:8888",
				"bookstore-apex.default.svc.cluster:8888",
				"bookstore-apex.default.svc.cluster.local:8888",
				"bookstore-v1.default",
				"bookstore-v1.default.svc",
				"bookstore-v1.default.svc.cluster",
				"bookstore-v1.default.svc.cluster.local",
				"bookstore-v1.default:8888",
				"bookstore-v1.default.svc:8888",
				"bookstore-v1.default.svc.cluster:8888",
				"bookstore-v1.default.svc.cluster.local:8888",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing hostnames when %s svc reaches %s svc", tc.downstream, tests.BookstoreV1Service), func(t *testing.T) {
			actual, err := mc.GetResolvableHostnamesForUpstreamService(tc.downstream, tests.BookstoreV1Service)
			assert.Nil(err)
			assert.Equal(actual, tc.expectedHostnames)
		})
	}
}

func TestBuildAllowAllTrafficPolicies(t *testing.T) {
	assert := tassert.New(t)

	mc := newFakeMeshCatalog()

	actual := mc.buildAllowAllTrafficPolicies(tests.BookstoreV1Service)
	var actualTargetNames []string
	for _, target := range actual {
		actualTargetNames = append(actualTargetNames, target.Name)
	}

	expected := []string{
		"default/bookstore-v1->default/bookbuyer",
		"default/bookstore-v1->default/bookstore-apex",
		"default/bookstore-v2->default/bookbuyer",
		"default/bookstore-v2->default/bookstore-apex",
		"default/bookbuyer->default/bookstore-v1",
		"default/bookbuyer->default/bookstore-apex",
		"default/bookstore-apex->default/bookstore-v1",
		"default/bookbuyer->default/bookstore-v2",
		"default/bookstore-apex->default/bookstore-v2",
		"default/bookstore-apex->default/bookbuyer",
		"default/bookstore-v1->default/bookstore-v2",
		"default/bookstore-v2->default/bookstore-v1",
	}
	assert.ElementsMatch(actualTargetNames, expected)
}

func TestListTrafficTargetPermutations(t *testing.T) {
	assert := tassert.New(t)

	mc := newFakeMeshCatalog()

	trafficTargets, err := mc.listTrafficTargetPermutations(tests.TrafficTarget, tests.TrafficTarget.Spec.Sources[0], tests.TrafficTarget.Spec.Destination)
	assert.Nil(err)

	var actualTargetNames []string
	for _, target := range trafficTargets {
		actualTargetNames = append(actualTargetNames, target.Name)
	}

	expected := []string{
		utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreV1Service),
		utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreV2Service),
		utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreApexService),
	}
	assert.ElementsMatch(actualTargetNames, expected)
}

func TestHashSrcDstService(t *testing.T) {
	assert := tassert.New(t)

	src := service.MeshService{
		Namespace: "src-ns",
		Name:      "source",
	}
	dst := service.MeshService{
		Namespace: "dst-ns",
		Name:      "destination",
	}

	srcDstServiceHash := hashSrcDstService(src, dst)
	assert.Equal(srcDstServiceHash, "src-ns/source:dst-ns/destination")
}

func TestGetTrafficTargetFromSrcDstHash(t *testing.T) {
	assert := tassert.New(t)

	src := service.MeshService{
		Namespace: "src-ns",
		Name:      "source",
	}
	dst := service.MeshService{
		Namespace: "dst-ns",
		Name:      "destination",
	}
	srcDstServiceHash := "src-ns/source:dst-ns/destination"

	targetName := "test"
	httpRoutes := []trafficpolicy.HTTPRouteMatch{
		{
			PathRegex: tests.BookstoreBuyPath,
			Methods:   []string{"GET"},
			Headers: map[string]string{
				"user-agent": tests.HTTPUserAgent,
			},
		},
	}

	trafficTarget := getTrafficTargetFromSrcDstHash(srcDstServiceHash, targetName, httpRoutes)

	expectedTrafficTarget := trafficpolicy.TrafficTarget{
		Source:           src,
		Destination:      dst,
		Name:             targetName,
		HTTPRouteMatches: httpRoutes,
	}

	assert.Equal(trafficTarget, expectedTrafficTarget)
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
	sourceSA := service.K8sServiceAccount{
		Name:      "bookbuyer",
		Namespace: "bookbuyer-ns",
	}
	destSA := service.K8sServiceAccount{
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
	mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*specs.HTTPRouteGroup{&trafficSpec}).AnyTimes()
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
					HTTPRouteMatch:   wildCardRouteMatch,
					WeightedClusters: mapset.NewSet(bookstoreWeightedCluster),
				},
			},
		},
	}
	actual := mc.buildOutboundPolicies(sourceSA, &trafficTarget)
	assert.ElementsMatch(expected, actual)
}

func TestBuildInboundPoliciesDiffNamespaces(t *testing.T) {
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

	sourceSA := service.K8sServiceAccount{
		Name:      "bookbuyer",
		Namespace: "bookbuyer-ns",
	}
	destSA := service.K8sServiceAccount{
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

	mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*specs.HTTPRouteGroup{&trafficSpec}).AnyTimes()
	mockEndpointProvider.EXPECT().GetServicesForServiceAccount(destSA).Return([]service.MeshService{destMeshService}, nil).AnyTimes()
	mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
	mockKubeController.EXPECT().GetService(destMeshService).Return(destK8sService).AnyTimes()

	trafficTarget := tests.NewSMITrafficTarget(sourceSA.Name, sourceSA.Namespace, destSA.Name, destSA.Namespace)
	expectedHostnames := []string{
		"bookstore",
		"bookstore.bookstore-ns",
		"bookstore.bookstore-ns.svc",
		"bookstore.bookstore-ns.svc.cluster",
		"bookstore.bookstore-ns.svc.cluster.local",
		"bookstore:8888",
		"bookstore.bookstore-ns:8888",
		"bookstore.bookstore-ns.svc:8888",
		"bookstore.bookstore-ns.svc.cluster:8888",
		"bookstore.bookstore-ns.svc.cluster.local:8888",
	}
	bookstoreWeightedCluster := service.WeightedCluster{
		ClusterName: "bookstore-ns/bookstore",
		Weight:      100,
	}
	expectedPolicies := []*trafficpolicy.InboundTrafficPolicy{
		{
			Name:      "bookstore.bookstore-ns",
			Hostnames: expectedHostnames,
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
						WeightedClusters: mapset.NewSet(bookstoreWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(sourceSA),
				},
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
						WeightedClusters: mapset.NewSet(bookstoreWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(sourceSA),
				},
			},
		},
	}
	actual := mc.buildInboundPolicies(&trafficTarget)
	assert.ElementsMatch(expectedPolicies, actual)
}

func TestBuildInboundPoliciesSameNamespace(t *testing.T) {
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

	sourceSA := service.K8sServiceAccount{
		Name:      "bookbuyer",
		Namespace: "default",
	}
	destSA := service.K8sServiceAccount{
		Name:      "bookstore",
		Namespace: "default",
	}

	destMeshService := service.MeshService{
		Name:      "bookstore",
		Namespace: "default",
	}

	destK8sService := tests.NewServiceFixture(destMeshService.Name, destMeshService.Namespace, map[string]string{})

	trafficSpec := spec.HTTPRouteGroup{
		TypeMeta: v1.TypeMeta{
			APIVersion: "specs.smi-spec.io/v1alpha4",
			Kind:       "HTTPRouteGroup",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: "default",
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

	mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*specs.HTTPRouteGroup{&trafficSpec}).AnyTimes()
	mockEndpointProvider.EXPECT().GetServicesForServiceAccount(destSA).Return([]service.MeshService{destMeshService}, nil).AnyTimes()
	mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
	mockKubeController.EXPECT().GetService(destMeshService).Return(destK8sService).AnyTimes()

	trafficTarget := tests.NewSMITrafficTarget(sourceSA.Name, sourceSA.Namespace, destSA.Name, destSA.Namespace)
	expectedHostnames := []string{
		"bookstore",
		"bookstore.default",
		"bookstore.default.svc",
		"bookstore.default.svc.cluster",
		"bookstore.default.svc.cluster.local",
		"bookstore:8888",
		"bookstore.default:8888",
		"bookstore.default.svc:8888",
		"bookstore.default.svc.cluster:8888",
		"bookstore.default.svc.cluster.local:8888",
	}
	bookstoreWeightedCluster := service.WeightedCluster{
		ClusterName: "default/bookstore",
		Weight:      100,
	}
	expectedPolicies := []*trafficpolicy.InboundTrafficPolicy{
		{
			Name:      "bookstore.default",
			Hostnames: expectedHostnames,
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
						WeightedClusters: mapset.NewSet(bookstoreWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(sourceSA),
				},
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
						WeightedClusters: mapset.NewSet(bookstoreWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(sourceSA),
				},
			},
		},
	}
	actual := mc.buildInboundPolicies(&trafficTarget)
	assert.ElementsMatch(expectedPolicies, actual)
}

func TestBuildInboundPermissiveModePolicies(t *testing.T) {
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
		name                    string
		expectedInboundPolicies []*trafficpolicy.InboundTrafficPolicy
		meshService             service.MeshService
		serviceAccounts         map[string]string
	}{
		{
			name: "inbound traffic policies for permissive mode",
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore.bookstore-ns",
					Hostnames: []string{
						"bookstore",
						"bookstore.bookstore-ns",
						"bookstore.bookstore-ns.svc",
						"bookstore.bookstore-ns.svc.cluster",
						"bookstore.bookstore-ns.svc.cluster.local",
						"bookstore:8888",
						"bookstore.bookstore-ns:8888",
						"bookstore.bookstore-ns.svc:8888",
						"bookstore.bookstore-ns.svc.cluster:8888",
						"bookstore.bookstore-ns.svc.cluster.local:8888",
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: wildCardRouteMatch,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "bookstore-ns/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(service.K8sServiceAccount{
								Name:      "bookstore",
								Namespace: "bookstore-ns",
							}, service.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "bookbuyer-ns",
							}),
						},
					},
				},
			},
			meshService: service.MeshService{
				Name:      "bookstore",
				Namespace: "bookstore-ns",
			},
			serviceAccounts: map[string]string{"bookstore": "bookstore-ns", "bookbuyer": "bookbuyer-ns"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k8sService := tests.NewServiceFixture(tc.meshService.Name, tc.meshService.Namespace, map[string]string{})
			k8sServiceAccounts := []*corev1.ServiceAccount{}

			for name, namespace := range tc.serviceAccounts {
				k8sServiceAccounts = append(k8sServiceAccounts, tests.NewServiceAccountFixture(name, namespace))
			}

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockKubeController.EXPECT().GetService(tc.meshService).Return(k8sService)
			mockKubeController.EXPECT().ListServiceAccounts().Return(k8sServiceAccounts)
			actual := mc.buildInboundPermissiveModePolicies(tc.meshService)
			assert.Len(actual, len(tc.expectedInboundPolicies))
			assert.ElementsMatch(tc.expectedInboundPolicies, actual)
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
		srcServices              []service.MeshService
		services                 map[string]string
		expectedOutboundPolicies []*trafficpolicy.OutboundTrafficPolicy
	}{
		{
			name:        "outbound traffic policies for permissive mode",
			srcServices: []service.MeshService{tests.BookbuyerService},
			services:    map[string]string{"bookstore-v1": "default", "bookstore-apex": "default", "bookbuyer": "default"},
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
							HTTPRouteMatch:   wildCardRouteMatch,
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
							HTTPRouteMatch:   wildCardRouteMatch,
							WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
						},
					},
				},
			},
		},
		{
			name:        "outbound traffic policies for permissive mode with no service on proxy",
			srcServices: nil,
			services:    map[string]string{"bookstore-v1": "default", "bookstore-apex": "default", "bookbuyer": "default"},
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
							HTTPRouteMatch:   wildCardRouteMatch,
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
							HTTPRouteMatch:   wildCardRouteMatch,
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
							HTTPRouteMatch:   wildCardRouteMatch,
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
				if len(tc.srcServices) > 0 {
					for _, srcService := range tc.srcServices {
						if !reflect.DeepEqual(meshSvc, srcService) {
							mockKubeController.EXPECT().GetService(meshSvc).Return(svcFixture)
						}
					}
				} else {
					mockKubeController.EXPECT().GetService(meshSvc).Return(svcFixture)
				}
			}

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockKubeController.EXPECT().ListServices().Return(k8sServices)

			actual := mc.buildOutboundPermissiveModePolicies(tc.srcServices)
			assert.Len(actual, len(tc.expectedOutboundPolicies))
			assert.ElementsMatch(tc.expectedOutboundPolicies, actual)
		})
	}
}

func TestDifference(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name             string
		srcServices      []service.MeshService
		destServices     []service.MeshService
		expectedServices []service.MeshService
	}{
		{
			name:             "source services is a subset of destination services",
			srcServices:      []service.MeshService{tests.BookstoreApexService, tests.BookstoreV1Service},
			destServices:     []service.MeshService{tests.BookbuyerService, tests.BookstoreApexService, tests.BookstoreV1Service, tests.BookstoreV2Service},
			expectedServices: []service.MeshService{tests.BookbuyerService, tests.BookstoreV2Service},
		},
		{
			name:             "source services is empty",
			srcServices:      []service.MeshService{},
			destServices:     []service.MeshService{tests.BookbuyerService, tests.BookstoreApexService, tests.BookstoreV1Service, tests.BookstoreV2Service},
			expectedServices: []service.MeshService{tests.BookbuyerService, tests.BookstoreApexService, tests.BookstoreV1Service, tests.BookstoreV2Service},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := difference(tc.destServices, tc.srcServices)
			assert.ElementsMatch(tc.expectedServices, actual)
		})
	}
}

func TestListPoliciesFromTrafficTargets(t *testing.T) {
	assert := tassert.New(t)

	expectedBookbuyerOutbound := []*trafficpolicy.OutboundTrafficPolicy{
		{
			Name:      "bookstore-v1",
			Hostnames: tests.BookstoreV1Hostnames,
			Routes: []*trafficpolicy.RouteWeightedClusters{
				{
					HTTPRouteMatch:   wildCardRouteMatch,
					WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
				},
			},
		},
		{
			Name:      "bookstore-v2",
			Hostnames: tests.BookstoreV2Hostnames,
			Routes: []*trafficpolicy.RouteWeightedClusters{
				{
					HTTPRouteMatch:   wildCardRouteMatch,
					WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
				},
			},
		},
		{
			Name:      "bookstore-apex",
			Hostnames: tests.BookstoreApexHostnames,
			Routes: []*trafficpolicy.RouteWeightedClusters{
				{
					HTTPRouteMatch:   wildCardRouteMatch,
					WeightedClusters: mapset.NewSet(tests.BookstoreApexDefaultWeightedCluster),
				},
			},
		},
	}

	expectedBookstoreInbound := []*trafficpolicy.InboundTrafficPolicy{
		{
			Name:      "bookstore-v1.default",
			Hostnames: tests.BookstoreV1Hostnames,
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
						WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(tests.BookbuyerServiceAccount),
				},
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
						WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(tests.BookbuyerServiceAccount),
				},
			},
		},
		{
			Name:      "bookstore-v2.default",
			Hostnames: tests.BookstoreV2Hostnames,
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
						WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(tests.BookbuyerServiceAccount),
				},
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
						WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(tests.BookbuyerServiceAccount),
				},
			},
		},
		{
			Name:      "bookstore-apex.default",
			Hostnames: tests.BookstoreApexHostnames,
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
						WeightedClusters: mapset.NewSet(tests.BookstoreApexDefaultWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(tests.BookbuyerServiceAccount),
				},
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
						WeightedClusters: mapset.NewSet(tests.BookstoreApexDefaultWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(tests.BookbuyerServiceAccount),
				},
			},
		},
	}

	testCases := []struct {
		name             string
		serviceAccount   service.K8sServiceAccount
		expectedInbound  []*trafficpolicy.InboundTrafficPolicy
		expectedOutbound []*trafficpolicy.OutboundTrafficPolicy
		expectedErr      bool
	}{
		{
			name:             "outbound policies",
			serviceAccount:   tests.BookbuyerServiceAccount,
			expectedInbound:  []*trafficpolicy.InboundTrafficPolicy{},
			expectedOutbound: expectedBookbuyerOutbound,
			expectedErr:      false,
		},
		{
			name:             "inbound policies",
			serviceAccount:   tests.BookstoreServiceAccount,
			expectedInbound:  expectedBookstoreInbound,
			expectedOutbound: []*trafficpolicy.OutboundTrafficPolicy{},
			expectedErr:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mc := newFakeMeshCatalogForRoutes(t, testParams{})
			inbound, outbound, errs := mc.listPoliciesFromTrafficTargets(tc.serviceAccount)
			assert.ElementsMatch(tc.expectedInbound, inbound)
			assert.ElementsMatch(tc.expectedOutbound, outbound)
			if tc.expectedErr {
				assert.NotNil(errs)
			} else {
				assert.Nil(errs)
			}
		})
	}
}

func TestListPoliciesForPermissiveMode(t *testing.T) {
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

	expectedBookbuyerOutbound := []*trafficpolicy.OutboundTrafficPolicy{
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
					HTTPRouteMatch:   wildCardRouteMatch,
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
					HTTPRouteMatch:   wildCardRouteMatch,
					WeightedClusters: mapset.NewSet(tests.BookstoreV2DefaultWeightedCluster),
				},
			},
		},
	}

	expectedBookbuyerInbound := []*trafficpolicy.InboundTrafficPolicy{
		{
			Name: "bookbuyer.default",
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
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   wildCardRouteMatch,
						WeightedClusters: mapset.NewSet(tests.BookbuyerDefaultWeightedCluster),
					},
					AllowedServiceAccounts: mapset.NewSet(tests.BookbuyerServiceAccount, tests.BookstoreServiceAccount),
				},
			},
		},
	}

	bookbuyerK8sService := tests.NewServiceFixture(tests.BookbuyerService.Name, tests.BookbuyerService.Namespace, map[string]string{})
	bookstorev1K8sService := tests.NewServiceFixture(tests.BookstoreV1Service.Name, tests.BookstoreV1Service.Namespace, map[string]string{})
	bookstorev2K8sService := tests.NewServiceFixture(tests.BookstoreV2Service.Name, tests.BookstoreV2Service.Namespace, map[string]string{})

	services := []*corev1.Service{}
	services = append(services, bookbuyerK8sService)
	services = append(services, bookstorev2K8sService)
	services = append(services, bookstorev1K8sService)

	serviceAccounts := []*corev1.ServiceAccount{}
	serviceAccounts = append(serviceAccounts, tests.NewServiceAccountFixture(tests.BookbuyerServiceAccountName, tests.Namespace))
	serviceAccounts = append(serviceAccounts, tests.NewServiceAccountFixture(tests.BookstoreServiceAccountName, tests.Namespace))

	mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
	mockKubeController.EXPECT().GetService(tests.BookstoreV1Service).Return(bookstorev1K8sService).AnyTimes()
	mockKubeController.EXPECT().GetService(tests.BookstoreV2Service).Return(bookstorev2K8sService).AnyTimes()
	mockKubeController.EXPECT().GetService(tests.BookbuyerService).Return(bookbuyerK8sService).AnyTimes()
	mockKubeController.EXPECT().ListServices().Return(services).AnyTimes()
	mockKubeController.EXPECT().ListServiceAccounts().Return(serviceAccounts).AnyTimes()

	inbound, outbound, err := mc.ListPoliciesForPermissiveMode([]service.MeshService{tests.BookbuyerService})
	assert.Nil(err)
	assert.ElementsMatch(expectedBookbuyerInbound, inbound)
	assert.ElementsMatch(expectedBookbuyerOutbound, outbound)
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

	destSA := service.K8sServiceAccount{
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

func TestBuildPolicyName(t *testing.T) {
	assert := tassert.New(t)

	svc := service.MeshService{
		Namespace: "default",
		Name:      "foo",
	}

	testCases := []struct {
		name          string
		svc           service.MeshService
		sameNamespace bool
		expectedName  string
	}{
		{
			name:          "same namespace",
			svc:           svc,
			sameNamespace: true,
			expectedName:  "foo",
		},
		{
			name:          "different namespace",
			svc:           svc,
			sameNamespace: false,
			expectedName:  "foo.default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := buildPolicyName(tc.svc, tc.sameNamespace)
			assert.Equal(tc.expectedName, actual)
		})
	}
}
