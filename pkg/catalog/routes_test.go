package catalog

import (
	"fmt"
	reflect "reflect"
	"testing"

	mapset "github.com/deckarep/golang-set"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

var (
	testGetAllRoute = trafficpolicy.HTTPRoute{
		PathRegex: "/all",
		Methods:   []string{"GET"},
		Headers: map[string]string{
			"user-agent": "some-agent",
		},
	}

	testGetSomeRoute = trafficpolicy.HTTPRoute{
		PathRegex: "/some",
		Methods:   []string{"GET"},
		Headers: map[string]string{
			"user-agent": "another-agent",
		},
	}
)

func TestIsValidTrafficTarget(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name     string
		input    *target.TrafficTarget
		expected bool
	}{
		{
			name:     "is valid",
			input:    &tests.TrafficTarget,
			expected: true,
		},
		{
			name: "is not valid",
			input: &target.TrafficTarget{
				TypeMeta: v1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha2",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: v1.ObjectMeta{
					Name:      "target",
					Namespace: "default",
				},
				Spec: target.TrafficTargetSpec{
					Destination: target.IdentityBindingSubject{
						Kind:      "Name",
						Name:      "dest-id",
						Namespace: "default",
					},
					Sources: []target.IdentityBindingSubject{{
						Kind:      "Name",
						Name:      "source-id",
						Namespace: "default",
					}},
				},
			},
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

func TestBuildTrafficPolicies(t *testing.T) {
	mc := newFakeMeshCatalogForRoutes(t)

	testCases := []struct {
		name             string
		sourceServices   []service.MeshService
		destServices     []service.MeshService
		routes           []trafficpolicy.HTTPRoute
		expectedPolicies []*trafficpolicy.TrafficPolicy
	}{
		{
			name:           "policy for 1 source service and 1 destination service with 1 route",
			sourceServices: []service.MeshService{tests.BookbuyerService},
			destServices:   []service.MeshService{tests.BookstoreV2Service},
			routes:         []trafficpolicy.HTTPRoute{testGetAllRoute},
			expectedPolicies: []*trafficpolicy.TrafficPolicy{
				{
					Name:        "bookstore-v2-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV2Service,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						{
							HTTPRoute: testGetAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
					},
					Hostnames: tests.BookstoreV2Hostnames,
				},
			},
		},
		{
			name:           "policy for 1 source service and 2 destination services with one destination service that doesn't exist",
			sourceServices: []service.MeshService{tests.BookbuyerService},
			destServices: []service.MeshService{tests.BookstoreV2Service, {
				Namespace: "default",
				Name:      "nonexistentservices",
			}},
			routes: []trafficpolicy.HTTPRoute{testGetAllRoute},
			expectedPolicies: []*trafficpolicy.TrafficPolicy{
				{
					Name:        "bookstore-v2-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV2Service,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						{
							HTTPRoute: testGetAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
					},
					Hostnames: tests.BookstoreV2Hostnames,
				},
			},
		},
		{
			name:           "policies for 1 source service, 2 destination services, and multiple routes ",
			sourceServices: []service.MeshService{tests.BookbuyerService},
			destServices:   []service.MeshService{tests.BookstoreV2Service, tests.BookstoreV1Service},
			routes:         []trafficpolicy.HTTPRoute{testGetAllRoute, testGetSomeRoute},
			expectedPolicies: []*trafficpolicy.TrafficPolicy{
				{
					Name:        "bookstore-v2-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV2Service,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						{
							HTTPRoute: testGetAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
						{
							HTTPRoute: testGetSomeRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
					},
					Hostnames: tests.BookstoreV2Hostnames,
				},
				{
					Name:        "bookstore-v1-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV1Service,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						{
							HTTPRoute: testGetAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v1",
								Weight:      100,
							}),
						},
						{
							HTTPRoute: testGetSomeRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v1",
								Weight:      100,
							}),
						},
					},
					Hostnames: tests.BookstoreV1Hostnames,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing build traffic policies when %s ", tc.name), func(t *testing.T) {
			policies := mc.buildTrafficPolicies(tc.sourceServices, tc.destServices, tc.routes)
			assert.ElementsMatch(t, tc.expectedPolicies, policies, tc.name)
		})
	}
}

func TestGetHostnamesForUpstreamService(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalogForRoutes(t)

	testCases := []struct {
		name              string
		upstream          service.MeshService
		downstream        service.MeshService
		expectedHostnames []string
		expectedErr       bool
	}{
		{
			name:     "When upstream and downstream are  in same namespace",
			upstream: tests.BookstoreV1Service,
			downstream: service.MeshService{
				Namespace: "default",
				Name:      "foo",
			},
			expectedHostnames: tests.BookstoreV1Hostnames,
			expectedErr:       false,
		},
		{
			name:     "When upstream and downstream are not in same namespace",
			upstream: tests.BookstoreV1Service,
			downstream: service.MeshService{
				Namespace: "bar",
				Name:      "foo",
			},
			expectedHostnames: []string{
				"bookstore-v1.default",
				"bookstore-v1.default.svc",
				"bookstore-v1.default.svc.cluster",
				"bookstore-v1.default.svc.cluster.local",
				"bookstore-v1.default:8888",
				"bookstore-v1.default.svc:8888",
				"bookstore-v1.default.svc.cluster:8888",
				"bookstore-v1.default.svc.cluster.local:8888",
			},
			expectedErr: false,
		},
		{
			name: "When upstream service does not exist",
			upstream: service.MeshService{
				Namespace: "bar",
				Name:      "DoesNotExist",
			},
			downstream: service.MeshService{
				Namespace: "bar",
				Name:      "foo",
			},
			expectedHostnames: nil,
			expectedErr:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing hostnames when %s svc reaches %s svc", tc.downstream, tc.upstream), func(t *testing.T) {
			actual, err := mc.GetHostnamesForUpstreamService(tc.downstream, tc.upstream)
			if tc.expectedErr == false {
				assert.Nil(err)
			} else {
				assert.NotNil(err)
			}
			assert.Equal(actual, tc.expectedHostnames, tc.name)
		})
	}
}

func TestGetServicesForServiceAccounts(t *testing.T) {
	assert := assert.New(t)
	mc := newFakeMeshCatalog()

	testCases := []struct {
		name     string
		input    []service.K8sServiceAccount
		expected []service.MeshService
	}{
		{
			name:     "multiple service accounts and services",
			input:    []service.K8sServiceAccount{tests.BookstoreServiceAccount, tests.BookbuyerServiceAccount},
			expected: []service.MeshService{tests.BookbuyerService, tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService},
		},
		{
			name:     "single service account and service",
			input:    []service.K8sServiceAccount{tests.BookbuyerServiceAccount},
			expected: []service.MeshService{tests.BookbuyerService},
		},
		{
			name: "service account does not exist",
			input: []service.K8sServiceAccount{{
				Name:      "DoesNotExist",
				Namespace: "default",
			}},
			expected: []service.MeshService{},
		},
		{
			name:     "duplicate service accounts and services",
			input:    []service.K8sServiceAccount{tests.BookbuyerServiceAccount, tests.BookbuyerServiceAccount},
			expected: []service.MeshService{tests.BookbuyerService},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing GetServicesForServiceAccounts where %s", tc.name), func(t *testing.T) {
			actual := mc.GetServicesForServiceAccounts(tc.input)
			assert.ElementsMatch(tc.expected, actual)
		})
	}
}

func TestRoutesFromRules(t *testing.T) {
	assert := assert.New(t)
	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	testCases := []struct {
		name           string
		rules          []target.TrafficTargetRule
		namespace      string
		expectedRoutes []trafficpolicy.HTTPRoute
	}{
		{
			name: "http route group and match name exist",
			rules: []target.TrafficTargetRule{
				{
					Kind:    "HTTPRouteGroup",
					Name:    tests.RouteGroupName,
					Matches: []string{tests.BuyBooksMatchName},
				},
			},
			namespace:      tests.Namespace,
			expectedRoutes: []trafficpolicy.HTTPRoute{tests.BookstoreBuyHTTPRoute},
		},
		{
			name: "http route group and match name do not exist",
			rules: []target.TrafficTargetRule{
				{
					Kind:    "HTTPRouteGroup",
					Name:    "DoesNotExist",
					Matches: []string{"hello"},
				},
			},
			namespace:      tests.Namespace,
			expectedRoutes: []trafficpolicy.HTTPRoute{},
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
	assert := assert.New(t)

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
	assert := assert.New(t)

	type getTrafficPoliciesForServiceTest struct {
		input  service.MeshService
		output []trafficpolicy.TrafficTarget
	}

	getTrafficPoliciesForServiceTests := []getTrafficPoliciesForServiceTest{
		{
			input: tests.BookbuyerService,
			output: []trafficpolicy.TrafficTarget{
				{
					Name:        utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreV1Service),
					Destination: tests.BookstoreV1Service,
					Source:      tests.BookbuyerService,
					HTTPRoutes:  tests.BookstoreV1TrafficPolicy.HTTPRoutes,
				},
				{
					Name:        utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreV2Service),
					Destination: tests.BookstoreV2Service,
					Source:      tests.BookbuyerService,
					HTTPRoutes:  tests.BookstoreV2TrafficPolicy.HTTPRoutes,
				},
				{
					Name:        utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreApexService),
					Destination: tests.BookstoreApexService,
					Source:      tests.BookbuyerService,
					HTTPRoutes:  tests.BookstoreApexTrafficPolicy.HTTPRoutes,
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
	assert := assert.New(t)

	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}
	actual, err := mc.getHTTPPathsPerRoute()
	assert.Nil(err)

	specKey := mc.getTrafficSpecName("HTTPRouteGroup", tests.Namespace, tests.RouteGroupName)
	expected := map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRoute{
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
	assert := assert.New(t)

	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	actual := mc.getTrafficSpecName("HTTPRouteGroup", tests.Namespace, tests.RouteGroupName)
	expected := trafficpolicy.TrafficSpecName(fmt.Sprintf("HTTPRouteGroup/%s/%s", tests.Namespace, tests.RouteGroupName))
	assert.Equal(actual, expected)
}

func TestListAllowedInboundServices(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()

	actualList, err := mc.ListAllowedInboundServices(tests.BookstoreV1Service)
	assert.Nil(err)
	expectedList := []service.MeshService{tests.BookbuyerService}
	assert.ElementsMatch(actualList, expectedList)
}

func TestBuildAllowPolicyForSourceToDest(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()

	selectors := map[string]string{
		tests.SelectorKey: tests.SelectorValue,
	}
	source := tests.NewServiceFixture(tests.BookbuyerServiceName, tests.Namespace, selectors)
	expectedSourceTrafficResource := utils.K8sSvcToMeshSvc(source)
	destination := tests.NewServiceFixture(tests.BookstoreV1ServiceName, tests.Namespace, selectors)
	expectedDestinationTrafficResource := utils.K8sSvcToMeshSvc(destination)

	expectedHostHeaders := map[string]string{"user-agent": tests.HTTPUserAgent}
	expectedRoute := trafficpolicy.HTTPRoute{
		PathRegex: constants.RegexMatchAll,
		Methods:   []string{constants.WildcardHTTPMethod},
		Headers:   expectedHostHeaders,
	}

	trafficTarget := mc.buildAllowPolicyForSourceToDest(source, destination)
	assert.Equal(trafficTarget.Source, expectedSourceTrafficResource)
	assert.Equal(trafficTarget.Destination, expectedDestinationTrafficResource)
	assert.Equal(trafficTarget.HTTPRoutes[0].PathRegex, expectedRoute.PathRegex)
	assert.ElementsMatch(trafficTarget.HTTPRoutes[0].Methods, expectedRoute.Methods)
}

func TestListAllowedOutboundServices(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()
	actualList, err := mc.ListAllowedOutboundServices(tests.BookbuyerService)
	assert.Nil(err)

	expectedList := []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService}
	assert.ElementsMatch(actualList, expectedList)
}

func TestGetWeightedClusterForService(t *testing.T) {
	assert := assert.New(t)

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
	assert := assert.New(t)

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

func TestHostnamesTostr(t *testing.T) {
	assert := assert.New(t)
	actual := hostnamesTostr([]string{"foo", "bar", "baz"})
	expected := "foo,bar,baz"
	assert.Equal(actual, expected)
}

func TestGetDefaultWeightedClusterForService(t *testing.T) {
	assert := assert.New(t)

	actual := getDefaultWeightedClusterForService(tests.BookstoreV1Service)
	expected := service.WeightedCluster{
		ClusterName: "default/bookstore-v1",
		Weight:      100,
	}
	assert.Equal(actual, expected)
}

func TestGetResolvableHostnamesForUpstreamService(t *testing.T) {
	assert := assert.New(t)

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
	assert := assert.New(t)

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
	assert := assert.New(t)

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
	assert := assert.New(t)

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
	assert := assert.New(t)

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
	httpRoutes := []trafficpolicy.HTTPRoute{
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
		Source:      src,
		Destination: dst,
		Name:        targetName,
		HTTPRoutes:  httpRoutes,
	}

	assert.Equal(trafficTarget, expectedTrafficTarget)
}
