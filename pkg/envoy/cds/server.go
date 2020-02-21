package cds

import (
	"context"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	serverName = "CDS"
)

// NewCDSServer creates a new CDS server
func NewCDSServer(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec) *Server {
	return &Server{
		catalog: catalog,
	}
}
