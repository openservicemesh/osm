package registry

import (
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

// RegisterProxy registers a newly connected proxy.
func (pr *ProxyRegistry) RegisterProxy(proxy *envoy.Proxy) {
	pr.connectedProxies.Store(proxy.UUID.String(), proxy)
	log.Debug().Str("proxy", proxy.String()).Msg("Registered new proxy")
}

// GetConnectedProxy loads a connected proxy from the registry.
func (pr *ProxyRegistry) GetConnectedProxy(uuid string) *envoy.Proxy {
	p, ok := pr.connectedProxies.Load(uuid)
	if !ok {
		return nil
	}
	return p.(*envoy.Proxy)
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (pr *ProxyRegistry) UnregisterProxy(p *envoy.Proxy) {
	pr.connectedProxies.Delete(p.UUID.String())
	log.Debug().Msgf("Unregistered proxy %s", p.String())
}

// GetConnectedProxyCount counts the number of connected proxies
// TODO(steeling): switch to a regular map with mutex so we can get the count without iterating the entire list.
func (pr *ProxyRegistry) GetConnectedProxyCount() int {
	return len(pr.ListConnectedProxies())
}

// ListConnectedProxies lists the Envoy proxies already connected and the time they first connected.
func (pr *ProxyRegistry) ListConnectedProxies() map[string]*envoy.Proxy {
	proxies := make(map[string]*envoy.Proxy)
	pr.connectedProxies.Range(func(keyIface, propsIface interface{}) bool {
		uuid := keyIface.(string)
		proxies[uuid] = propsIface.(*envoy.Proxy)
		return true // continue the iteration
	})
	return proxies
}
