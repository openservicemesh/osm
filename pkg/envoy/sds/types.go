package sds

import (
	"github.com/deislabs/smc/pkg/catalog"
)

// Server is the SDS server struct
type Server struct {
	lastVersion   uint64
	lastNonce     string
	connectionNum int
	catalog       catalog.MeshCataloger

	// close channel.
	closing chan bool
}
