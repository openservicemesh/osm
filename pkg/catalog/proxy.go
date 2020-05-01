package catalog

import (
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

// ExpectProxy catalogs the fact that a certificate was issued for an Envoy proxy and this is expected to connect to XDS.
func (sc *MeshCatalog) ExpectProxy(cn certificate.CommonName) {
	sc.expectedProxies[cn] = expectedProxy{time.Now()}
}

// RegisterProxy implements MeshCatalog and registers a newly connected proxy.
func (sc *MeshCatalog) RegisterProxy(p *envoy.Proxy) {
	sc.connectedProxies[p.CommonName] = connectedProxy{p, time.Now()}
	log.Info().Msgf("Registered new proxy: CN=%v, ip=%v", p.GetCommonName(), p.GetIP())
}

// UnregisterProxy unregisters the given proxy from the catalog.
func (sc *MeshCatalog) UnregisterProxy(p *envoy.Proxy) {
	delete(sc.connectedProxies, p.CommonName)
	log.Info().Msgf("Unregistered p: CN=%v, ip=%v", p.GetCommonName(), p.GetIP())
}
