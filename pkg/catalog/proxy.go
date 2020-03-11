package catalog

import (
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/golang/glog"
)

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (sc *MeshCatalog) RegisterProxy(p *envoy.Proxy) {
	sc.connectedProxies.Add(p)
	glog.Infof("Registered new proxy: CN=%v, ip=%v", p.GetCommonName(), p.GetIP())
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (sc *MeshCatalog) UnregisterProxy(p *envoy.Proxy) {
	sc.connectedProxies.Remove(p)
	glog.Infof("Unregistered p: CN=%v, ip=%v", p.GetCommonName(), p.GetIP())
}
