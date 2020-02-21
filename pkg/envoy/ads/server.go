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
	return &Server{
		catalog: meshCatalog,
		xdsHandlers: map[envoy.TypeURI]func(*envoy.Proxy) (*envoy_api_v2.DiscoveryResponse, error){
			envoy.TypeEDS: eds.NewEDSServer(ctx, meshCatalog, meshSpec).NewDiscoveryResponse,
			envoy.TypeCDS: cds.NewCDSServer(ctx, meshCatalog, meshSpec).NewDiscoveryResponse,
			envoy.TypeRDS: rds.NewRDSServer(ctx, meshCatalog, meshSpec).NewDiscoveryResponse,
			envoy.TypeLDS: lds.NewLDSServer(ctx, meshCatalog, meshSpec).NewDiscoveryResponse,
			envoy.TypeSDS: sds.NewSDSServer(ctx, meshCatalog, meshSpec).NewDiscoveryResponse,
		},
	}
}

// DeltaAggregatedResources implements xds.AggregatedDiscoveryServiceServer
func (s *Server) DeltaAggregatedResources(server xds.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	panic("NotImplemented")
}

// Liveness is the Kubernetes liveness probe handler.
func (s *Server) Liveness() bool {
	// TODO(draychev): implement
	return true
}

// Readiness is the Kubernetes readiness probe handler.
func (s *Server) Readiness() bool {
	// TODO(draychev): implement
	return true
}
