package registry

import (
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// ListConnectedProxies lists the Envoy proxies already connected and the time they first connected.
func (pr *ProxyRegistry) ListConnectedProxies() map[certificate.CommonName]*envoy.Proxy {
	proxies := make(map[certificate.CommonName]*envoy.Proxy)
	pr.connectedProxies.Range(func(cnIface, propsIface interface{}) bool {
		cn := cnIface.(certificate.CommonName)
		props := propsIface.(connectedProxy)
		proxies[cn] = props.proxy
		return true // continue the iteration
	})
	return proxies
}
