package lds

import (
	"context"
	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	serverName = "LDS"
)

// Server is the ClusterDiscoveryService server struct
type Server struct {
	ctx      context.Context
	catalog  catalog.MeshCataloger
	meshSpec smi.MeshSpec
}
