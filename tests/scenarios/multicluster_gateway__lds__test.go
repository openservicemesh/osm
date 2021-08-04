package scenarios

import (
	"fmt"
	"sort"
	"testing"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy/lds"
)

func TestMulticlusterGatewayListenerDiscoveryService(t *testing.T) {
	assert := tassert.New(t)

	// -------------------  SETUP  -------------------
	meshCatalog, proxy, proxyRegistry, mockConfigurator, err := setupMulticlusterGatewayTest(gomock.NewController(t))
	assert.Nil(err, fmt.Sprintf("Error setting up the test: %+v", err))

	// -------------------  TEST lds.NewResponse()  -------------------
	resources, err := lds.NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil, proxyRegistry)
	assert.Nil(err, fmt.Sprintf("lds.NewResponse return unexpected error: %+v", err))
	assert.NotNil(resources, "No LDS resources!")
	assert.Len(resources, 1)

	listener, ok := resources[0].(*xds_listener.Listener)
	assert.True(ok)
	assert.Equal("multicluster-listener", listener.Name)

	expectedGatewayListenerAddress := "0.0.0.0"
	expectedGatewayListenerPort := uint32(15443)
	assert.Equal(expectedGatewayListenerAddress, listener.Address.GetSocketAddress().Address)
	assert.Equal(expectedGatewayListenerPort, listener.Address.GetSocketAddress().GetPortValue())

	assert.Len(listener.FilterChains, 4)
	assert.Len(listener.FilterChains[0].Filters, 2)

	var actualNames []string
	for idx := range listener.FilterChains {
		actualNames = append(actualNames, listener.FilterChains[idx].Name)
	}

	sort.Strings(actualNames)
	expectedNames := []string{
		"multicluster-gateway-filter-chain-bookbuyer",
		"multicluster-gateway-filter-chain-bookstore-apex",
		"multicluster-gateway-filter-chain-bookstore-v1",
		"multicluster-gateway-filter-chain-bookstore-v2",
	}
	assert.Equal(actualNames, expectedNames)

	assert.Equal("tls", listener.FilterChains[0].FilterChainMatch.TransportProtocol)
	assert.Equal([]string{"osm"}, listener.FilterChains[0].FilterChainMatch.ApplicationProtocols)

	var actualServerNames []string
	var actualFilterNames []string
	for idx := range listener.FilterChains {
		actualServerNames = append(actualServerNames, listener.FilterChains[idx].FilterChainMatch.ServerNames...)
		for filterIDX := range listener.FilterChains[idx].Filters {
			actualFilterNames = append(actualFilterNames, listener.FilterChains[idx].Filters[filterIDX].Name)
		}
	}

	sort.Strings(actualServerNames)
	expectedServerNames := []string{
		"bookbuyer.default.svc.cluster.local",
		"bookstore-apex.default.svc.cluster.local",
		"bookstore-v1.default.svc.cluster.local",
		"bookstore-v2.default.svc.cluster.local",
	}
	assert.Equal(expectedServerNames, actualServerNames)

	sort.Strings(actualFilterNames)
	expectedFilterNames := []string{
		"envoy.filters.network.sni_cluster",
		"envoy.filters.network.sni_cluster",
		"envoy.filters.network.sni_cluster",
		"envoy.filters.network.sni_cluster",
		"envoy.filters.network.tcp_proxy",
		"envoy.filters.network.tcp_proxy",
		"envoy.filters.network.tcp_proxy",
		"envoy.filters.network.tcp_proxy",
	}
	assert.Equal(expectedFilterNames, actualFilterNames)
}
