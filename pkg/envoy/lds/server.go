package lds

import (
	"github.com/deislabs/smc/pkg/catalog"
)

// NewLDSServer creates a new LDS server
func NewLDSServer(catalog catalog.MeshCataloger) *Server {
	return &Server{
		connectionNum: 0,
		catalog:       catalog,
		closing:       make(chan bool),
	}
}
