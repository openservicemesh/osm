package catalog

import (
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

// ExpectProxy catalogs the fact that a certificate was issued for an Envoy proxy and this is expected to connect to XDS.
func (mc *MeshCatalog) ExpectProxy(cn certificate.CommonName) {
	mc.expectedProxiesLock.Lock()
	mc.expectedProxies[cn] = expectedProxy{
		certificateIssuedAt: time.Now(),
	}
	mc.expectedProxiesLock.Unlock()
}

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (mc *MeshCatalog) RegisterProxy(p *envoy.Proxy) {
	mc.connectedProxiesLock.Lock()
	mc.connectedProxies[p.CommonName] = connectedProxy{
		proxy:       p,
		connectedAt: time.Now(),
	}
	mc.connectedProxiesLock.Unlock()
	log.Info().Msgf("Registered new proxy: CN=%v, ip=%v", p.GetCommonName(), p.GetIP())
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (mc *MeshCatalog) UnregisterProxy(p *envoy.Proxy) {
	mc.connectedProxiesLock.Lock()
	delete(mc.connectedProxies, p.CommonName)
	mc.connectedProxiesLock.Unlock()

	mc.disconnectedProxiesLock.Lock()
	mc.disconnectedProxies[p.CommonName] = disconnectedProxy{
		lastSeen: time.Now(),
	}
	mc.disconnectedProxiesLock.Unlock()

	log.Info().Msgf("Unregistered proxy: CN=%v, ip=%v", p.GetCommonName(), p.GetIP())
}
