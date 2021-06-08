package catalog

import (
	"fmt"
	"reflect"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
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

func TestListInboundTrafficPolicies(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                    string
		downstreamSA            identity.ServiceIdentity
		upstreamSA              identity.ServiceIdentity
		upstreamServices        []service.MeshService
		meshServices            []service.MeshService
		meshServiceAccounts     []identity.K8sServiceAccount
		trafficSpec             spec.HTTPRouteGroup
		trafficSplit            split.TrafficSplit
		expectedInboundPolicies []*trafficpolicy.InboundTrafficPolicy
		permissiveMode          bool
	}{
		{
			name:         "inbound policies in same namespaces, without traffic split",
			downstreamSA: tests.BookbuyerServiceIdentity,
			upstreamSA:   tests.BookstoreServiceIdentity,
			upstreamServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			meshServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			meshServiceAccounts: []identity.K8sServiceAccount{},
			trafficSpec: spec.HTTPRouteGroup{
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
			},
			trafficSplit: split.TrafficSplit{},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore.default.local",
					Hostnames: []string{
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
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
			},
			permissiveMode: false,
		},
		{
			name:         "inbound policies in same namespaces, with traffic split",
			downstreamSA: tests.BookbuyerServiceIdentity,
			upstreamSA:   tests.BookstoreServiceIdentity,
			upstreamServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			meshServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}, {
				Name:      "bookstore-apex",
				Namespace: "default",
			}},
			meshServiceAccounts: []identity.K8sServiceAccount{},
			trafficSpec: spec.HTTPRouteGroup{
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
			},
			trafficSplit: split.TrafficSplit{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
				},
				Spec: split.TrafficSplitSpec{
					Service: "bookstore-apex",
					Backends: []split.TrafficSplitBackend{
						{
							Service: "bookstore",
							Weight:  tests.Weight90,
						},
						{
							Service: tests.BookstoreV2ServiceName,
							Weight:  tests.Weight10,
						},
					},
				},
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore.default.local",
					Hostnames: []string{
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
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
				{
					Name: "bookstore-apex.default.local",
					Hostnames: []string{
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
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
			},
			permissiveMode: false,
		},
		{
			name:         "inbound policies in same namespaces, with traffic split and host header",
			downstreamSA: tests.BookbuyerServiceIdentity,
			upstreamSA:   tests.BookstoreServiceIdentity,
			upstreamServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			meshServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}, {
				Name:      "bookstore-apex",
				Namespace: "default",
			}},
			meshServiceAccounts: []identity.K8sServiceAccount{},
			trafficSpec: spec.HTTPRouteGroup{
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
								"user-agent":  tests.HTTPUserAgent,
								hostHeaderKey: tests.HTTPHostHeader,
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
			},
			trafficSplit: split.TrafficSplit{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
				},
				Spec: split.TrafficSplitSpec{
					Service: "bookstore-apex",
					Backends: []split.TrafficSplitBackend{
						{
							Service: "bookstore",
							Weight:  tests.Weight90,
						},
						{
							Service: tests.BookstoreV2ServiceName,
							Weight:  tests.Weight10,
						},
					},
				},
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore.default.local",
					Hostnames: []string{
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
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
				{
					Name: "bookstore-apex.default.local",
					Hostnames: []string{
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
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
				{
					Name:      tests.HTTPHostHeader,
					Hostnames: []string{tests.HTTPHostHeader},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRouteWithHost,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
			},
			permissiveMode: false,
		},
		{
			name:                "permissive mode",
			downstreamSA:        tests.BookstoreServiceIdentity,
			upstreamSA:          tests.BookbuyerServiceIdentity,
			upstreamServices:    []service.MeshService{tests.BookbuyerService},
			meshServices:        []service.MeshService{tests.BookbuyerService, tests.BookstoreV1Service, tests.BookstoreV2Service},
			meshServiceAccounts: []identity.K8sServiceAccount{tests.BookbuyerServiceAccount, tests.BookstoreServiceAccount},
			trafficSpec:         spec.HTTPRouteGroup{},
			trafficSplit:        split.TrafficSplit{},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookbuyer.default.local",
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
								HTTPRouteMatch:   tests.WildCardRouteMatch,
								WeightedClusters: mapset.NewSet(tests.BookbuyerDefaultWeightedCluster),
							},
							AllowedServiceAccounts: mapset.NewSet(wildcardServiceAccount),
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

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
				configurator:       mockConfigurator,
			}

			var services []*corev1.Service
			for _, meshSvc := range tc.meshServices {
				k8sService := tests.NewServiceFixture(meshSvc.Name, meshSvc.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(meshSvc).Return(k8sService).AnyTimes()
				services = append(services, k8sService)
			}

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()

			if tc.permissiveMode {
				var serviceAccounts []*corev1.ServiceAccount
				for _, sa := range tc.meshServiceAccounts {
					k8sSvcAccount := tests.NewServiceAccountFixture(sa.Name, sa.Namespace)
					serviceAccounts = append(serviceAccounts, k8sSvcAccount)
				}
				mockKubeController.EXPECT().ListServices().Return(services).AnyTimes()
				mockKubeController.EXPECT().ListServiceAccounts().Return(serviceAccounts).AnyTimes()
			} else {
				mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*spec.HTTPRouteGroup{&tc.trafficSpec}).AnyTimes()
				mockMeshSpec.EXPECT().ListTrafficSplits().Return([]*split.TrafficSplit{&tc.trafficSplit}).AnyTimes()
				trafficTarget := tests.NewSMITrafficTarget(tc.downstreamSA, tc.upstreamSA)
				mockMeshSpec.EXPECT().ListTrafficTargets().Return([]*access.TrafficTarget{&trafficTarget}).AnyTimes()
			}

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).AnyTimes()
			actual := mc.ListInboundTrafficPolicies(tc.upstreamSA, tc.upstreamServices)
			assert.ElementsMatch(tc.expectedInboundPolicies, actual)
		})
	}
}

func TestListInboundPoliciesForTrafficSplits(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                    string
		downstreamSA            identity.ServiceIdentity
		upstreamSA              identity.ServiceIdentity
		upstreamServices        []service.MeshService
		meshServices            []service.MeshService
		trafficSpec             spec.HTTPRouteGroup
		trafficSplit            split.TrafficSplit
		expectedInboundPolicies []*trafficpolicy.InboundTrafficPolicy
	}{
		// TODO(draychev): use ServiceIdentity in the rest of the tests [https://github.com/openservicemesh/osm/issues/2218]
		{
			name: "inbound policies in same namespaces, without traffic split",
			downstreamSA: identity.K8sServiceAccount{
				Name:      "bookbuyer",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamSA: identity.K8sServiceAccount{
				Name:      "bookstore",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			meshServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			trafficSpec: spec.HTTPRouteGroup{
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
			},
			trafficSplit:            split.TrafficSplit{},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{},
		},
		{
			name: "inbound policies in same namespaces, with traffic split",
			downstreamSA: identity.K8sServiceAccount{
				Name:      "bookbuyer",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamSA: identity.K8sServiceAccount{
				Name:      "bookstore",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			meshServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}, {
				Name:      "bookstore-apex",
				Namespace: "default",
			}},
			trafficSpec: spec.HTTPRouteGroup{
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
			},
			trafficSplit: split.TrafficSplit{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
				},
				Spec: split.TrafficSplitSpec{
					Service: "bookstore-apex",
					Backends: []split.TrafficSplitBackend{
						{
							Service: "bookstore",
							Weight:  tests.Weight90,
						},
						{
							Service: tests.BookstoreV2ServiceName,
							Weight:  tests.Weight10,
						},
					},
				},
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore-apex.default.local",
					Hostnames: []string{
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
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
			},
		},
		{
			name: "inbound policies in same namespaces, with traffic split with namespaced root service",
			downstreamSA: identity.K8sServiceAccount{
				Name:      "bookbuyer",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamSA: identity.K8sServiceAccount{
				Name:      "bookstore",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			meshServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}, {
				Name:      "bookstore-apex",
				Namespace: "default",
			}},
			trafficSpec: spec.HTTPRouteGroup{
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
			},
			trafficSplit: split.TrafficSplit{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
				},
				Spec: split.TrafficSplitSpec{
					Service: "bookstore-apex.default",
					Backends: []split.TrafficSplitBackend{
						{
							Service: "bookstore",
							Weight:  tests.Weight90,
						},
						{
							Service: tests.BookstoreV2ServiceName,
							Weight:  tests.Weight10,
						},
					},
				},
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore-apex.default.local",
					Hostnames: []string{
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
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
			},
		},
		{
			name: "inbound policies in same namespaces: with traffic split (namespaced root service) and traffic target (having host header)",
			downstreamSA: identity.K8sServiceAccount{
				Name:      "bookbuyer",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamSA: identity.K8sServiceAccount{
				Name:      "bookstore",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			meshServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}, {
				Name:      "bookstore-apex",
				Namespace: "default",
			}},
			trafficSpec: spec.HTTPRouteGroup{
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
								"user-agent":  tests.HTTPUserAgent,
								hostHeaderKey: tests.HTTPHostHeader,
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
			},
			trafficSplit: split.TrafficSplit{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "default",
				},
				Spec: split.TrafficSplitSpec{
					Service: "bookstore-apex.default",
					Backends: []split.TrafficSplitBackend{
						{
							Service: "bookstore",
							Weight:  tests.Weight90,
						},
						{
							Service: tests.BookstoreV2ServiceName,
							Weight:  tests.Weight10,
						},
					},
				},
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore-apex.default.local",
					Hostnames: []string{
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
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
				{
					Name:      tests.HTTPHostHeader,
					Hostnames: []string{tests.HTTPHostHeader},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRouteWithHost,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
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

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
			}

			for _, meshSvc := range tc.meshServices {
				k8sService := tests.NewServiceFixture(meshSvc.Name, meshSvc.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(meshSvc).Return(k8sService).AnyTimes()
			}

			mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*spec.HTTPRouteGroup{&tc.trafficSpec}).AnyTimes()
			mockMeshSpec.EXPECT().ListTrafficSplits().Return([]*split.TrafficSplit{&tc.trafficSplit}).AnyTimes()
			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()

			trafficTarget := tests.NewSMITrafficTarget(tc.downstreamSA, tc.upstreamSA)
			mockMeshSpec.EXPECT().ListTrafficTargets().Return([]*access.TrafficTarget{&trafficTarget}).AnyTimes()

			actual := mc.listInboundPoliciesForTrafficSplits(tc.upstreamSA, tc.upstreamServices)
			assert.ElementsMatch(tc.expectedInboundPolicies, actual)
		})
	}
}

func TestBuildInboundPolicies(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                    string
		sourceSA                identity.ServiceIdentity
		destSA                  identity.ServiceIdentity
		inboundService          service.MeshService
		trafficSpec             spec.HTTPRouteGroup
		expectedInboundPolicies []*trafficpolicy.InboundTrafficPolicy
	}{
		{
			name: "inbound policies in different namespaces",
			sourceSA: identity.K8sServiceAccount{
				Name:      "bookbuyer",
				Namespace: "bookbuyer-ns",
			}.ToServiceIdentity(),
			destSA: identity.K8sServiceAccount{
				Name:      "bookstore",
				Namespace: "bookstore-ns",
			}.ToServiceIdentity(),
			inboundService: service.MeshService{
				Name:      "bookstore",
				Namespace: "bookstore-ns",
			},
			trafficSpec: spec.HTTPRouteGroup{
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
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore.bookstore-ns.local",
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
								HTTPRouteMatch: tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "bookstore-ns/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "bookbuyer-ns",
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "bookstore-ns/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "bookbuyer-ns",
							}),
						},
					},
				},
			},
		},
		{
			name: "inbound policies in same namespaces",
			sourceSA: identity.K8sServiceAccount{
				Name:      "bookbuyer",
				Namespace: "default",
			}.ToServiceIdentity(),
			destSA: identity.K8sServiceAccount{
				Name:      "bookstore",
				Namespace: "default",
			}.ToServiceIdentity(),
			inboundService: service.MeshService{
				Name:      "bookstore",
				Namespace: "default",
			},
			trafficSpec: spec.HTTPRouteGroup{
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
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore.default.local",
					Hostnames: []string{
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
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
			},
		},
		{
			name:           "inbound policies with host header",
			sourceSA:       tests.BookbuyerServiceIdentity,
			destSA:         tests.BookstoreServiceIdentity,
			inboundService: tests.BookstoreV1Service,
			trafficSpec: spec.HTTPRouteGroup{
				TypeMeta: v1.TypeMeta{
					APIVersion: "specs.smi-spec.io/v1alpha4",
					Kind:       "HTTPRouteGroup",
				},
				ObjectMeta: v1.ObjectMeta{
					Namespace: tests.Namespace,
					Name:      tests.RouteGroupName,
				},

				Spec: spec.HTTPRouteGroupSpec{
					Matches: []spec.HTTPMatch{
						{
							Name:      tests.BuyBooksMatchName,
							PathRegex: tests.BookstoreBuyPath,
							Methods:   []string{"GET"},
							Headers: map[string]string{
								"user-agent":  tests.HTTPUserAgent,
								hostHeaderKey: tests.HTTPHostHeader,
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
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name:      tests.BookstoreV1Service.Name + "." + tests.BookstoreV1Service.Namespace + ".local",
					Hostnames: tests.BookstoreV1Hostnames,
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore-v1",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
				{
					Name:      tests.HTTPHostHeader,
					Hostnames: []string{tests.HTTPHostHeader},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRouteWithHost,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore-v1",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
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

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
			}

			destK8sService := tests.NewServiceFixture(tc.inboundService.Name, tc.inboundService.Namespace, map[string]string{})
			mockKubeController.EXPECT().GetService(tc.inboundService).Return(destK8sService).AnyTimes()

			mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*spec.HTTPRouteGroup{&tc.trafficSpec}).AnyTimes()
			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockEndpointProvider.EXPECT().GetServicesForServiceAccount(tc.destSA).Return([]service.MeshService{tc.inboundService}, nil).AnyTimes()

			trafficTarget := tests.NewSMITrafficTarget(tc.sourceSA, tc.destSA)

			actual := mc.buildInboundPolicies(&trafficTarget, tc.inboundService)
			assert.ElementsMatch(tc.expectedInboundPolicies, actual)
		})
	}
}

func TestBuildInboundPermissiveModePolicies(t *testing.T) {
	assert := tassert.New(t)

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
					Name: "bookstore.bookstore-ns.local",
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
								HTTPRouteMatch: tests.WildCardRouteMatch,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "bookstore-ns/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(wildcardServiceAccount),
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

			k8sService := tests.NewServiceFixture(tc.meshService.Name, tc.meshService.Namespace, map[string]string{})

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockKubeController.EXPECT().GetService(tc.meshService).Return(k8sService)
			actual := mc.buildInboundPermissiveModePolicies(tc.meshService)
			assert.Len(actual, len(tc.expectedInboundPolicies))
			assert.ElementsMatch(tc.expectedInboundPolicies, actual)
		})
	}
}

func TestListInboundPoliciesFromTrafficTargets(t *testing.T) {
	assert := tassert.New(t)

	type testCase struct {
		name                      string
		downstreamServiceIdentity identity.ServiceIdentity
		upstreamServiceIdentity   identity.ServiceIdentity
		upstreamServices          []service.MeshService
		trafficSpec               spec.HTTPRouteGroup
		expectedInboundPolicies   []*trafficpolicy.InboundTrafficPolicy
	}

	testCases := []testCase{
		{
			name: "inbound policies in same namespaces",
			downstreamServiceIdentity: identity.K8sServiceAccount{
				Name:      "bookbuyer",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamServiceIdentity: identity.K8sServiceAccount{
				Name:      "bookstore",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			trafficSpec: spec.HTTPRouteGroup{
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
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore.default.local",
					Hostnames: []string{
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
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreBuyHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
			},
		},
		{
			name: "inbound policies in same namespaces with host header",
			downstreamServiceIdentity: identity.K8sServiceAccount{
				Name:      "bookbuyer",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamServiceIdentity: identity.K8sServiceAccount{
				Name:      "bookstore",
				Namespace: "default",
			}.ToServiceIdentity(),
			upstreamServices: []service.MeshService{{
				Name:      "bookstore",
				Namespace: "default",
			}},
			trafficSpec: spec.HTTPRouteGroup{
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
								"user-agent":  tests.HTTPUserAgent,
								hostHeaderKey: tests.HTTPHostHeader,
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
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name: "bookstore.default.local",
					Hostnames: []string{
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
					},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
							}),
						},
					},
				},
				{
					Name:      tests.HTTPHostHeader,
					Hostnames: []string{tests.HTTPHostHeader},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          tests.BookstoreBuyPath,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{"GET"},
									Headers: map[string]string{
										"user-agent":  tests.HTTPUserAgent,
										hostHeaderKey: tests.HTTPHostHeader,
									},
								},
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(identity.K8sServiceAccount{
								Name:      "bookbuyer",
								Namespace: "default",
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

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
			}

			for _, destMeshSvc := range tc.upstreamServices {
				destK8sService := tests.NewServiceFixture(destMeshSvc.Name, destMeshSvc.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(destMeshSvc).Return(destK8sService).AnyTimes()
			}

			mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*spec.HTTPRouteGroup{&tc.trafficSpec}).AnyTimes()
			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()
			mockEndpointProvider.EXPECT().GetServicesForServiceAccount(tc.upstreamServiceIdentity).Return(tc.upstreamServices, nil).AnyTimes()

			trafficTarget := tests.NewSMITrafficTarget(tc.downstreamServiceIdentity, tc.upstreamServiceIdentity)
			mockMeshSpec.EXPECT().ListTrafficTargets().Return([]*access.TrafficTarget{&trafficTarget}).AnyTimes()

			actual := mc.listInboundPoliciesFromTrafficTargets(tc.upstreamServiceIdentity, tc.upstreamServices)
			assert.ElementsMatch(tc.expectedInboundPolicies, actual, "The expected and actual do not match!")
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
			expectedRoutes: nil,
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

func TestGetHTTPPathsPerRoute(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                      string
		trafficSpec               spec.HTTPRouteGroup
		expectedHTTPPathsPerRoute map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch
	}{
		{
			name: "HTTP route with path, method and headers",
			trafficSpec: spec.HTTPRouteGroup{
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
			},
			expectedHTTPPathsPerRoute: map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch{
				"HTTPRouteGroup/default/bookstore-service-routes": {
					trafficpolicy.TrafficSpecMatchName(tests.BuyBooksMatchName): {
						Path:          tests.BookstoreBuyPath,
						PathMatchType: trafficpolicy.PathMatchRegex,
						Methods:       []string{"GET"},
						Headers: map[string]string{
							"user-agent": tests.HTTPUserAgent,
						},
					},
					trafficpolicy.TrafficSpecMatchName(tests.SellBooksMatchName): {
						Path:          tests.BookstoreSellPath,
						PathMatchType: trafficpolicy.PathMatchRegex,
						Methods:       []string{"GET"},
						Headers: map[string]string{
							"user-agent": tests.HTTPUserAgent,
						},
					},
				},
			},
		},
		{
			name: "HTTP route with only path",
			trafficSpec: spec.HTTPRouteGroup{
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
						},
						{
							Name:      tests.SellBooksMatchName,
							PathRegex: tests.BookstoreSellPath,
							Methods:   nil,
						},
					},
				},
			},
			expectedHTTPPathsPerRoute: map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch{
				"HTTPRouteGroup/default/bookstore-service-routes": {
					trafficpolicy.TrafficSpecMatchName(tests.BuyBooksMatchName): {
						Path:          tests.BookstoreBuyPath,
						PathMatchType: trafficpolicy.PathMatchRegex,
						Methods:       []string{"*"},
					},
					trafficpolicy.TrafficSpecMatchName(tests.SellBooksMatchName): {
						Path:          tests.BookstoreSellPath,
						PathMatchType: trafficpolicy.PathMatchRegex,
						Methods:       []string{"*"},
					},
				},
			},
		},
		{
			name: "HTTP route with only method",
			trafficSpec: spec.HTTPRouteGroup{
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
							Name:    tests.BuyBooksMatchName,
							Methods: []string{"GET"},
						},
					},
				},
			},
			expectedHTTPPathsPerRoute: map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch{
				"HTTPRouteGroup/default/bookstore-service-routes": {
					trafficpolicy.TrafficSpecMatchName(tests.BuyBooksMatchName): {
						Path:    ".*",
						Methods: []string{"GET"},
					},
				},
			},
		},
		{
			name: "HTTP route with only headers",
			trafficSpec: spec.HTTPRouteGroup{
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
							Name: tests.WildcardWithHeadersMatchName,
							Headers: map[string]string{
								"user-agent": tests.HTTPUserAgent,
							},
						},
					},
				},
			},
			expectedHTTPPathsPerRoute: map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.HTTPRouteMatch{
				"HTTPRouteGroup/default/bookstore-service-routes": {
					trafficpolicy.TrafficSpecMatchName(tests.WildcardWithHeadersMatchName): {
						Path:          ".*",
						PathMatchType: trafficpolicy.PathMatchRegex,
						Methods:       []string{"*"},
						Headers: map[string]string{
							"user-agent": tests.HTTPUserAgent,
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

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
			}

			mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*spec.HTTPRouteGroup{&tc.trafficSpec}).AnyTimes()
			actual, err := mc.getHTTPPathsPerRoute()
			assert.Nil(err)
			assert.True(reflect.DeepEqual(actual, tc.expectedHTTPPathsPerRoute))
		})
	}
}

func TestGetTrafficSpecName(t *testing.T) {
	assert := tassert.New(t)

	mc := MeshCatalog{meshSpec: smi.NewFakeMeshSpecClient()}

	actual := mc.getTrafficSpecName("HTTPRouteGroup", tests.Namespace, tests.RouteGroupName)
	expected := trafficpolicy.TrafficSpecName(fmt.Sprintf("HTTPRouteGroup/%s/%s", tests.Namespace, tests.RouteGroupName))
	assert.Equal(actual, expected)
}
