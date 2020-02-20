package rds

import (
	"context"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/smi"
)

// NewRDSServer creates a new RDS server
func NewRDSServer(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec) *Server {
	glog.Info("[RDS] Create NewRDSServer")
	return &Server{
		ctx:      ctx,
		catalog:  catalog,
		meshSpec: meshSpec,
	}
}
