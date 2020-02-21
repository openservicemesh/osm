package sds

import (
	"context"
	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	serverName = "SDS"
)

// Server is the SDS server struct
type Server struct {
	ctx      context.Context
	catalog  catalog.MeshCataloger
	meshSpec smi.MeshSpec
}
