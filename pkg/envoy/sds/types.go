package sds

import (
	"github.com/deislabs/smc/pkg/catalog"
)

// Server is the SDS server struct
type Server struct {
	connectionNum int
	catalog       catalog.MeshCataloger
	closing       chan bool
}
