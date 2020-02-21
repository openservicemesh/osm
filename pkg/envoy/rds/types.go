package rds

import (
	"context"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	serverName = "RDS"
)

// Server implements the Envoy xDS Endpoint Discovery Services
type Server struct {
	ctx      context.Context
	catalog  catalog.MeshCataloger
	meshSpec smi.MeshSpec
}
