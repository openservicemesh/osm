package ads

import (
	"context"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/cds"
	"github.com/open-service-mesh/osm/pkg/envoy/eds"
	"github.com/open-service-mesh/osm/pkg/envoy/lds"
	"github.com/open-service-mesh/osm/pkg/envoy/rds"
	"github.com/open-service-mesh/osm/pkg/envoy/sds"
	"github.com/open-service-mesh/osm/pkg/smi"
)

// NewADSServer creates a new Aggregated Discovery Service server
func NewADSServer(ctx context.Context, meshCatalog catalog.MeshCataloger, meshSpec smi.MeshSpec, enableDebug bool, osmNamespace string) *Server {
	server := Server{
		catalog:      meshCatalog,
		ctx:          ctx,
		meshSpec:     meshSpec,
		xdsHandlers:  getHandlers(),
		enableDebug:  enableDebug,
		osmNamespace: osmNamespace,
	}

	if enableDebug {
		server.xdsLog = make(map[certificate.CommonName]map[envoy.TypeURI][]time.Time)
	}

	return &server
}

func getHandlers() map[envoy.TypeURI]func(context.Context, catalog.MeshCataloger, smi.MeshSpec, *envoy.Proxy, *xds.DiscoveryRequest, *configurator.Config) (*xds.DiscoveryResponse, error) {
	return map[envoy.TypeURI]func(context.Context, catalog.MeshCataloger, smi.MeshSpec, *envoy.Proxy, *xds.DiscoveryRequest, *configurator.Config) (*xds.DiscoveryResponse, error){
		envoy.TypeEDS: eds.NewResponse,
		envoy.TypeCDS: cds.NewResponse,
		envoy.TypeRDS: rds.NewResponse,
		envoy.TypeLDS: lds.NewResponse,
		envoy.TypeSDS: sds.NewResponse,
	}
}

// DeltaAggregatedResources implements envoy_service_discovery_v2.AggregatedDiscoveryServiceServer
func (s *Server) DeltaAggregatedResources(server envoy_service_discovery_v2.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
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
