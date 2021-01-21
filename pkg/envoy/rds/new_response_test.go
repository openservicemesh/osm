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
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func TestNewResponse(t *testing.T) {
	assert := tassert.New(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)

	uuid := uuid.New().String()
	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s.one.two.three.co.uk", uuid, "some-service", "some-namespace"))
	certSerialNumber := certificate.SerialNumber("123456")
	testProxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)

	testInbound := []*trafficpolicy.InboundTrafficPolicy{
		{
			Name:      "bookstore-v1-default",
			Hostnames: tests.BookstoreV1Hostnames,
			Rules: []*trafficpolicy.Rule{
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreBuyHTTPRoute,
						WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSet(tests.BookbuyerServiceAccount),
				},
				{
					Route: trafficpolicy.RouteWeightedClusters{
						HTTPRouteMatch:   tests.BookstoreSellHTTPRoute,
						WeightedClusters: set.NewSet(tests.BookstoreV1DefaultWeightedCluster),
					},
					AllowedServiceAccounts: set.NewSet(tests.BookbuyerServiceAccount),
				},
			},
		},
	}

	mockCatalog.EXPECT().ListTrafficPoliciesForServiceAccount(gomock.Any()).Return(testInbound, nil, nil).AnyTimes()

	actual, err := newResponse(mockCatalog, testProxy)
	assert.Nil(err)

	routeConfig := &xds_route.RouteConfiguration{}
	unmarshallErr := proto.UnmarshalAny(actual.GetResources()[0], routeConfig)
	if err != nil {
		t.Fatal(unmarshallErr)
	}
	assert.Equal("RDS_Inbound", routeConfig.Name)
	assert.Equal(1, len(routeConfig.VirtualHosts))
	assert.Equal("inbound_virtualHost|bookstore-v1-default", routeConfig.VirtualHosts[0].Name)
}
