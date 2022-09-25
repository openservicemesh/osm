package catalog

import (
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/compute"
)

// NewMeshCatalog creates a new service catalog
func NewMeshCatalog(computeInterface compute.Interface, certManager *certificate.Manager) *MeshCatalog {
	return &MeshCatalog{
		Interface:   computeInterface,
		certManager: certManager,
	}
}
