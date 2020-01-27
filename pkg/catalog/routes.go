package catalog

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/smi"
)

// ListTrafficRoutes constructs a DiscoveryResponse with all traffic routes the given Envoy proxy should be aware of.
func (sc *MeshCatalog) ListTrafficRoutes(clientID smi.ClientIdentity) (resp *v2.DiscoveryResponse, err error) {
	glog.Info("[catalog] Listing Endpoints for client: ", clientID)
	// TODO(draychev): implement
	panic("NotImplemented")
}
