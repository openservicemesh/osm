package eds

import (
	"context"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

// Server implements the Envoy xDS Endpoint Discovery Services
type Server struct {
	ctx      context.Context
	catalog  catalog.MeshCataloger
	meshSpec smi.MeshSpec

	lastVersion uint64
	lastNonce   string
}
