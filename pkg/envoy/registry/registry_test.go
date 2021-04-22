package registry

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

func TestIsProxyConnected(t *testing.T) {
	assert := tassert.New(t)
	proxyRegistry := NewProxyRegistry()

	cn1 := certificate.CommonName("common.name.1")
	cn2 := certificate.CommonName("common.name.2")
	proxy1 := envoy.NewProxy(cn1, "", nil)
	proxy2 := envoy.NewProxy(cn2, "", nil)

	assert.False(proxyRegistry.IsProxyConnected(cn1))
	assert.False(proxyRegistry.IsProxyConnected(cn2))

	proxyRegistry.RegisterProxy(proxy1)

	assert.True(proxyRegistry.IsProxyConnected(cn1))
	assert.False(proxyRegistry.IsProxyConnected(cn2))

	proxyRegistry.RegisterProxy(proxy2)

	assert.True(proxyRegistry.IsProxyConnected(cn1))
	assert.True(proxyRegistry.IsProxyConnected(cn2))

	proxyRegistry.UnregisterProxy(proxy1)

	assert.False(proxyRegistry.IsProxyConnected(cn1))
	assert.True(proxyRegistry.IsProxyConnected(cn2))

	proxyRegistry.UnregisterProxy(proxy2)

	assert.False(proxyRegistry.IsProxyConnected(cn1))
	assert.False(proxyRegistry.IsProxyConnected(cn2))
}
