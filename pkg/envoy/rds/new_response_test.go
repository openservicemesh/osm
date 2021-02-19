package rds

import (
	"fmt"
	"testing"

	set "github.com/deckarep/golang-set"
	xds_route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/mock/gomock"
	proto "github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	uuid := uuid.New().String()
	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.one.two.three.co.uk", uuid, "some-service", "some-namespace"))
	certSerialNumber := certificate.SerialNumber("123456")
	testProxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)

	testInbound := []*trafficpolicy.InboundTrafficPolicy{
		{
			Name:      "bookstore-v1.default",
			Hostnames: tests.BookstoreV1Hostnames,
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
						WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSet(tests.BookstoreServiceAccount),
				},
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
						WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSet(tests.BookstoreServiceAccount),
				},
			},
		},
	}

	testIngressInbound := []*trafficpolicy.InboundTrafficPolicy{
		{
			Name:      "bookstore-v1-default-bookstore-v1.default.svc.cluster.local",
			Hostnames: []string{"bookstore-v1.default.svc.cluster.local"},
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
							PathRegex: tests.BookstoreBuyPath,
							Methods:   []string{constants.WildcardHTTPMethod},
						},
						WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSet(tests.BookstoreServiceAccount),
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
							PathRegex: tests.BookstoreBuyPath,
							Methods:   []string{constants.WildcardHTTPMethod},
						},
						WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSet(tests.BookstoreServiceAccount),
				},
			},
		},
	}

	mockCatalog.EXPECT().ListTrafficPoliciesForServiceAccount(gomock.Any()).Return(testInbound, nil, nil).AnyTimes()
	mockCatalog.EXPECT().GetIngressPoliciesForService(gomock.Any(), gomock.Any()).Return(testIngressInbound, nil).AnyTimes()
	mockCatalog.EXPECT().GetServicesFromEnvoyCertificate(gomock.Any()).Return([]service.MeshService{tests.BookstoreV1Service}, nil).AnyTimes()

	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()

	actual, err := newResponse(mockCatalog, testProxy, nil, mockConfigurator, nil)
	assert.Nil(err)

	routeConfig := &xds_route.RouteConfiguration{}
	unmarshallErr := proto.UnmarshalAny(actual.GetResources()[0], routeConfig)
	if err != nil {
		t.Fatal(unmarshallErr)
	}
	assert.Equal("RDS_Inbound", routeConfig.Name)
	assert.Equal(2, len(routeConfig.VirtualHosts))

	assert.Equal("inbound_virtual-host|bookstore-v1.default", routeConfig.VirtualHosts[0].Name)
	assert.Equal(tests.BookstoreV1Hostnames, routeConfig.VirtualHosts[0].Domains)
	assert.Equal(3, len(routeConfig.VirtualHosts[0].Routes))
	assert.Equal(tests.BookstoreBuyHTTPRoute.PathRegex, routeConfig.VirtualHosts[0].Routes[0].GetMatch().GetSafeRegex().Regex)
	assert.Equal(tests.BookstoreSellHTTPRoute.PathRegex, routeConfig.VirtualHosts[0].Routes[1].GetMatch().GetSafeRegex().Regex)
	assert.Equal(tests.BookstoreBuyHTTPRoute.PathRegex, routeConfig.VirtualHosts[0].Routes[2].GetMatch().GetSafeRegex().Regex)

	assert.Equal("inbound_virtual-host|bookstore-v1.default|*", routeConfig.VirtualHosts[1].Name)
	assert.Equal([]string{"*"}, routeConfig.VirtualHosts[1].Domains)
	assert.Equal(1, len(routeConfig.VirtualHosts[1].Routes))
	assert.Equal(tests.BookstoreBuyHTTPRoute.PathRegex, routeConfig.VirtualHosts[1].Routes[0].GetMatch().GetSafeRegex().Regex)
}

func TestNewResponseWithPermissiveMode(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	uuid := uuid.New().String()
	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.one.two.three.co.uk", uuid, "some-service", "some-namespace"))
	certSerialNumber := certificate.SerialNumber("123456")
	testProxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)

	testPermissiveInbound := []*trafficpolicy.InboundTrafficPolicy{
		{
			Name:      "bookstore-v1.default",
			Hostnames: tests.BookstoreV1Hostnames,
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
							PathRegex: constants.RegexMatchAll,
							Methods:   []string{constants.WildcardHTTPMethod},
						},
						WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSet(tests.BookstoreServiceAccount),
				},
			},
		},
	}

	testOutbound := []*trafficpolicy.OutboundTrafficPolicy{
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
					HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
						PathRegex: constants.RegexMatchAll,
						Methods:   []string{constants.WildcardHTTPMethod},
					},
					WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
				},
			},
		},
	}

	testIngressInbound := []*trafficpolicy.InboundTrafficPolicy{
		{
			Name:      "bookstore-v1-default-bookstore-v1.default.svc.cluster.local",
			Hostnames: []string{"bookstore-v1.default.svc.cluster.local"},
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch: trafficpolicy.HTTPRouteMatch{
							PathRegex: tests.BookstoreBuyPath,
							Methods:   []string{constants.WildcardHTTPMethod},
						},
						WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSet(tests.BookstoreServiceAccount),
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
							PathRegex: tests.BookstoreBuyPath,
							Methods:   []string{constants.WildcardHTTPMethod},
						},
						WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSet(tests.BookstoreServiceAccount),
				},
			},
		},
	}

	mockCatalog.EXPECT().GetServicesFromEnvoyCertificate(gomock.Any()).Return([]service.MeshService{tests.BookstoreV1Service}, nil).AnyTimes()
	mockCatalog.EXPECT().ListPoliciesForPermissiveMode(gomock.Any()).Return(testPermissiveInbound, testOutbound, nil).AnyTimes()
	mockCatalog.EXPECT().GetIngressPoliciesForService(gomock.Any(), gomock.Any()).Return(testIngressInbound, nil).AnyTimes()

	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(true).AnyTimes()

	actual, err := newResponse(mockCatalog, testProxy, nil, mockConfigurator, nil)
	assert.Nil(err)

	routeConfig := &xds_route.RouteConfiguration{}
	unmarshallErr := proto.UnmarshalAny(actual.GetResources()[0], routeConfig)
	if err != nil {
		t.Fatal(unmarshallErr)
	}
	assert.Equal("RDS_Inbound", routeConfig.Name)
	assert.Equal(2, len(routeConfig.VirtualHosts))

	assert.Equal("inbound_virtual-host|bookstore-v1.default", routeConfig.VirtualHosts[0].Name)
	assert.Equal(tests.BookstoreV1Hostnames, routeConfig.VirtualHosts[0].Domains)
	assert.Equal(2, len(routeConfig.VirtualHosts[0].Routes))
	assert.Equal(constants.RegexMatchAll, routeConfig.VirtualHosts[0].Routes[0].GetMatch().GetSafeRegex().Regex)
	assert.Equal(tests.BookstoreBuyHTTPRoute.PathRegex, routeConfig.VirtualHosts[0].Routes[1].GetMatch().GetSafeRegex().Regex)

	assert.Equal("inbound_virtual-host|bookstore-v1.default|*", routeConfig.VirtualHosts[1].Name)
	assert.Equal([]string{"*"}, routeConfig.VirtualHosts[1].Domains)
	assert.Equal(1, len(routeConfig.VirtualHosts[1].Routes))
	assert.Equal(tests.BookstoreBuyHTTPRoute.PathRegex, routeConfig.VirtualHosts[1].Routes[0].GetMatch().GetSafeRegex().Regex)

	routeConfig = &xds_route.RouteConfiguration{}
	unmarshallErr = proto.UnmarshalAny(actual.GetResources()[1], routeConfig)
	if err != nil {
		t.Fatal(unmarshallErr)
	}
	assert.Equal("RDS_Outbound", routeConfig.Name)
	assert.Equal(1, len(routeConfig.VirtualHosts))

	assert.Equal("outbound_virtual-host|bookbuyer.default", routeConfig.VirtualHosts[0].Name)
	assert.Equal(1, len(routeConfig.VirtualHosts[0].Routes))
	assert.Equal(constants.RegexMatchAll, routeConfig.VirtualHosts[0].Routes[0].GetMatch().GetSafeRegex().Regex)
}
