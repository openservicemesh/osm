package catalog

import (
	"github.com/open-service-mesh/osm/pkg/service"
)

// GetServicesByServiceAccountName returns a list of services corresponding to a service account, and refreshes the cache if requested
func (mc *MeshCatalog) GetServicesByServiceAccountName(sa service.NamespacedServiceAccount, refreshCache bool) []service.NamespacedService {
	if refreshCache {
		mc.refreshCache()
	}
	return mc.serviceAccountToServicesCache[sa]
}
