package cds

import (
	"github.com/deislabs/smc/pkg/catalog"
)

const (
	serverName = "CDS"
)

// NewCDSServer creates a new CDS server
func NewCDSServer(catalog catalog.MeshCataloger) *Server {
	return &Server{
		catalog: catalog,
	}
}
