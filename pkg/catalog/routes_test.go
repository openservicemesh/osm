package catalog

import (
	"fmt"
	reflect "reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

func TestListTrafficPolicies(t *testing.T) {
	assert := assert.New(t)

	type listTrafficPoliciesTest struct {
		input  service.MeshService
		output []trafficpolicy.TrafficTarget
	}

	listTrafficPoliciesTests := []listTrafficPoliciesTest{
		{
			input:  tests.BookstoreService,
			output: []trafficpolicy.TrafficTarget{tests.BookstoreTrafficPolicy},
		},
		{
			input:  tests.BookbuyerService,
			output: []trafficpolicy.TrafficTarget{tests.BookstoreTrafficPolicy, tests.BookstoreApexTrafficPolicy},
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
					Name:        utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreService),
					Destination: tests.BookstoreService,
					Source:      tests.BookbuyerService,
					HTTPRoutes: []trafficpolicy.HTTPRoute{
						{
							PathRegex: tests.BookstoreBuyPath,
							Methods:   []string{"GET"},
							Headers: map[string]string{
								"user-agent": tests.HTTPUserAgent,
							},
						},
						{
							PathRegex: tests.BookstoreSellPath,
							Methods:   []string{"GET"},
							Headers: map[string]string{
								"user-agent": tests.HTTPUserAgent,
							},
						},
					},
				},
				{
					Name:        utils.GetTrafficTargetName(tests.TrafficTargetName, tests.BookbuyerService, tests.BookstoreApexService),
					Destination: tests.BookstoreApexService,
					Source:      tests.BookbuyerService,
					HTTPRoutes: []trafficpolicy.HTTPRoute{
						{
							PathRegex: tests.BookstoreBuyPath,
							Methods:   []string{"GET"},
							Headers: map[string]string{
								"user-agent": tests.HTTPUserAgent,
							},
						},
						{
							PathRegex: tests.BookstoreSellPath,
							Methods:   []string{"GET"},
							Headers: map[string]string{
								"user-agent": tests.HTTPUserAgent,
							},
						},
					},
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

	actualList, err := mc.ListAllowedInboundServices(tests.BookstoreService)
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
	destination := tests.NewServiceFixture(tests.BookstoreServiceName, tests.Namespace, selectors)
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

	expectedList := []service.MeshService{tests.BookstoreService, tests.BookstoreApexService}
	assert.ElementsMatch(actualList, expectedList)
}

func TestGetWeightedClusterForService(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()
	weightedCluster, err := mc.GetWeightedClusterForService(tests.BookstoreService)
	assert.Nil(err)

	expected := service.WeightedCluster{
		ClusterName: "default/bookstore",
		Weight:      100,
	}
	assert.Equal(weightedCluster, expected)
}

func TestGetServiceHostnames(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()
	actual, err := mc.getServiceHostnames(tests.BookstoreService)
	assert.Nil(err)

	expected := []string{
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
	assert.ElementsMatch(actual, expected)
}

func TestHostnamesTostr(t *testing.T) {
	assert := assert.New(t)
	actual := hostnamesTostr([]string{"foo", "bar", "baz"})
	expected := "foo,bar,baz"
	assert.Equal(actual, expected)
}

func TestGetDefaultWeightedClusterForService(t *testing.T) {
	assert := assert.New(t)

	actual := getDefaultWeightedClusterForService(tests.BookstoreService)
	expected := service.WeightedCluster{
		ClusterName: "default/bookstore",
		Weight:      100,
	}
	assert.Equal(actual, expected)
}

func TestGetHostnamesForService(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()

	actual, err := mc.GetHostnamesForService(tests.BookstoreService)
	assert.Nil(err)

	expected := strings.Join(
		[]string{
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
		},
		",")

	assert.Equal(actual, expected)
}

func TestBuildAllowAllTrafficPolicies(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()

	actual := mc.buildAllowAllTrafficPolicies(tests.BookstoreService)
	var actualTargetNames []string
	for _, target := range actual {
		actualTargetNames = append(actualTargetNames, target.Name)
	}

	expected := []string{
		"default/bookstore->default/bookbuyer",
		"default/bookstore->default/bookstore-apex",
		"default/bookbuyer->default/bookstore",
		"default/bookbuyer->default/bookstore-apex",
		"default/bookstore-apex->default/bookstore",
		"default/bookstore-apex->default/bookbuyer",
	}
	assert.ElementsMatch(actualTargetNames, expected)
}

func TestListTrafficTargetPermutations(t *testing.T) {
	assert := assert.New(t)

	destList := []service.MeshService{tests.BookstoreService, tests.BookstoreApexService}
	srcList := []service.MeshService{tests.BookbuyerService, tests.BookwarehouseService}
	trafficTargets := listTrafficTargetPermutations(tests.TrafficTargetName, srcList, destList)

	var actualTargetNames []string
	for _, target := range trafficTargets {
		actualTargetNames = append(actualTargetNames, target.Name)
	}

	expected := []string{
		utils.GetTrafficTargetName(tests.TrafficTargetName, srcList[0], destList[0]),
		utils.GetTrafficTargetName(tests.TrafficTargetName, srcList[0], destList[1]),
		utils.GetTrafficTargetName(tests.TrafficTargetName, srcList[1], destList[0]),
		utils.GetTrafficTargetName(tests.TrafficTargetName, srcList[1], destList[1]),
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
