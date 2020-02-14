package ads

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy/cds"
	"github.com/deislabs/smc/pkg/envoy/eds"
	"github.com/deislabs/smc/pkg/envoy/lds"
	"github.com/deislabs/smc/pkg/envoy/rds"
	"github.com/deislabs/smc/pkg/envoy/sds"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	serverName = "ADS"
)

// NewADSServer creates a new CDS server
func NewADSServer(ctx context.Context, meshCatalog catalog.MeshCataloger, meshSpec smi.MeshSpec) *Server {
	return &Server{
		catalog: meshCatalog,

		cdsServer: cds.NewCDSServer(meshCatalog),
		rdsServer: rds.NewRDSServer(ctx, meshCatalog, meshSpec),
		edsServer: eds.NewEDSServer(ctx, meshCatalog, meshSpec),
		ldsServer: lds.NewLDSServer(meshCatalog),
		sdsServer: sds.NewSDSServer(meshCatalog),
	}
}

// DeltaAggregatedResources implements xds.AggregatedDiscoveryServiceServer
func (s *Server) DeltaAggregatedResources(server xds.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	panic("NotImplemented")
}
