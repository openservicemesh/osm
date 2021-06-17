package catalog

import (
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/witesand"
)
func (mc *MeshCatalog) GetProvider(ident string) endpoint.Provider {
	for _, ep := range mc.endpointsProviders {
		if ep.GetID() == ident {
			return ep
		}
	}
	return nil
}

func (mc *MeshCatalog) GetWitesandCataloger() witesand.WitesandCataloger {
	return mc.witesandCatalog
}


