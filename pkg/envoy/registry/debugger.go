package registry

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// ListConnectedProxies lists the Envoy proxies already connected and the time they first connected.
func (pr *ProxyRegistry) ListConnectedProxies() map[certificate.CommonName]*envoy.Proxy {
	proxies := make(map[certificate.CommonName]*envoy.Proxy)
	pr.connectedProxies.Range(func(cnIface, propsIface interface{}) bool {
		cn := cnIface.(certificate.CommonName)
		props := propsIface.(connectedProxy)
		if _, isDisconnected := pr.disconnectedProxies.Load(cn); !isDisconnected {
			proxies[cn] = props.proxy
		}
		return true // continue the iteration
	})
	return proxies
}

// ListDisconnectedProxies lists the Envoy proxies disconnected and the time last seen.
func (pr *ProxyRegistry) ListDisconnectedProxies() map[certificate.CommonName]time.Time {
	proxies := make(map[certificate.CommonName]time.Time)
	pr.disconnectedProxies.Range(func(cnInterface, disconnectedProxyInterface interface{}) bool {
		cn := cnInterface.(certificate.CommonName)
		props := disconnectedProxyInterface.(disconnectedProxy)
		proxies[cn] = props.lastSeen
		return true // continue the iteration
	})
	return proxies
}
