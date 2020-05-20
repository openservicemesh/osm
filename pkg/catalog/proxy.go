package catalog

import (
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

// ExpectProxy catalogs the fact that a certificate was issued for an Envoy proxy and this is expected to connect to XDS.
func (mc *MeshCatalog) ExpectProxy(cn certificate.CommonName) {
	mc.expectedProxies[cn] = expectedProxy{time.Now()}
}

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (mc *MeshCatalog) RegisterProxy(p *envoy.Proxy) {
	mc.connectedProxies[p.CommonName] = connectedProxy{p, time.Now()}
	log.Info().Msgf("Registered new proxy: CN=%v, ip=%v", p.GetCommonName(), p.GetIP())
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (mc *MeshCatalog) UnregisterProxy(p *envoy.Proxy) {
	delete(mc.connectedProxies, p.CommonName)
	log.Info().Msgf("Unregistered proxy: CN=%v, ip=%v", p.GetCommonName(), p.GetIP())
}
