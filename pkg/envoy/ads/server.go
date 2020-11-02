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
	"github.com/openservicemesh/osm/pkg/utils"
)

// ServerType is the type identifier for the ADS server
const ServerType = "ADS"

// NewADSServer creates a new Aggregated Discovery Service server
func NewADSServer(meshCatalog catalog.MeshCataloger, enableDebug bool, osmNamespace string, cfg configurator.Configurator, certManager certificate.Manager) *Server {
	server := Server{
		catalog: meshCatalog,
		xdsHandlers: map[envoy.TypeURI]func(catalog.MeshCataloger, *envoy.Proxy, *xds_discovery.DiscoveryRequest, configurator.Configurator, certificate.Manager) (*xds_discovery.DiscoveryResponse, error){
			envoy.TypeEDS: eds.NewResponse,
			envoy.TypeCDS: cds.NewResponse,
			envoy.TypeRDS: rds.NewResponse,
			envoy.TypeLDS: lds.NewResponse,
			envoy.TypeSDS: sds.NewResponse,
		},
		enableDebug:  enableDebug,
		osmNamespace: osmNamespace,
		cfg:          cfg,
		certManager:  certManager,
	}

	if enableDebug {
		server.xdsLog = make(map[certificate.CommonName]map[envoy.TypeURI][]time.Time)
	}

	return &server
}

// Start starts the ADS server
func (s *Server) Start(ctx context.Context, cancel context.CancelFunc, port int, adsCert certificate.Certificater) error {
	grpcServer, lis, err := utils.NewGrpc(ServerType, port, adsCert.GetCertificateChain(), adsCert.GetPrivateKey(), adsCert.GetIssuingCA())
	if err != nil {
		log.Error().Err(err).Msg("Error starting ADS server")
		return err
	}

	xds_discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, s)
	go utils.GrpcServe(ctx, grpcServer, lis, cancel, ServerType, nil)
	s.ready = true

	return nil
}

// DeltaAggregatedResources implements discovery.AggregatedDiscoveryServiceServer
func (s *Server) DeltaAggregatedResources(xds_discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	panic("NotImplemented")
}
