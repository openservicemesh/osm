package ads

import (
	"context"
	"time"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/cds"
	"github.com/openservicemesh/osm/pkg/envoy/eds"
	"github.com/openservicemesh/osm/pkg/envoy/lds"
	"github.com/openservicemesh/osm/pkg/envoy/rds"
	"github.com/openservicemesh/osm/pkg/envoy/sds"
)

// NewADSServer creates a new Aggregated Discovery Service server
func NewADSServer(ctx context.Context, meshCatalog catalog.MeshCataloger, enableDebug bool, osmNamespace string, cfg configurator.Configurator) *Server {
	server := Server{
		catalog:      meshCatalog,
		ctx:          ctx,
		xdsHandlers:  getHandlers(),
		enableDebug:  enableDebug,
		osmNamespace: osmNamespace,
		cfg:          cfg,
	}

	if enableDebug {
		server.xdsLog = make(map[certificate.CommonName]map[envoy.TypeURI][]time.Time)
	}

	return &server
}

func getHandlers() map[envoy.TypeURI]func(context.Context, catalog.MeshCataloger, *envoy.Proxy, *xds_discovery.DiscoveryRequest, configurator.Configurator) (*xds_discovery.DiscoveryResponse, error) {
	return map[envoy.TypeURI]func(context.Context, catalog.MeshCataloger, *envoy.Proxy, *xds_discovery.DiscoveryRequest, configurator.Configurator) (*xds_discovery.DiscoveryResponse, error){
		envoy.TypeEDS: eds.NewResponse,
		envoy.TypeCDS: cds.NewResponse,
		envoy.TypeRDS: rds.NewResponse,
		envoy.TypeLDS: lds.NewResponse,
		envoy.TypeSDS: sds.NewResponse,
	}
}

// DeltaAggregatedResources implements discovery.AggregatedDiscoveryServiceServer
func (s *Server) DeltaAggregatedResources(server xds_discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
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
