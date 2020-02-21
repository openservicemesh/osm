package lds

import (
	"context"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

// NewLDSServer creates a new LDS server
func NewLDSServer(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec) *Server {
	return &Server{
		ctx:      ctx,
		catalog:  catalog,
		meshSpec: meshSpec,
	}
}
