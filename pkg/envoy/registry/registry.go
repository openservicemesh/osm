package registry

import (
	"time"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// NewProxyRegistry initializes a new empty *ProxyRegistry.
func NewProxyRegistry(mapper ProxyServiceMapper, msgBroker *messaging.Broker) *ProxyRegistry {
	return &ProxyRegistry{
		ProxyServiceMapper: mapper,
		msgBroker:          msgBroker,
	}
}

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (pr *ProxyRegistry) RegisterProxy(proxy *envoy.Proxy) {
	pr.connectedProxies.Store(proxy.UUID.String(), connectedProxy{
		proxy:       proxy,
		connectedAt: time.Now(),
	})
	log.Debug().Str("proxy", proxy.String()).Msg("Registered new proxy")
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (pr *ProxyRegistry) UnregisterProxy(p *envoy.Proxy) {
	pr.connectedProxies.Delete(p.UUID.String())
	log.Debug().Msgf("Unregistered proxy %s", p.String())
}

// GetConnectedProxyCount counts the number of connected proxies
func (pr *ProxyRegistry) GetConnectedProxyCount() int {
	return len(pr.ListConnectedProxies())
}
