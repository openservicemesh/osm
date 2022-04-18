package lds

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestBuildMulticlusterGatewayListeners(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigurator.EXPECT().GetFeatureFlags().Return(configv1alpha3.FeatureFlags{EnableMulticlusterMode: true}).AnyTimes()

	id := identity.K8sServiceAccount{Name: "osm", Namespace: "osm-system"}.ToServiceIdentity()
	meshServices := []service.MeshService{
		tests.BookstoreV1Service,
		tests.BookstoreV2Service,
	}

	mockCatalog.EXPECT().ListOutboundServicesForMulticlusterGateway().Return(meshServices).AnyTimes()
	lb := &listenerBuilder{
		meshCatalog:     mockCatalog,
		cfg:             mockConfigurator,
		serviceIdentity: id,
	}

	listener, err := lb.buildMulticlusterGatewayListener()
	assert.Nil(err)
	assert.Equal(listener.Name, multiclusterListenerName)
	assert.Equal(listener.Address, envoy.GetAddress(constants.WildcardIPAddr, multiclusterGatewayListenerPort))
	assert.Equal(len(listener.ListenerFilters), 1)
	assert.Len(listener.FilterChains, 2)
}

func TestGetGatewayFilterChains(t *testing.T) {
	assert := tassert.New(t)

	meshServices := []service.MeshService{tests.BookstoreV1Service}
	filterChains, err := getMulticlusterGatewayFilterChains(meshServices)
	assert.Nil(err)
	assert.NotNil(filterChains)
	assert.Equal(len(filterChains), 1)
	assert.Equal(filterChains[0].Name, fmt.Sprintf("%s-%s", multiclusterGatewayFilterChainName, tests.BookstoreV1ServiceName))
	assert.ElementsMatch(filterChains[0].FilterChainMatch.ServerNames, []string{tests.BookstoreV1Service.ServerName()})
	assert.Equal(filterChains[0].FilterChainMatch.ApplicationProtocols, []string{"osm"})
	assert.Equal(filterChains[0].FilterChainMatch.TransportProtocol, "tls")
	assert.Equal(len(filterChains[0].Filters), 2)
}
