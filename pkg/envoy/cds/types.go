package cds

import (
	"context"
	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	serverName = "CDS"
)

//Server implements the Envoy xDS Cluster Discovery Services
type Server struct {
	ctx      context.Context
	catalog  catalog.MeshCataloger
	meshSpec smi.MeshSpec
}
