package catalog

import (
	"github.com/deislabs/smc/pkg/endpoint"
)

// GetServicesByServiceAccountName returns a list of services corresponding to a service account, and refreshes the cache if requested
func (mc *MeshCatalog) GetServicesByServiceAccountName(sa endpoint.ServiceAccount, refreshCache bool) []endpoint.ServiceName {
	if refreshCache {
		mc.refreshCache()
	}
	return mc.serviceAccountsCache[sa]
}
