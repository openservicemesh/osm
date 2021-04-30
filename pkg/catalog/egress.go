package catalog

import (
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// GetEgressTrafficPolicy returns the Egress traffic policy associated with the given service identity
func (mc *MeshCatalog) GetEgressTrafficPolicy(serviceIdentity identity.ServiceIdentity) (*trafficpolicy.EgressTrafficPolicy, error) {
	// TODO(#3045): Implement egres policies
	return nil, nil
}
