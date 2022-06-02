package registry

import (
	"github.com/openservicemesh/osm/pkg/envoy"
)

// ListConnectedProxies lists the Envoy proxies already connected and the time they first connected.
func (pr *ProxyRegistry) ListConnectedProxies() map[string]*envoy.Proxy {
	proxies := make(map[string]*envoy.Proxy)
	pr.connectedProxies.Range(func(keyIface, propsIface interface{}) bool {
		key := keyIface.(string)
		props := propsIface.(connectedProxy)
		proxies[key] = props.proxy
		return true // continue the iteration
	})
	return proxies
}
