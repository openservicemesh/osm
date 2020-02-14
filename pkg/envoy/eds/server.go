package eds

import (
	"context"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

// NewEDSServer creates a new EDS server
func NewEDSServer(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec) *Server {
	return &Server{
		ctx:      ctx,
		catalog:  catalog,
		meshSpec: meshSpec,
	}
}
