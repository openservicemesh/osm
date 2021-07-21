package lds

import (
	"testing"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetGatewayFilterChain(t *testing.T) {
	assert := tassert.New(t)
	filterChain, err := getGatewayFilterChain("one.two")
	assert.Nil(err)
	assert.Equal(len(filterChain.Filters), 2)
}

func TestBuildGatewayListeners(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetFeatureFlags().Return(v1alpha1.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()

	// Mock calls used to build the HTTP connection manager
	mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetTracingEndpoint().Return("test-api").AnyTimes()
	mockConfigurator.EXPECT().GetClusterDomain().Return("cluster-x").AnyTimes()
	id := identity.K8sServiceAccount{Name: "osm-multicluster-gateway", Namespace: "osm-system"}.ToServiceIdentity()

	returnMeshSvc := []service.MeshService{
		tests.BookbuyerService,
		tests.BookwarehouseService,
		// Non local should get filtered out.
		{
			Name:          "bookthief",
			Namespace:     "default",
			ClusterDomain: "non-local",
		},
	}
	mockCatalog.EXPECT().ListMeshServicesForIdentity(id).Return(returnMeshSvc).AnyTimes()

	svcMap := map[uint32]string{80: ""}
	mockCatalog.EXPECT().GetPortToProtocolMappingForService(tests.BookbuyerService).Return(svcMap, nil).AnyTimes()
	mockCatalog.EXPECT().GetPortToProtocolMappingForService(tests.BookwarehouseService).Return(svcMap, nil).AnyTimes()

	mockCatalog.EXPECT().GetWeightedClustersForUpstream(tests.BookbuyerService).Return(nil).AnyTimes()
	mockCatalog.EXPECT().GetWeightedClustersForUpstream(tests.BookwarehouseService).Return(nil).AnyTimes()

	mockCatalog.EXPECT().ListMeshServicesForIdentity("gateway.osm-system.cluster.local").AnyTimes()

	lb := &listenerBuilder{
		meshCatalog:     mockCatalog,
		cfg:             mockConfigurator,
		serviceIdentity: id,
	}

	listeners := lb.buildGatewayListeners()
	assert.Len(listeners, 1)
	listener, ok := listeners[0].(*xds_listener.Listener)
	assert.True(ok)
	assert.Len(listener.FilterChains, 1)

	// This filter is of utmost importance to the functioning of the Multicluster Gateway.
	expectedFilterChain := "envoy.filters.network.sni_cluster"
	assert.Equal(listener.FilterChains[0].Filters[0].Name, expectedFilterChain)

	assert.Equal(listener.FilterChains[0].FilterChainMatch.ServerNames, []string{"*.local"})
	assert.Equal(listener.FilterChains[0].FilterChainMatch.ApplicationProtocols, []string{"osm"})
	assert.Equal(listener.FilterChains[0].FilterChainMatch.TransportProtocol, "tls")
	assert.Len(listener.FilterChains[0].Filters, 2)
}
