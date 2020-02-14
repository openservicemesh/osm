package sds

import (
	"github.com/deislabs/smc/pkg/catalog"
)

// NewSDSServer creates a new SDS server
func NewSDSServer(catalog catalog.MeshCataloger) *Server {
	return &Server{
		connectionNum: 0,
		catalog:       catalog,
		closing:       make(chan bool),
	}
}
