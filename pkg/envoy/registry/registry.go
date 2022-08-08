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
		connectedProxies:   make(map[int64]*envoy.Proxy),
	}
}

// RegisterProxy registers a newly connected proxy.
func (pr *ProxyRegistry) RegisterProxy(proxy *envoy.Proxy) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.connectedProxies[proxy.GetConnectionID()] = proxy
	log.Debug().Str("proxy", proxy.String()).Msgf("Registered proxy %s from stream %d", proxy.String(), proxy.GetConnectionID())
}

// GetConnectedProxy loads a connected proxy from the registry.
func (pr *ProxyRegistry) GetConnectedProxy(connectionID int64) *envoy.Proxy {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.connectedProxies[connectionID]
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (pr *ProxyRegistry) UnregisterProxy(connectionID int64) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	delete(pr.connectedProxies, connectionID)
	log.Debug().Msgf("Unregistered proxy from stream %d", connectionID)
}

// GetConnectedProxyCount counts the number of connected proxies
func (pr *ProxyRegistry) GetConnectedProxyCount() int {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	return len(pr.connectedProxies)
}

// ListConnectedProxies lists the Envoy proxies already connected and the time they first connected.
func (pr *ProxyRegistry) ListConnectedProxies() map[string]*envoy.Proxy {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	proxies := make(map[string]*envoy.Proxy, len(pr.connectedProxies))
	for _, p := range pr.connectedProxies {
		// A proxy could connect twice quickly and not register the disconnect, so we return the proxy with the higher connection ID.
		if prior := proxies[p.UUID.String()]; prior == nil || prior.GetConnectionID() < p.GetConnectionID() {
			proxies[p.UUID.String()] = p
		}
	}
	return proxies
}
