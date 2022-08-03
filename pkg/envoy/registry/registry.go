package registry

import (
	"github.com/cskr/pubsub"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// NewProxyRegistry initializes a new empty *ProxyRegistry.
func NewProxyRegistry(mapper ProxyServiceMapper, msgBroker *messaging.Broker) *ProxyRegistry {
	return &ProxyRegistry{
		ProxyServiceMapper: mapper,
		msgBroker:          msgBroker,
		connectedProxies:   make(map[int64]*envoy.Proxy),
		pubsub:             pubsub.New(0),
		// ch 							: make(chan *envoy.Proxy, 10),
	}
}

// RegisterProxy registers a newly connected proxy.
func (pr *ProxyRegistry) RegisterProxy(proxy *envoy.Proxy) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.connectedProxies[proxy.GetConnectionID()] = proxy
	log.Debug().Str("proxy", proxy.String()).Msg("Registered new proxy")
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
}

// GetConnectedProxyCount counts the number of connected proxies
func (pr *ProxyRegistry) GetConnectedProxyCount() int {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	return len(pr.connectedProxies)
}

// ListConnectedProxies lists the Envoy proxies already connected and the time they first connected.
func (pr *ProxyRegistry) ListConnectedProxies() []*envoy.Proxy {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	proxies := make([]*envoy.Proxy, 0, len(pr.connectedProxies))
	for _, p := range pr.connectedProxies {
		proxies = append(proxies, p)
	}
	return proxies
}
