package lds

import "github.com/deislabs/smc/pkg/catalog"

// Server is the ClusterDiscoveryService server struct
type Server struct {
	connectionNum int
	catalog       catalog.MeshCataloger
	closing       chan bool
}
