package cds

import (
	"github.com/deislabs/smc/pkg/catalog"
)

//Server implements the Envoy xDS Cluster Discovery Services
type Server struct {
	catalog catalog.MeshCataloger
}
