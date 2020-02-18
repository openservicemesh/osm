package ads

import (
	"context"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy"
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
	s := Server{
		catalog: meshCatalog,

		cdsServer: cds.NewCDSServer(meshCatalog),
		rdsServer: rds.NewRDSServer(ctx, meshCatalog, meshSpec),
		edsServer: eds.NewEDSServer(ctx, meshCatalog, meshSpec),
		ldsServer: lds.NewLDSServer(meshCatalog),
		sdsServer: sds.NewSDSServer(meshCatalog),
	}
	s.xdsHandlers = map[envoy.TypeURI]func(*envoy.Proxy) (*envoy_api_v2.DiscoveryResponse, error){
		envoy.TypeEDS: s.edsServer.NewEndpointDiscoveryResponse,
		envoy.TypeCDS: s.cdsServer.NewClusterDiscoveryResponse,
		envoy.TypeRDS: s.rdsServer.NewRouteDiscoveryResponse,
		envoy.TypeLDS: s.ldsServer.NewListenerDiscoveryResponse,
		envoy.TypeSDS: s.sdsServer.NewSecretDiscoveryResponse,
	}
	return &s
}

// DeltaAggregatedResources implements xds.AggregatedDiscoveryServiceServer
func (s *Server) DeltaAggregatedResources(server xds.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	panic("NotImplemented")
}

func (s *Server) Liveness() bool {
	return true
}

func (s *Server) Readiness() bool {
	return true
}
