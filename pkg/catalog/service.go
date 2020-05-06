package catalog

import (
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

// GetServicesByServiceAccountName returns a list of services corresponding to a service account, and refreshes the cache if requested
func (mc *MeshCatalog) GetServicesByServiceAccountName(sa endpoint.NamespacedServiceAccount, refreshCache bool) []endpoint.NamespacedService {
	if refreshCache {
		mc.refreshCache()
	}
	return mc.serviceAccountToServicesCache[sa]
}
