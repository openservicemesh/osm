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
		connectedProxies:   make(map[string]*envoy.Proxy),
	}
}

// RegisterProxy registers a newly connected proxy.
func (pr *ProxyRegistry) RegisterProxy(proxy *envoy.Proxy) {
	uuid := proxy.UUID.String()
	if pr.GetConnectedProxy(uuid) != nil {
		log.Debug().Str("proxy", proxy.String()).Msgf("Proxy %s already registered", proxy.String())
		return
	}

	pr.mu.Lock()
	pr.connectedProxies[uuid] = proxy
	pr.mu.Unlock()
	log.Debug().Str("proxy", proxy.String()).Msg("Registered new proxy")
}

// GetConnectedProxy loads a connected proxy from the registry.
func (pr *ProxyRegistry) GetConnectedProxy(uuid string) *envoy.Proxy {
	pr.mu.Lock()
	p, ok := pr.connectedProxies[uuid]
	pr.mu.Unlock()

	if !ok {
		return nil
	}
	return p
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (pr *ProxyRegistry) UnregisterProxy(proxy *envoy.Proxy) {
	uuid := proxy.UUID.String()
	if pr.GetConnectedProxy(uuid) == nil {
		log.Debug().Str("proxy", proxy.String()).Msgf("Proxy %s does not exist", proxy.String())
		return
	}

	pr.mu.Lock()
	delete(pr.connectedProxies, uuid)
	pr.mu.Unlock()
	log.Debug().Msgf("Unregistered proxy %s", proxy.String())
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

	proxies := make(map[string]*envoy.Proxy)
	for uuid, p := range pr.connectedProxies {
		proxies[uuid] = p
	}
	return proxies
}
