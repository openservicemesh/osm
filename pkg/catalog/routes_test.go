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
			input:  tests.BookstoreV1Service,
			output: []trafficpolicy.TrafficTarget{tests.BookstoreV1TrafficPolicy},
		},
		{
			input:  tests.BookbuyerService,
			output: []trafficpolicy.TrafficTarget{tests.BookstoreV1TrafficPolicy, tests.BookstoreV2TrafficPolicy, tests.BookstoreV3TrafficPolicy},
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
		output map[string]trafficpolicy.TrafficTarget
	}

	getTrafficPoliciesForServiceTests := []getTrafficPoliciesForServiceTest{
		{
			input: tests.BookbuyerService,
			output: map[string]trafficpolicy.TrafficTarget{
				hashSrcDstService(tests.BookbuyerService, tests.BookstoreV1Service): {
					Name:        utils.GetTrafficTargetName(tests.BookstoreTrafficTargetName, tests.BookbuyerService, tests.BookstoreV1Service),
					Destination: tests.BookstoreV1Service,
					Source:      tests.BookbuyerService,
					HTTPRoutes:  tests.BookstoreV1TrafficPolicy.HTTPRoutes,
				},
				hashSrcDstService(tests.BookbuyerService, tests.BookstoreV2Service): {
					Name:        utils.GetTrafficTargetName(tests.BookstoreV2TrafficTargetName, tests.BookbuyerService, tests.BookstoreV2Service),
					Destination: tests.BookstoreV2Service,
					Source:      tests.BookbuyerService,
					HTTPRoutes:  tests.BookstoreV2TrafficPolicy.HTTPRoutes,
				},
				hashSrcDstService(tests.BookbuyerService, tests.BookstoreV3Service): {
					Name:        utils.GetTrafficTargetName(tests.BookstoreTrafficTargetName, tests.BookbuyerService, tests.BookstoreV3Service),
					Destination: tests.BookstoreV3Service,
					Source:      tests.BookbuyerService,
					HTTPRoutes:  tests.BookstoreV3TrafficPolicy.HTTPRoutes,
				},
			},
		},
	}

	mc := newFakeMeshCatalog()

	for _, test := range getTrafficPoliciesForServiceTests {
		allTrafficPolicies, err := getTrafficPoliciesForService(mc, tests.RoutePolicyMap, test.input)
		assert.Nil(err)
		assert.True(reflect.DeepEqual(allTrafficPolicies, test.output))
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

	expectedList := []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreV3Service}
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
	actual, err := mc.getServiceHostnames(tests.BookstoreV1Service)
	assert.Nil(err)

	expected := []string{
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

	actual := getDefaultWeightedClusterForService(tests.BookstoreV1Service)
	expected := service.WeightedCluster{
		ClusterName: "default/bookstore-v1",
		Weight:      100,
	}
	assert.Equal(actual, expected)
}

func TestGetHostnamesForService(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()

	actual, err := mc.GetHostnamesForService(tests.BookstoreV1Service)
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

	trafficTargets, err := mc.listTrafficTargetPermutations(tests.BookstoreTrafficTarget, tests.BookstoreTrafficTarget.Spec.Sources[0], tests.BookstoreTrafficTarget.Spec.Destination)
	assert.Nil(err)

	var actualTargetNames []string
	for _, target := range trafficTargets {
		actualTargetNames = append(actualTargetNames, target.Name)
	}

	expected := []string{
		utils.GetTrafficTargetName(tests.BookstoreTrafficTargetName, tests.BookbuyerService, tests.BookstoreV1Service),
		utils.GetTrafficTargetName(tests.BookstoreTrafficTargetName, tests.BookbuyerService, tests.BookstoreV3Service),
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

func TestGetWeightedBackendsForService(t *testing.T) {
	assert := assert.New(t)
	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	type getWeightedBackendsForServiceTest struct {
		input  service.MeshService
		output []service.WeightedService
	}

	getWeightedBackendsForServiceTests := []getWeightedBackendsForServiceTest{
		{
			input:  tests.BookstoreApexService,
			output: []service.WeightedService{tests.BookstoreV1WeightedService, tests.BookstoreV2WeightedService, tests.BookstoreV3WeightedService},
		},
		{
			input: tests.BookbuyerService,
			output: []service.WeightedService{
				{
					Service: tests.BookbuyerService,
					Weight:  constants.ClusterWeightAcceptAll,
				},
			},
		},
	}

	for _, test := range getWeightedBackendsForServiceTests {
		backendServices := mc.getWeightedBackendsForService(test.input)
		assert.ElementsMatch(backendServices, test.output)
	}
}

func TestGetRouteKey(t *testing.T) {
	assert := assert.New(t)

	type getRouteKeyTest struct {
		inputRoute trafficpolicy.HTTPRoute
		inputMap   map[*trafficpolicy.HTTPRoute][]service.WeightedService
		output     *trafficpolicy.HTTPRoute
	}

	getRouteKeyTests := []getRouteKeyTest{
		{
			inputRoute: tests.BookstoreV2TrafficPolicy.HTTPRoutes[0],
			inputMap: map[*trafficpolicy.HTTPRoute][]service.WeightedService{
				&tests.BookstoreV1TrafficPolicy.HTTPRoutes[0]: {{Service: tests.BookstoreV1Service, Weight: constants.ClusterWeightAcceptAll}},
			},
			output: &tests.BookstoreV1TrafficPolicy.HTTPRoutes[0],
		},
		{
			inputRoute: tests.BookstoreV1TrafficPolicy.HTTPRoutes[1],
			inputMap: map[*trafficpolicy.HTTPRoute][]service.WeightedService{
				&tests.BookstoreV1TrafficPolicy.HTTPRoutes[0]: {{Service: tests.BookstoreV1Service, Weight: constants.ClusterWeightAcceptAll}},
			},
			output: nil,
		},
	}

	for _, test := range getRouteKeyTests {
		key := getRouteKey(test.inputRoute, test.inputMap)
		assert.Equal(key, test.output)
	}
}

func TestGetRouteWeightedServices(t *testing.T) {
	assert := assert.New(t)

	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	src := tests.BookbuyerService
	dst := tests.BookstoreApexService

	trafficTargets := map[string]trafficpolicy.TrafficTarget{
		hashSrcDstService(tests.BookbuyerService, tests.BookstoreV1Service): tests.BookstoreV1TrafficPolicy,
		hashSrcDstService(tests.BookbuyerService, tests.BookstoreV2Service): tests.BookstoreV2TrafficPolicy,
	}

	expectedRouteWeightedSvcs := []trafficpolicy.RouteWeightedServices{
		{
			HTTPRoute:        tests.BookstoreV1TrafficPolicy.HTTPRoutes[0],
			WeightedServices: []service.WeightedService{tests.BookstoreV1WeightedService, tests.BookstoreV2WeightedService},
		},
		{
			HTTPRoute:        tests.BookstoreV1TrafficPolicy.HTTPRoutes[1],
			WeightedServices: []service.WeightedService{tests.BookstoreV1WeightedService},
		},
	}

	actual := mc.getRouteWeightedServices(src, dst, trafficTargets)

	assert.ElementsMatch(expectedRouteWeightedSvcs, actual)
}

func TestGetTrafficRoutesForService(t *testing.T) {
	assert := assert.New(t)

	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	//TODO: Test names
	trafficTargets := map[string]trafficpolicy.TrafficTarget{
		hashSrcDstService(tests.BookbuyerService, tests.BookstoreV1Service): {
			Name:        utils.GetTrafficTargetName(tests.BookstoreTrafficTargetName, tests.BookbuyerService, tests.BookstoreV1Service),
			Destination: tests.BookstoreV1Service,
			Source:      tests.BookbuyerService,
			HTTPRoutes:  tests.BookstoreV1TrafficPolicy.HTTPRoutes,
		},
		hashSrcDstService(tests.BookbuyerService, tests.BookstoreV2Service): {
			Name:        utils.GetTrafficTargetName(tests.BookstoreV2TrafficTargetName, tests.BookbuyerService, tests.BookstoreV2Service),
			Destination: tests.BookstoreV2Service,
			Source:      tests.BookbuyerService,
			HTTPRoutes:  tests.BookstoreV2TrafficPolicy.HTTPRoutes,
		},
		hashSrcDstService(tests.BookbuyerService, tests.BookstoreApexService): {
			Name:        utils.GetTrafficTargetName(tests.BookstoreTrafficTargetName, tests.BookbuyerService, tests.BookstoreApexService),
			Destination: tests.BookstoreApexService,
			Source:      tests.BookbuyerService,
			HTTPRoutes:  tests.BookstoreApexTrafficPolicy.HTTPRoutes,
		},
	}

	actualTrafficRoutes := mc.getTrafficRoutesForService(tests.BookbuyerService, trafficTargets)

	bookstoreV1DefaultWeightedService := service.WeightedService{Service: tests.BookstoreV1Service, Weight: constants.ClusterWeightAcceptAll}
	bookstoreV2DefaultWeightedService := service.WeightedService{Service: tests.BookstoreV2Service, Weight: constants.ClusterWeightAcceptAll}

	expectedTrafficRoutes := []trafficpolicy.TrafficRoutes{
		{
			//Name:        utils.GetTrafficTargetName(tests.BookstoreTrafficTargetName, tests.BookbuyerService, tests.BookstoreV1Service),
			Destination: tests.BookstoreV1Service,
			Source:      tests.BookbuyerService,
			RouteWeightedServices: []trafficpolicy.RouteWeightedServices{
				{
					HTTPRoute:        tests.BookstoreV1TrafficPolicy.HTTPRoutes[0],
					WeightedServices: []service.WeightedService{bookstoreV1DefaultWeightedService},
				},
				{
					HTTPRoute:        tests.BookstoreV1TrafficPolicy.HTTPRoutes[1],
					WeightedServices: []service.WeightedService{bookstoreV1DefaultWeightedService},
				},
			},
		},
		{
			//Name:        utils.GetTrafficTargetName(tests.BookstoreTrafficTargetName, tests.BookbuyerService, tests.BookstoreV2Service),
			Destination: tests.BookstoreV2Service,
			Source:      tests.BookbuyerService,
			RouteWeightedServices: []trafficpolicy.RouteWeightedServices{
				{
					HTTPRoute:        tests.BookstoreV2TrafficPolicy.HTTPRoutes[0],
					WeightedServices: []service.WeightedService{bookstoreV2DefaultWeightedService},
				},
			},
		},
		{
			//Name:        utils.GetTrafficTargetName(tests.BookstoreTrafficTargetName, tests.BookbuyerService, tests.BookstoreApexService),
			Destination: tests.BookstoreApexService,
			Source:      tests.BookbuyerService,
			RouteWeightedServices: []trafficpolicy.RouteWeightedServices{
				{
					HTTPRoute:        tests.BookstoreV1TrafficPolicy.HTTPRoutes[0],
					WeightedServices: []service.WeightedService{tests.BookstoreV1WeightedService, tests.BookstoreV2WeightedService},
				},
				{
					HTTPRoute:        tests.BookstoreV1TrafficPolicy.HTTPRoutes[1],
					WeightedServices: []service.WeightedService{tests.BookstoreV1WeightedService},
				},
			},
		},
	}
	assert.Equal(len(actualTrafficRoutes), len(expectedTrafficRoutes))
	numChecked := 0
	for _, actual := range actualTrafficRoutes {
		for _, expected := range expectedTrafficRoutes {
			if actual.Destination == expected.Destination {
				numChecked++
				assert.Equal(actual.Source, expected.Source)
				assert.ElementsMatch(actual.RouteWeightedServices, expected.RouteWeightedServices)
			}
		}
	}

	assert.Equal(3, numChecked)
	// Cannot use with list of structs with nested list
	//assert.ElementsMatch(actualTrafficRoutes, expectedTrafficRoutes)
}
