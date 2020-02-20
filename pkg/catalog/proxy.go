package catalog

import (
	"net"

	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/golang/glog"
)

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (sc *MeshCatalog) RegisterProxy(cn certificate.CommonName, ip net.IP) *envoy.Proxy {
	proxy := envoy.NewProxy(cn, ip)
	sc.connectedProxies.Add(proxy)
	glog.Infof("Registered new proxy: CN=%v, ip=%v", proxy.GetCommonName(), proxy.GetIP())
	return proxy
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (sc *MeshCatalog) UnregisterProxy(proxy *envoy.Proxy) error {
	sc.connectedProxies.Remove(proxy)
	glog.Infof("Unregistered proxy: CN=%v, ip=%v", proxy.GetCommonName(), proxy.GetIP())
	return nil
}
