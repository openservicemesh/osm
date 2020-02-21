package sds

import (
	"context"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

// NewSDSServer creates a new SDS server
func NewSDSServer(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec) *Server {
	return &Server{
		connectionNum: 0,
		catalog:       catalog,
		closing:       make(chan bool),
	}
}
