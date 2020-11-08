package catalog

import (
	"fmt"
	reflect "reflect"
	"testing"

	mapset "github.com/deckarep/golang-set"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

var (
	testPermissiveDefault = false

	getAllRoute = trafficpolicy.HTTPRoute{
		PathRegex: "/all",
		Methods:   []string{"GET"},
		Headers: map[string]string{
			"user-agent": "some-agent",
		},
	}
	getSomeRoute = trafficpolicy.HTTPRoute{
		PathRegex: "/some",
		Methods:   []string{"GET"},
		Headers: map[string]string{
			"user-agent": "another-agent",
		},
	}
)

func TestListTrafficPoliciesForService(t *testing.T) {
	assert := assert.New(t)
	testCases := []struct {
		name             string
		input            service.K8sServiceAccount
		expectedInbound  []*trafficpolicy.TrafficPolicy
		expectedOutbound []*trafficpolicy.TrafficPolicy
		mc               *MeshCatalog
	}{
		{
			name:  "bookbuyer service account in permissive mode",
			input: tests.BookbuyerServiceAccount,
			expectedInbound: []*trafficpolicy.TrafficPolicy{
				&trafficpolicy.TrafficPolicy{
					Name:        "bookstore-v2-default-bookbuyer-default",
					Source:      tests.BookstoreV2Service,
					Destination: tests.BookbuyerService,
					Hostnames:   tests.BookbuyerHostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookbuyer",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookstore-v1-default-bookbuyer-default",
					Source:      tests.BookstoreV1Service,
					Destination: tests.BookbuyerService,
					Hostnames:   tests.BookbuyerHostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookbuyer",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookstore-apex-default-bookbuyer-default",
					Source:      tests.BookstoreApexService,
					Destination: tests.BookbuyerService,
					Hostnames:   tests.BookbuyerHostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookbuyer",
								Weight:      100,
							}),
						},
					},
				},
			},
			expectedOutbound: []*trafficpolicy.TrafficPolicy{
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-apex-default",
					Destination: tests.BookstoreApexService,
					Source:      tests.BookbuyerService,
					Hostnames:   tests.BookstoreApexHostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-apex",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v1-default",
					Destination: tests.BookstoreV1Service,
					Source:      tests.BookbuyerService,
					Hostnames:   tests.BookstoreV1Hostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v1",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v2-default",
					Destination: tests.BookstoreV2Service,
					Source:      tests.BookbuyerService,
					Hostnames:   tests.BookstoreV2Hostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
					},
				},
			},
			mc: newFakeMeshCatalogForRoutes(t, true),
		}, {
			name:            "bookbuyer service account in non-permissive mode",
			input:           tests.BookbuyerServiceAccount,
			expectedInbound: []*trafficpolicy.TrafficPolicy{},
			expectedOutbound: []*trafficpolicy.TrafficPolicy{
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-apex-default",
					Destination: tests.BookstoreApexService,
					Source:      tests.BookbuyerService,
					Hostnames:   tests.BookstoreApexHostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-apex",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v1-default",
					Destination: tests.BookstoreV1Service,
					Source:      tests.BookbuyerService,
					Hostnames:   tests.BookstoreV1Hostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v1",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v2-default",
					Destination: tests.BookstoreV2Service,
					Source:      tests.BookbuyerService,
					Hostnames:   tests.BookstoreV2Hostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
					},
				},
			},
			mc: newFakeMeshCatalogForRoutes(t, false),
		},
	}
	for _, tc := range testCases {
		inbound, outbound, err := tc.mc.ListTrafficPoliciesForService(tests.BookbuyerServiceAccount)
		assert.Nil(err)
		assert.ElementsMatch(tc.expectedInbound, inbound)
		assert.ElementsMatch(tc.expectedOutbound, outbound)
	}

}

func TestBuildTrafficPolicies(t *testing.T) {
	mc := newFakeMeshCatalogForRoutes(t, testPermissiveDefault)

	testCases := []struct {
		Name             string
		sourceServices   []service.MeshService
		destServices     []service.MeshService
		routes           []trafficpolicy.HTTPRoute
		expectedPolicies []*trafficpolicy.TrafficPolicy
	}{
		{
			Name:           "policy for 1 source service and 1 destination service with 1 route",
			sourceServices: []service.MeshService{tests.BookbuyerService},
			destServices:   []service.MeshService{tests.BookstoreV2Service},
			routes:         []trafficpolicy.HTTPRoute{getAllRoute},
			expectedPolicies: []*trafficpolicy.TrafficPolicy{
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v2-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV2Service,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: getAllRoute,
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
			Name:           "policy for 1 source service and 2 destination services with one destination service that doesn't exist",
			sourceServices: []service.MeshService{tests.BookbuyerService},
			destServices: []service.MeshService{tests.BookstoreV2Service, service.MeshService{
				Namespace: "default",
				Name:      "nonexistentservices",
			}},
			routes: []trafficpolicy.HTTPRoute{getAllRoute},
			expectedPolicies: []*trafficpolicy.TrafficPolicy{
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v2-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV2Service,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: getAllRoute,
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
			Name:           "policies for 1 source service, 2 destination services, and multiple routes ",
			sourceServices: []service.MeshService{tests.BookbuyerService},
			destServices:   []service.MeshService{tests.BookstoreV2Service, tests.BookstoreV1Service},
			routes:         []trafficpolicy.HTTPRoute{getAllRoute, getSomeRoute},
			expectedPolicies: []*trafficpolicy.TrafficPolicy{
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v2-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV2Service,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: getAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: getSomeRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
					},
					Hostnames: tests.BookstoreV2Hostnames,
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v1-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV1Service,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: getAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v1",
								Weight:      100,
							}),
						},
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: getSomeRoute,
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
		policies := mc.buildTrafficPolicies(tc.sourceServices, tc.destServices, tc.routes)
		assert.ElementsMatch(t, tc.expectedPolicies, policies, tc.Name)
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

func TestHTTPRoutesFromRules(t *testing.T) {
	assert := assert.New(t)
	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	testCases := []struct {
		name           string
		rules          []target.TrafficTargetRule
		namespace      string
		expectedRoutes []trafficpolicy.HTTPRoute
	}{
		{
			rules: []target.TrafficTargetRule{
				target.TrafficTargetRule{
					Kind:    "HTTPRouteGroup",
					Name:    tests.RouteGroupName,
					Matches: []string{tests.BuyBooksMatchName},
				},
			},
			namespace: tests.Namespace,
			expectedRoutes: []trafficpolicy.HTTPRoute{
				trafficpolicy.HTTPRoute{
					PathRegex: tests.BookstoreBuyPath,
					Methods:   []string{"GET"},
					Headers: map[string]string{
						"user-agent": tests.HTTPUserAgent,
					},
				},
			},
		},
		{
			rules: []target.TrafficTargetRule{
				target.TrafficTargetRule{
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
		routes, err := mc.HTTPRoutesFromRules(tc.rules, tc.namespace) // returns []trafficpolicy.HTTPRoute
		assert.Nil(err)
		assert.EqualValues(tc.expectedRoutes, routes)
	}

}

func TestListAllowedInboundServices(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()

	actualList, err := mc.ListAllowedInboundServices(tests.BookstoreServiceAccount)
	assert.Nil(err)
	expectedList := []service.MeshService{tests.BookbuyerService}
	assert.ElementsMatch(actualList, expectedList)
}

func TestListAllowedOutboundServices(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()
	actualList, err := mc.ListAllowedOutboundServices(tests.BookbuyerServiceAccount)
	assert.Nil(err)

	expectedList := []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService}
	assert.ElementsMatch(actualList, expectedList)
}

func TestGetResolvableHostnamesForUpstreamService(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalogForRoutes(t, testPermissiveDefault)

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
			actual, err := mc.GetResolvableHostnamesForUpstreamService(tc.downstream, tc.upstream)
			if tc.expectedErr == false {
				assert.Nil(err)
			} else {
				assert.NotNil(err)
			}
			assert.Equal(actual, tc.expectedHostnames, tc.name)
		})
	}
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
			tests.BookstoreV1Hostnames,
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

func TestGetTrafficSpecName(t *testing.T) {
	assert := assert.New(t)

	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	actual := mc.getTrafficSpecName("HTTPRouteGroup", tests.Namespace, tests.RouteGroupName)
	expected := trafficpolicy.TrafficSpecName(fmt.Sprintf("HTTPRouteGroup/%s/%s", tests.Namespace, tests.RouteGroupName))
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

func TestBuildAllowAllTrafficPolicies(t *testing.T) {
	assert := assert.New(t)

	mc := newFakeMeshCatalog()

	expectedInbound := []*trafficpolicy.TrafficPolicy{
		&trafficpolicy.TrafficPolicy{
			Name:        "bookstore-v1-default-bookbuyer-default",
			Source:      tests.BookstoreV1Service,
			Destination: tests.BookbuyerService,
			Hostnames:   tests.BookbuyerHostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: allowAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookbuyer",
						Weight:      100,
					}),
				},
			},
		},
		&trafficpolicy.TrafficPolicy{
			Name:        "bookstore-v2-default-bookbuyer-default",
			Source:      tests.BookstoreV2Service,
			Destination: tests.BookbuyerService,
			Hostnames:   tests.BookbuyerHostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: allowAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookbuyer",
						Weight:      100,
					}),
				},
			},
		},
		&trafficpolicy.TrafficPolicy{
			Name:        "bookstore-apex-default-bookbuyer-default",
			Source:      tests.BookstoreApexService,
			Destination: tests.BookbuyerService,
			Hostnames:   tests.BookbuyerHostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: allowAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookbuyer",
						Weight:      100,
					}),
				},
			},
		},
	}
	expectedOutbound := []*trafficpolicy.TrafficPolicy{
		&trafficpolicy.TrafficPolicy{
			Name:        "bookbuyer-default-bookstore-v1-default",
			Source:      tests.BookbuyerService,
			Destination: tests.BookstoreV1Service,
			Hostnames:   tests.BookstoreV1Hostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: allowAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v1",
						Weight:      100,
					}),
				},
			},
		},
		&trafficpolicy.TrafficPolicy{
			Name:        "bookbuyer-default-bookstore-v2-default",
			Source:      tests.BookbuyerService,
			Destination: tests.BookstoreV2Service,
			Hostnames:   tests.BookstoreV2Hostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: allowAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v2",
						Weight:      100,
					}),
				},
			},
		},
		&trafficpolicy.TrafficPolicy{
			Name:        "bookbuyer-default-bookstore-apex-default",
			Source:      tests.BookbuyerService,
			Destination: tests.BookstoreApexService,
			Hostnames:   tests.BookstoreApexHostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: allowAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-apex",
						Weight:      100,
					}),
				},
			},
		},
	}

	inbound, outbound, err := mc.buildAllowAllTrafficPolicies(tests.BookbuyerServiceAccount)
	assert.Nil(err)
	assert.ElementsMatch(expectedInbound, inbound)
	assert.ElementsMatch(expectedOutbound, outbound)

}

func TestListTrafficPoliciesFromTrafficTargets(t *testing.T) {
	assert := assert.New(t)
	mc := newFakeMeshCatalogForRoutes(t, testPermissiveDefault)

	testCases := []struct {
		input                    service.K8sServiceAccount
		expectedInboundPolicies  []*trafficpolicy.TrafficPolicy
		expectedOutboundPolicies []*trafficpolicy.TrafficPolicy
	}{
		{
			input: tests.BookstoreServiceAccount,
			expectedInboundPolicies: []*trafficpolicy.TrafficPolicy{
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v2-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV2Service,
					Hostnames:   tests.BookstoreV2Hostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: trafficpolicy.HTTPRoute{
								PathRegex: "/buy",
								Methods:   []string{"GET"},
								Headers: map[string]string{
									"user-agent": tests.HTTPUserAgent,
								},
							},
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: trafficpolicy.HTTPRoute{
								PathRegex: "/sell",
								Methods:   []string{"GET"},
								Headers: map[string]string{
									"user-agent": tests.HTTPUserAgent,
								},
							},
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v1-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV1Service,
					Hostnames:   tests.BookstoreV1Hostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: trafficpolicy.HTTPRoute{
								PathRegex: "/buy",
								Methods:   []string{"GET"},
								Headers: map[string]string{
									"user-agent": tests.HTTPUserAgent,
								},
							},
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v1",
								Weight:      100,
							}),
						},
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: trafficpolicy.HTTPRoute{
								PathRegex: "/sell",
								Methods:   []string{"GET"},
								Headers: map[string]string{
									"user-agent": tests.HTTPUserAgent,
								},
							},
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v1",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-apex-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreApexService,
					Hostnames:   tests.BookstoreApexHostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: trafficpolicy.HTTPRoute{
								PathRegex: "/buy",
								Methods:   []string{"GET"},
								Headers: map[string]string{
									"user-agent": tests.HTTPUserAgent,
								},
							},
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-apex",
								Weight:      100,
							}),
						},
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: trafficpolicy.HTTPRoute{
								PathRegex: "/sell",
								Methods:   []string{"GET"},
								Headers: map[string]string{
									"user-agent": tests.HTTPUserAgent,
								},
							},
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-apex",
								Weight:      100,
							}),
						},
					},
				},
			},
			expectedOutboundPolicies: []*trafficpolicy.TrafficPolicy{},
		},
		{
			input:                   tests.BookbuyerServiceAccount,
			expectedInboundPolicies: []*trafficpolicy.TrafficPolicy{},
			expectedOutboundPolicies: []*trafficpolicy.TrafficPolicy{
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v1-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV1Service,
					Hostnames:   tests.BookstoreV1Hostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v1",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-v2-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreV2Service,
					Hostnames:   tests.BookstoreV2Hostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-v2",
								Weight:      100,
							}),
						},
					},
				},
				&trafficpolicy.TrafficPolicy{
					Name:        "bookbuyer-default-bookstore-apex-default",
					Source:      tests.BookbuyerService,
					Destination: tests.BookstoreApexService,
					Hostnames:   tests.BookstoreApexHostnames,
					HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
						trafficpolicy.RouteWeightedClusters{
							HTTPRoute: allowAllRoute,
							WeightedClusters: mapset.NewSet(service.WeightedCluster{
								ClusterName: "default/bookstore-apex",
								Weight:      100,
							}),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		inboundPolicies, outboundPolicies, err := mc.listTrafficPoliciesFromTrafficTargets(tc.input)
		assert.Nil(err)
		assert.ElementsMatch(tc.expectedInboundPolicies, inboundPolicies)
		assert.ElementsMatch(tc.expectedOutboundPolicies, outboundPolicies)

	}
}

func TestConsolidatePolicies(t *testing.T) {
	assert := assert.New(t)
	policies := []*trafficpolicy.TrafficPolicy{
		&trafficpolicy.TrafficPolicy{
			Name:        "bookbuyer-default-bookstore-v1-default",
			Source:      tests.BookbuyerService,
			Destination: tests.BookstoreV1Service,
			Hostnames:   tests.BookstoreV1Hostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: allowAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v1",
						Weight:      100,
					}),
				},
			},
		},
		&trafficpolicy.TrafficPolicy{
			Name:        "bookbuyer-default-bookstore-v1-default",
			Source:      tests.BookbuyerService,
			Destination: tests.BookstoreV1Service,
			Hostnames:   tests.BookstoreV1Hostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: getAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v1",
						Weight:      100,
					}),
				},
			},
		},
		&trafficpolicy.TrafficPolicy{
			Name:        "bookbuyer-default-bookstore-v1-default",
			Source:      tests.BookbuyerService,
			Destination: tests.BookstoreV1Service,
			Hostnames:   tests.BookstoreV1Hostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: getSomeRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v1",
						Weight:      100,
					}),
				},
			},
		},

		&trafficpolicy.TrafficPolicy{
			Name:        "bookbuyer-default-bookstore-v2-default",
			Source:      tests.BookbuyerService,
			Destination: tests.BookstoreV2Service,
			Hostnames:   tests.BookstoreV2Hostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: getSomeRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v2",
						Weight:      100,
					}),
				},
			},
		},
	}
	expectedPolicies := []*trafficpolicy.TrafficPolicy{
		&trafficpolicy.TrafficPolicy{
			Name:        "bookbuyer-default-bookstore-v1-default",
			Source:      tests.BookbuyerService,
			Destination: tests.BookstoreV1Service,
			Hostnames:   tests.BookstoreV1Hostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: allowAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v1",
						Weight:      100,
					}),
				},
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: getAllRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v1",
						Weight:      100,
					}),
				},
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: getSomeRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v1",
						Weight:      100,
					}),
				},
			},
		},
		&trafficpolicy.TrafficPolicy{
			Name:        "bookbuyer-default-bookstore-v2-default",
			Source:      tests.BookbuyerService,
			Destination: tests.BookstoreV2Service,
			Hostnames:   tests.BookstoreV2Hostnames,
			HTTPRoutesClusters: []trafficpolicy.RouteWeightedClusters{
				trafficpolicy.RouteWeightedClusters{
					HTTPRoute: getSomeRoute,
					WeightedClusters: mapset.NewSet(service.WeightedCluster{
						ClusterName: "default/bookstore-v2",
						Weight:      100,
					}),
				},
			},
		},
	}
	actualPolicies := consolidatePolicies(policies)
	assert.ElementsMatch(expectedPolicies, actualPolicies)

}
