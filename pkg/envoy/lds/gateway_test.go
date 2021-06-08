package lds

import (
	"fmt"
	"testing"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestNewMultiClusterGatewayListener(t *testing.T) {
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
	id := identity.K8sServiceAccount{Name: "gateway", Namespace: "osm-system"}.ToServiceIdentity()
	mockCatalog.EXPECT().ListMeshServicesForIdentity(id).Return([]service.MeshService{
		tests.BookbuyerService,
		tests.BookwarehouseService,
		// Non local should get filtered out.
		{
			Name:          "bookthief",
			Namespace:     "default",
			ClusterDomain: "non-local",
		},
	})

	mockCatalog.EXPECT().GetPortToProtocolMappingForService(tests.BookbuyerService).Return(map[uint32]string{
		80: "",
	}, nil).AnyTimes()
	mockCatalog.EXPECT().GetPortToProtocolMappingForService(tests.BookwarehouseService).Return(map[uint32]string{
		80: "",
	}, nil).AnyTimes()

	mockCatalog.EXPECT().GetWeightedClustersForUpstream(tests.BookbuyerService).Return(nil).AnyTimes()
	mockCatalog.EXPECT().GetWeightedClustersForUpstream(tests.BookwarehouseService).Return(nil).AnyTimes()

	mockCatalog.EXPECT().GetServiceHostnames(tests.BookbuyerService, service.RemoteCluster).Return([]string{
		"bookbuyer.default.svc.cluster.cluster-x",
		"bookbuyer.default.svc.cluster.global",
	}, nil).AnyTimes()
	mockCatalog.EXPECT().GetServiceHostnames(tests.BookwarehouseService, service.RemoteCluster).Return([]string{
		"bookwarehouse.default.svc.cluster.cluster-x",
		"bookwarehouse.default.svc.cluster.global",
	}, nil).AnyTimes()

	lb := &listenerBuilder{
		meshCatalog:     mockCatalog,
		cfg:             mockConfigurator,
		serviceIdentity: id,
	}

	testCases := []struct {
		name string
		port uint32

		expectedFilterChainMatch *xds_listener.FilterChainMatch
	}{
		{
			name: "bookbuyer gateway filter chain",
			port: 80,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: 80},
				ServerNames: []string{
					"bookbuyer.default.svc.cluster.cluster-x",
					"bookbuyer.default.svc.cluster.global",
				},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
		},
		{
			name: "bookwarehouse gateway filter chain",
			port: 80,
			expectedFilterChainMatch: &xds_listener.FilterChainMatch{
				DestinationPort: &wrapperspb.UInt32Value{Value: 80},
				ServerNames: []string{
					"bookwarehouse.default.svc.cluster.cluster-x",
					"bookwarehouse.default.svc.cluster.global",
				},
				TransportProtocol:    "tls",
				ApplicationProtocols: []string{"osm"},
			},
		},
	}
	listeners := lb.buildGatewayListeners()
	assert.Len(listeners, 1)
	listener, ok := listeners[0].(*xds_listener.Listener)
	assert.True(ok)
	assert.Len(listener.FilterChains, 2)

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert.Equal(listener.FilterChains[i].FilterChainMatch.ServerNames, tc.expectedFilterChainMatch.ServerNames)
			assert.Equal(listener.FilterChains[i].FilterChainMatch.ApplicationProtocols, tc.expectedFilterChainMatch.ApplicationProtocols)
			assert.Equal(listener.FilterChains[i].FilterChainMatch.TransportProtocol, tc.expectedFilterChainMatch.TransportProtocol)
			assert.Equal(listener.FilterChains[i].FilterChainMatch.DestinationPort.Value, tc.expectedFilterChainMatch.DestinationPort.Value)
			assert.Len(listener.FilterChains[i].Filters, 1)
		})
	}
}
