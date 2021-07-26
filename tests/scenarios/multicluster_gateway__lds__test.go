package scenarios

import (
	"fmt"
	"testing"

	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
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
	expectedGatewayListenerPort := uint32(15003)
	assert.Equal(expectedGatewayListenerAddress, listener.Address.GetSocketAddress().Address)
	assert.Equal(expectedGatewayListenerPort, listener.Address.GetSocketAddress().GetPortValue())

	assert.Len(listener.FilterChains, 1)
	assert.Len(listener.FilterChains[0].Filters, 2)

	assert.Equal("inbound-multicluster-gateway-filter-chain", listener.FilterChains[0].Name)

	assert.Equal("tls", listener.FilterChains[0].FilterChainMatch.TransportProtocol)
	assert.Equal([]string{"osm"}, listener.FilterChains[0].FilterChainMatch.ApplicationProtocols)
	assert.Equal([]string{"*.local"}, listener.FilterChains[0].FilterChainMatch.ServerNames)

	assert.Equal("envoy.filters.network.sni_cluster", listener.FilterChains[0].Filters[0].Name)
	assert.Equal("envoy.filters.network.tcp_proxy", listener.FilterChains[0].Filters[1].Name)

	assert.Equal("envoy.transport_sockets.tls", listener.FilterChains[0].TransportSocket.Name)

	dTLS := envoy_extensions_transport_sockets_tls_v3.DownstreamTlsContext{}
	err = listener.FilterChains[0].TransportSocket.GetTypedConfig().UnmarshalTo(&dTLS)
	assert.Nil(err)

	assert.Equal("service-cert:osm-system/osm", dTLS.CommonTlsContext.TlsCertificateSdsSecretConfigs[0].Name)
	assert.Equal("root-cert-for-mtls-inbound:osm-system/osm", dTLS.CommonTlsContext.GetValidationContextSdsSecretConfig().Name)
}
