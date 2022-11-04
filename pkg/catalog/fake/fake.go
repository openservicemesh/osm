package fake

import (
	"time"

	"github.com/openservicemesh/osm/pkg/catalog"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/messaging"
)

// NewFakeMeshCatalog creates a new catalog.MeshCatalog struct used for testing.
func NewFakeMeshCatalog(provider catalog.Interface) *catalog.MeshCatalog {
	stop := make(<-chan struct{})

	certManager := tresorFake.NewFake(1 * time.Hour)

	return catalog.NewMeshCatalog(provider, certManager, stop, messaging.NewBroker(stop))
}
