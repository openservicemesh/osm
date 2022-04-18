package rds

import (
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                     string
		downstreamSA             identity.ServiceIdentity
		upstreamSA               identity.ServiceIdentity
		upstreamServices         []service.MeshService
		meshServices             []service.MeshService
		trafficSpec              spec.HTTPRouteGroup
		trafficSplit             split.TrafficSplit
		ingressInboundPolicies   []*trafficpolicy.InboundTrafficPolicy
		expectedInboundPolicies  map[int][]*trafficpolicy.InboundTrafficPolicy
		expectedOutboundPolicies map[int][]*trafficpolicy.OutboundTrafficPolicy
	}{
		{
			name:             "Test RDS NewResponse",
			downstreamSA:     tests.BookbuyerServiceIdentity,
			upstreamSA:       tests.BookstoreServiceIdentity,
			upstreamServices: []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			meshServices:     []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService},
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
					Namespace: tests.Namespace,
				},
				Spec: split.TrafficSplitSpec{
					Service: tests.BookstoreApexServiceName,
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
			},
			ingressInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
				{
					Name:      "bookstore-v1-default-bookstore-v1.default.svc.cluster.local",
					Hostnames: []string{"bookstore-v1.default.svc.cluster.local"},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          tests.BookstoreBuyPath,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
							},
							AllowedServiceIdentities: mapset.NewSet(tests.BookstoreServiceAccount.ToServiceIdentity()),
						},
					},
				},
				{
					Name:      "bookstore-v1.default|*",
					Hostnames: []string{"*"},
					Rules: []*trafficpolicy.Rule{
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
									Path:          tests.BookstoreBuyPath,
									PathMatchType: trafficpolicy.PathMatchRegex,
									Methods:       []string{constants.WildcardHTTPMethod},
								},
								WeightedClusters: mapset.NewSet(tests.BookstoreV1DefaultWeightedCluster),
							},
							AllowedServiceIdentities: mapset.NewSet(tests.BookstoreServiceAccount.ToServiceIdentity()),
						},
					},
				},
			},
			expectedInboundPolicies: map[int][]*trafficpolicy.InboundTrafficPolicy{
				8888: {
					{
						Name: "bookstore-v1.default",
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
						Rules: []*trafficpolicy.Rule{
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch: tests.BookstoreBuyHTTPRoute,
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "default/bookstore-v1",
										Weight:      100,
									}),
								},
								AllowedServiceIdentities: mapset.NewSet(identity.K8sServiceAccount{
									Name:      tests.BookbuyerServiceAccountName,
									Namespace: tests.Namespace,
								}.ToServiceIdentity()),
							},
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "default/bookstore-v1",
										Weight:      100,
									}),
								},
								AllowedServiceIdentities: mapset.NewSet(identity.K8sServiceAccount{
									Name:      tests.BookbuyerServiceAccountName,
									Namespace: tests.Namespace,
								}.ToServiceIdentity()),
							},
						},
					},
					{
						Name: tests.BookstoreApexServiceName,
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
										ClusterName: "default/bookstore-v1",
										Weight:      100,
									}),
								},
								AllowedServiceIdentities: mapset.NewSet(identity.K8sServiceAccount{
									Name:      tests.BookbuyerServiceAccountName,
									Namespace: tests.Namespace,
								}.ToServiceIdentity()),
							},
							{
								Route: trafficpolicy.RouteWeightedClusters{
									HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
									WeightedClusters: mapset.NewSet(service.WeightedCluster{
										ClusterName: "default/bookstore-v1",
										Weight:      100,
									}),
								},
								AllowedServiceIdentities: mapset.NewSet(identity.K8sServiceAccount{
									Name:      tests.BookbuyerServiceAccountName,
									Namespace: tests.Namespace,
								}.ToServiceIdentity()),
							},
						},
					},
				},
			},
			expectedOutboundPolicies: map[int][]*trafficpolicy.OutboundTrafficPolicy{
				80: {
					{
						Name:      tests.BookstoreApexServiceName,
						Hostnames: tests.BookstoreApexHostnames,
						Routes: []*trafficpolicy.RouteWeightedClusters{
							{
								HTTPRouteMatch: tests.WildCardRouteMatch,
								WeightedClusters: mapset.NewSetFromSlice([]interface{}{
									service.WeightedCluster{ClusterName: "default/bookstore-v1|80", Weight: 0},
									service.WeightedCluster{ClusterName: "default/bookstore-v2|80", Weight: 100},
								}),
							},
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
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
			kubeClient := testclient.NewSimpleClientset()
			proxy, err := getBookstoreV1Proxy(kubeClient)
			assert.Nil(err)

			proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
				return []service.MeshService{tests.BookstoreV1Service}, nil
			}), nil)

			for _, meshSvc := range tc.meshServices {
				k8sService := tests.NewServiceFixture(meshSvc.Name, meshSvc.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(meshSvc).Return(k8sService).AnyTimes()
			}

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()

			mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*spec.HTTPRouteGroup{&tc.trafficSpec}).AnyTimes()
			mockMeshSpec.EXPECT().ListTrafficSplits().Return([]*split.TrafficSplit{&tc.trafficSplit}).AnyTimes()
			trafficTarget := tests.NewSMITrafficTarget(tc.downstreamSA, tc.upstreamSA)
			mockMeshSpec.EXPECT().ListTrafficTargets().Return([]*access.TrafficTarget{&trafficTarget}).AnyTimes()

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

			mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
				EnableWASMStats: false,
			}).AnyTimes()

			mockCatalog.EXPECT().GetInboundMeshTrafficPolicy(gomock.Any(), gomock.Any()).Return(&trafficpolicy.InboundMeshTrafficPolicy{HTTPRouteConfigsPerPort: tc.expectedInboundPolicies}).AnyTimes()
			mockCatalog.EXPECT().GetOutboundMeshTrafficPolicy(gomock.Any()).Return(&trafficpolicy.OutboundMeshTrafficPolicy{HTTPRouteConfigsPerPort: tc.expectedOutboundPolicies}).AnyTimes()
			mockCatalog.EXPECT().GetIngressTrafficPolicy(gomock.Any()).Return(&trafficpolicy.IngressTrafficPolicy{HTTPRoutePolicies: tc.ingressInboundPolicies}, nil).AnyTimes()
			mockCatalog.EXPECT().GetEgressTrafficPolicy(gomock.Any()).Return(nil, nil).AnyTimes()

			// Empty discovery request
			discoveryRequest := xds_discovery.DiscoveryRequest{
				ResourceNames: []string{},
			}

			resources, err := NewResponse(mockCatalog, proxy, &discoveryRequest, mockConfigurator, nil, proxyRegistry)
			assert.Nil(err)
			assert.NotNil(resources)

			// The RDS response will have two route configurations
			// 1. rds-inbound
			// 2. rds-outbound
			// 3. rds-ingress
			assert.Equal(3, len(resources))

			// Check the inbound route configuration
			routeConfig, ok := resources[0].(*xds_route.RouteConfiguration)
			assert.True(ok)

			// The rds-inbound will have the following virtual hosts :
			// inbound_virtual-host|bookstore-v1.default
			// inbound_virtual-host|bookstore-apex
			assert.Equal("rds-inbound.8888", routeConfig.Name)
			assert.Equal(2, len(routeConfig.VirtualHosts))

			assert.Equal("inbound_virtual-host|bookstore-v1.default", routeConfig.VirtualHosts[0].Name)
			assert.Equal(tests.BookstoreV1Hostnames, routeConfig.VirtualHosts[0].Domains)
			assert.Equal(2, len(routeConfig.VirtualHosts[0].Routes))
			assert.Equal(tests.BookstoreBuyHTTPRoute.Path, routeConfig.VirtualHosts[0].Routes[0].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})
			assert.Equal(tests.BookstoreSellHTTPRoute.Path, routeConfig.VirtualHosts[0].Routes[1].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[0].Routes[1].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[0].Routes[1].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})

			assert.Equal("inbound_virtual-host|bookstore-apex", routeConfig.VirtualHosts[1].Name)
			assert.Equal(tests.BookstoreApexHostnames, routeConfig.VirtualHosts[1].Domains)
			assert.Equal(2, len(routeConfig.VirtualHosts[1].Routes))
			assert.Equal(tests.BookstoreBuyHTTPRoute.Path, routeConfig.VirtualHosts[1].Routes[0].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[1].Routes[0].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[1].Routes[0].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})
			assert.Equal(tests.BookstoreSellHTTPRoute.Path, routeConfig.VirtualHosts[1].Routes[1].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[1].Routes[1].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[1].Routes[1].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})

			// Check the outbound route configuration
			routeConfig, ok = resources[1].(*xds_route.RouteConfiguration)
			assert.True(ok)

			// The rds-outbound will have the following virtual hosts :
			// outbound_virtual-host|bookstore-apex
			assert.Equal("rds-outbound.80", routeConfig.Name)
			assert.Equal(1, len(routeConfig.VirtualHosts))

			assert.Equal("outbound_virtual-host|bookstore-apex", routeConfig.VirtualHosts[0].Name)
			assert.Equal(tests.BookstoreApexHostnames, routeConfig.VirtualHosts[0].Domains)
			assert.Equal(1, len(routeConfig.VirtualHosts[0].Routes))
			assert.Equal(tests.WildCardRouteMatch.Path, routeConfig.VirtualHosts[0].Routes[0].GetMatch().GetSafeRegex().Regex)
			assert.Equal(2, len(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})

			// Check the ingress route configuration
			routeConfig, ok = resources[2].(*xds_route.RouteConfiguration)
			assert.True(ok)

			// ingress_virtual-host|bookstore-v1-default-bookstore-v1.default.svc.cluster.local
			assert.Equal("ingress_virtual-host|bookstore-v1-default-bookstore-v1.default.svc.cluster.local", routeConfig.VirtualHosts[0].Name)
			assert.Equal([]string{"bookstore-v1.default.svc.cluster.local"}, routeConfig.VirtualHosts[0].Domains)
			assert.Equal(1, len(routeConfig.VirtualHosts[0].Routes))
			assert.Equal(tests.BookstoreBuyHTTPRoute.Path, routeConfig.VirtualHosts[0].Routes[0].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})

			// inbound_virtual-host|bookstore-v1.default|*
			assert.Equal("ingress_virtual-host|bookstore-v1.default|*", routeConfig.VirtualHosts[1].Name)
			assert.Equal([]string{"*"}, routeConfig.VirtualHosts[1].Domains)
			assert.Equal(1, len(routeConfig.VirtualHosts[1].Routes))
			assert.Equal(tests.BookstoreBuyHTTPRoute.Path, routeConfig.VirtualHosts[1].Routes[0].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[1].Routes[0].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[1].Routes[0].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})
		})
	}
}

func getBookstoreV1Proxy(kubeClient kubernetes.Interface) (*envoy.Proxy, error) {
	// Create pod for bookbuyer
	bookbuyerPodLabels := map[string]string{
		constants.AppLabel:               tests.BookbuyerService.Name,
		constants.EnvoyUniqueIDLabelName: uuid.New().String(),
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookbuyerServiceName, tests.BookbuyerServiceAccountName, bookbuyerPodLabels); err != nil {
		return nil, err
	}

	// Create pod for bookstore-v1
	bookstoreV1PodLabels := map[string]string{
		constants.AppLabel:               tests.BookstoreV1ServiceName,
		constants.EnvoyUniqueIDLabelName: tests.ProxyUUID,
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookstoreV1ServiceName, tests.BookstoreServiceAccountName, bookstoreV1PodLabels); err != nil {
		return nil, err
	}

	// Create a pod for bookstore-v2
	bookstoreV2PodLabels := map[string]string{
		constants.AppLabel:               tests.BookstoreV1ServiceName,
		constants.EnvoyUniqueIDLabelName: uuid.New().String(),
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookstoreV2ServiceName, tests.BookstoreServiceAccountName, bookstoreV2PodLabels); err != nil {
		return nil, err
	}

	// Create service for bookstore-v1 and bookstore-v2
	for _, svcName := range []string{tests.BookstoreV1ServiceName, tests.BookstoreV2ServiceName} {
		selectors := map[string]string{
			constants.AppLabel: svcName,
		}
		if _, err := tests.MakeService(kubeClient, svcName, selectors); err != nil {
			return nil, err
		}
	}

	// Create service for traffic split apex
	for _, svcName := range []string{tests.BookstoreApexServiceName} {
		selectors := map[string]string{
			constants.AppLabel: "bookstore",
		}
		if _, err := tests.MakeService(kubeClient, svcName, selectors); err != nil {
			return nil, err
		}
	}

	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", tests.ProxyUUID, tests.BookstoreServiceIdentity, tests.Namespace))
	certSerialNumber := certificate.SerialNumber("123456")
	return envoy.NewProxy(certCommonName, certSerialNumber, nil)
}

func TestResponseRequestCompletion(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	uuid := uuid.New().String()
	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.one.two.three.co.uk", uuid, "some-service", "some-namespace"))
	certSerialNumber := certificate.SerialNumber("123456")
	testProxy, err := envoy.NewProxy(certCommonName, certSerialNumber, nil)
	assert.Nil(err)

	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return []service.MeshService{tests.BookstoreV1Service}, nil
	}), nil)

	mockCatalog.EXPECT().GetInboundMeshTrafficPolicy(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCatalog.EXPECT().GetOutboundMeshTrafficPolicy(gomock.Any()).Return(nil).AnyTimes()
	mockCatalog.EXPECT().GetIngressTrafficPolicy(gomock.Any()).Return(nil, nil).AnyTimes()
	mockCatalog.EXPECT().GetEgressTrafficPolicy(gomock.Any()).Return(nil, nil).AnyTimes()
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{
		EnableWASMStats: false,
	}).AnyTimes()

	testCases := []struct {
		request *xds_discovery.DiscoveryRequest
	}{
		{
			request: &xds_discovery.DiscoveryRequest{
				ResourceNames: []string{"foo", "bar"},
			},
		},
		{
			request: &xds_discovery.DiscoveryRequest{
				ResourceNames: []string{"rds-inbound", "rds-outbound", "ingress", "bar", "doge"},
			},
		},
		{
			request: &xds_discovery.DiscoveryRequest{
				ResourceNames: []string{},
			},
		},
		{
			request: nil,
		},
	}

	for _, tc := range testCases {
		resources, err := NewResponse(mockCatalog, testProxy, tc.request, mockConfigurator, nil, proxyRegistry)
		assert.Nil(err)

		if tc.request != nil {
			requestMapset := mapset.NewSet()
			for idx := range tc.request.ResourceNames {
				requestMapset.Add(tc.request.ResourceNames[idx])
			}

			responseMapset := mapset.NewSet()
			for _, res := range resources {
				responseMapset.Add(cache.GetResourceName(res))
			}

			// verify all request resources where fulfilled
			diffMapset := requestMapset.Difference(responseMapset)
			assert.Equal(0, diffMapset.Cardinality())
		}
	}
}
