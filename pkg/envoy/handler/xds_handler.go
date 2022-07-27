package handler

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/openservicemesh/osm/pkg/envoy"
)

type XDSHandler interface {
	Respond() ([]types.Resource, error)

	// SetMeshCataloger(catalog.MeshCataloger)
	SetProxy(*envoy.Proxy)
	SetDiscoveryRequest(*xds_discovery.DiscoveryRequest)
	// SetConfigurator(configurator.Configurator)
	// SetCertManager(*certificate.Manager)
	// SetProxyRegistry(*registry.ProxyRegistry)
}
