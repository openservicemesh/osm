package catalog

import (
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/envoy"
)

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (sc *MeshCatalog) RegisterProxy(proxy envoy.Proxyer) {
	glog.Infof("[catalog] Register proxy %s", proxy.GetCommonName())
	sc.Lock()
	sc.connectedProxies = append(sc.connectedProxies, proxy)
	defer sc.Unlock()
}
