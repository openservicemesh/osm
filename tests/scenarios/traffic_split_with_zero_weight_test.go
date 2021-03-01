package scenarios

import (
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/mock/gomock"
	proto "github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/rds"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestRDSRespose(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                     string
		downstreamSA             service.K8sServiceAccount
		upstreamSA               service.K8sServiceAccount
		upstreamServices         []service.MeshService
		meshServices             []service.MeshService
		trafficSpec              spec.HTTPRouteGroup
		trafficSplit             split.TrafficSplit
		expectedInboundPolicies  []*trafficpolicy.InboundTrafficPolicy
		expectedOutboundPolicies []*trafficpolicy.OutboundTrafficPolicy
	}{
		{
			name:             "Test RDS response with a traffic split having zero weight",
			downstreamSA:     tests.BookbuyerServiceAccount,
			upstreamSA:       tests.BookstoreServiceAccount,
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
							Weight:  0,
						},
						{
							Service: tests.BookstoreV2ServiceName,
							Weight:  100,
						},
					},
				},
			},
			expectedInboundPolicies: []*trafficpolicy.InboundTrafficPolicy{
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
									ClusterName: "default/bookstore-v1-local",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(service.K8sServiceAccount{
								Name:      tests.BookbuyerServiceAccountName,
								Namespace: tests.Namespace,
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore-v1-local",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(service.K8sServiceAccount{
								Name:      tests.BookbuyerServiceAccountName,
								Namespace: tests.Namespace,
							}),
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
									ClusterName: "default/bookstore-v1-local",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(service.K8sServiceAccount{
								Name:      tests.BookbuyerServiceAccountName,
								Namespace: tests.Namespace,
							}),
						},
						{
							Route: trafficpolicy.RouteWeightedClusters{
								HTTPRouteMatch: tests.BookstoreSellHTTPRoute,
								WeightedClusters: mapset.NewSet(service.WeightedCluster{
									ClusterName: "default/bookstore-v1-local",
									Weight:      100,
								}),
							},
							AllowedServiceAccounts: mapset.NewSet(service.K8sServiceAccount{
								Name:      tests.BookbuyerServiceAccountName,
								Namespace: tests.Namespace,
							}),
						},
					},
				},
			},
			expectedOutboundPolicies: []*trafficpolicy.OutboundTrafficPolicy{
				{
					Name:      tests.BookstoreApexServiceName,
					Hostnames: tests.BookstoreApexHostnames,
					Routes: []*trafficpolicy.RouteWeightedClusters{
						{
							HTTPRouteMatch: tests.WildCardRouteMatch,
							WeightedClusters: mapset.NewSetFromSlice([]interface{}{
								service.WeightedCluster{ClusterName: "default/bookstore-v1", Weight: 0},
								service.WeightedCluster{ClusterName: "default/bookstore-v2", Weight: 100},
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
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
			kubeClient := testclient.NewSimpleClientset()
			proxy, err := getBookstoreV1Proxy(kubeClient)
			assert.Nil(err)

			for _, meshSvc := range tc.meshServices {
				k8sService := tests.NewServiceFixture(meshSvc.Name, meshSvc.Namespace, map[string]string{})
				mockKubeController.EXPECT().GetService(meshSvc).Return(k8sService).AnyTimes()
			}

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()

			mockMeshSpec.EXPECT().ListHTTPTrafficSpecs().Return([]*spec.HTTPRouteGroup{&tc.trafficSpec}).AnyTimes()
			mockMeshSpec.EXPECT().ListTrafficSplits().Return([]*split.TrafficSplit{&tc.trafficSplit}).AnyTimes()
			trafficTarget := tests.NewSMITrafficTarget(tc.downstreamSA.Name, tc.downstreamSA.Namespace, tc.upstreamSA.Name, tc.upstreamSA.Namespace)
			mockMeshSpec.EXPECT().ListTrafficTargets().Return([]*access.TrafficTarget{&trafficTarget}).AnyTimes()

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

			mockCatalog.EXPECT().GetServicesFromEnvoyCertificate(gomock.Any()).Return([]service.MeshService{tests.BookstoreV1Service}, nil).AnyTimes()
			mockCatalog.EXPECT().ListInboundTrafficPolicies(gomock.Any(), gomock.Any()).Return(tc.expectedInboundPolicies).AnyTimes()
			mockCatalog.EXPECT().ListOutboundTrafficPolicies(gomock.Any()).Return(tc.expectedOutboundPolicies).AnyTimes()
			mockCatalog.EXPECT().GetIngressPoliciesForService(gomock.Any()).Return([]*trafficpolicy.InboundTrafficPolicy{}, nil).AnyTimes()

			actual, err := rds.NewResponse(mockCatalog, proxy, nil, mockConfigurator, nil)
			assert.Nil(err)
			assert.NotNil(actual)

			// The RDS response will have two route configurations
			// 1. RDS_Inbound
			// 2. RDS_Outbound
			routeConfig := &xds_route.RouteConfiguration{}
			assert.Equal(2, len(actual.GetResources()))

			// Check the inbound route configuration
			unmarshallErr := proto.UnmarshalAny(actual.GetResources()[0], routeConfig)
			if err != nil {
				t.Fatal(unmarshallErr)
			}

			// The RDS_Inbound will have the following virtual hosts :
			// inbound_virtual-host|bookstore-v1.default
			// inbound_virtual-host|bookstore-apex
			assert.Equal("RDS_Inbound", routeConfig.Name)
			assert.Equal(2, len(routeConfig.VirtualHosts))

			assert.Equal("inbound_virtual-host|bookstore-v1.default", routeConfig.VirtualHosts[0].Name)
			assert.Equal(tests.BookstoreV1Hostnames, routeConfig.VirtualHosts[0].Domains)
			assert.Equal(2, len(routeConfig.VirtualHosts[0].Routes))
			assert.Equal(tests.BookstoreBuyHTTPRoute.PathRegex, routeConfig.VirtualHosts[0].Routes[0].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})
			assert.Equal(tests.BookstoreSellHTTPRoute.PathRegex, routeConfig.VirtualHosts[0].Routes[1].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[0].Routes[1].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[0].Routes[1].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})

			assert.Equal("inbound_virtual-host|bookstore-apex", routeConfig.VirtualHosts[1].Name)
			assert.Equal(tests.BookstoreApexHostnames, routeConfig.VirtualHosts[1].Domains)
			assert.Equal(2, len(routeConfig.VirtualHosts[1].Routes))
			assert.Equal(tests.BookstoreBuyHTTPRoute.PathRegex, routeConfig.VirtualHosts[1].Routes[0].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[1].Routes[0].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[1].Routes[0].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})
			assert.Equal(tests.BookstoreSellHTTPRoute.PathRegex, routeConfig.VirtualHosts[1].Routes[1].GetMatch().GetSafeRegex().Regex)
			assert.Equal(1, len(routeConfig.VirtualHosts[1].Routes[1].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[1].Routes[1].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})

			// Check the outbound route configuration
			unmarshallErr = proto.UnmarshalAny(actual.GetResources()[1], routeConfig)
			if err != nil {
				t.Fatal(unmarshallErr)
			}

			// The RDS_Outbound will have the following virtual hosts :
			// outbound_virtual-host|bookstore-apex
			assert.Equal("RDS_Outbound", routeConfig.Name)
			assert.Equal(1, len(routeConfig.VirtualHosts))

			assert.Equal("outbound_virtual-host|bookstore-apex", routeConfig.VirtualHosts[0].Name)
			assert.Equal(tests.BookstoreApexHostnames, routeConfig.VirtualHosts[0].Domains)
			assert.Equal(1, len(routeConfig.VirtualHosts[0].Routes))
			assert.Equal(tests.WildCardRouteMatch.PathRegex, routeConfig.VirtualHosts[0].Routes[0].GetMatch().GetSafeRegex().Regex)
			assert.Equal(2, len(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().Clusters))
			assert.Equal(routeConfig.VirtualHosts[0].Routes[0].GetRoute().GetWeightedClusters().TotalWeight, &wrappers.UInt32Value{Value: uint32(100)})
		})
	}
}

func getBookstoreV1Proxy(kubeClient kubernetes.Interface) (*envoy.Proxy, error) {
	// Create pod for bookbuyer
	bookbuyerPodLabels := map[string]string{
		tests.SelectorKey:                tests.BookbuyerService.Name,
		constants.EnvoyUniqueIDLabelName: uuid.New().String(),
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookbuyerServiceName, tests.BookbuyerServiceAccountName, bookbuyerPodLabels); err != nil {
		return nil, err
	}

	// Create pod for bookstore-v1
	bookstoreV1PodLabels := map[string]string{
		tests.SelectorKey:                tests.BookstoreV1ServiceName,
		constants.EnvoyUniqueIDLabelName: tests.ProxyUUID,
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookstoreV1ServiceName, tests.BookstoreServiceAccountName, bookstoreV1PodLabels); err != nil {
		return nil, err
	}

	// Create a pod for bookstore-v2
	bookstoreV2PodLabels := map[string]string{
		tests.SelectorKey:                tests.BookstoreV1ServiceName,
		constants.EnvoyUniqueIDLabelName: uuid.New().String(),
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookstoreV2ServiceName, tests.BookstoreServiceAccountName, bookstoreV2PodLabels); err != nil {
		return nil, err
	}

	// Create service for bookstore-v1 and bookstore-v2
	for _, svcName := range []string{tests.BookstoreV1ServiceName, tests.BookstoreV2ServiceName} {
		selectors := map[string]string{
			tests.SelectorKey: svcName,
		}
		if _, err := tests.MakeService(kubeClient, svcName, selectors); err != nil {
			return nil, err
		}
	}

	// Create service for traffic split apex
	for _, svcName := range []string{tests.BookstoreApexServiceName} {
		selectors := map[string]string{
			tests.SelectorKey: "bookstore",
		}
		if _, err := tests.MakeService(kubeClient, svcName, selectors); err != nil {
			return nil, err
		}
	}

	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", tests.ProxyUUID, tests.BookstoreServiceAccount, tests.Namespace))
	certSerialNumber := certificate.SerialNumber("123456")
	proxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)
	return proxy, nil
}
