package ads

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"sync"
	"time"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/cds"
	"github.com/openservicemesh/osm/pkg/envoy/eds"
	"github.com/openservicemesh/osm/pkg/envoy/lds"
	"github.com/openservicemesh/osm/pkg/envoy/rds"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/envoy/sds"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/workerpool"
)

const (
	// ServerType is the type identifier for the ADS server
	ServerType = "ADS"

	// workerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	workerPoolSize = 0

	maxStreams              = 100000
	streamKeepAliveDuration = 60 * time.Second
)

// NewADSServer creates a new Aggregated Discovery Service server
func NewADSServer(meshCatalog catalog.MeshCataloger, proxyRegistry *registry.ProxyRegistry, enableDebug bool, osmNamespace string, cfg configurator.Configurator, certManager certificate.Manager, kubecontroller k8s.Controller) *Server {
	server := Server{
		catalog:       meshCatalog,
		proxyRegistry: proxyRegistry,
		xdsHandlers: map[envoy.TypeURI]func(catalog.MeshCataloger, *envoy.Proxy, *xds_discovery.DiscoveryRequest, configurator.Configurator, certificate.Manager, *registry.ProxyRegistry) ([]types.Resource, error){
			envoy.TypeEDS: eds.NewResponse,
			envoy.TypeCDS: cds.NewResponse,
			envoy.TypeRDS: rds.NewResponse,
			envoy.TypeLDS: lds.NewResponse,
			envoy.TypeSDS: sds.NewResponse,
		},
		osmNamespace:   osmNamespace,
		cfg:            cfg,
		certManager:    certManager,
		xdsMapLogMutex: sync.Mutex{},
		xdsLog:         make(map[certificate.CommonName]map[envoy.TypeURI][]time.Time),
		workqueues:     workerpool.NewWorkerPool(workerPoolSize),
		kubecontroller: kubecontroller,
	}

	return &server
}

// withXdsLogMutex helper to run code that touches xdsLog map, to protect by mutex
func (s *Server) withXdsLogMutex(f func()) {
	s.xdsMapLogMutex.Lock()
	defer s.xdsMapLogMutex.Unlock()
	f()
}

// Start starts the ADS server
func (s *Server) Start(ctx context.Context, cancel context.CancelFunc, port int, adsCert certificate.Certificater) error {
	grpcServer, lis, err := NewGrpc(ServerType, port, adsCert.GetCertificateChain(), adsCert.GetPrivateKey(), adsCert.GetIssuingCA())
	if err != nil {
		log.Error().Err(err).Msg("Error starting ADS server")
		return err
	}

	xds_discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, s)
	go GrpcServe(ctx, grpcServer, lis, cancel, ServerType, nil)
	s.ready = true

	return nil
}

// DeltaAggregatedResources implements discovery.AggregatedDiscoveryServiceServer
func (s *Server) DeltaAggregatedResources(xds_discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	panic("NotImplemented")
}

// NewGrpc creates a new gRPC server
func NewGrpc(serverType string, port int, certPem, keyPem, rootCertPem []byte) (*grpc.Server, net.Listener, error) {
	log.Info().Msgf("Setting up %s gRPC server...", serverType)
	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error().Err(err).Msgf("Error starting %s gRPC server on %s", serverType, addr)
		return nil, nil, err
	}

	log.Debug().Msgf("Parameters for %s gRPC server: MaxConcurrentStreams=%d;  KeepAlive=%+v", serverType, maxStreams, streamKeepAliveDuration)

	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(maxStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: streamKeepAliveDuration,
		}),
	}

	mutualTLS, err := setupMutualTLS(false, serverType, certPem, keyPem, rootCertPem)
	if err != nil {
		log.Error().Err(err).Msg("Error setting up mutual tls for GRPC server")
		return nil, nil, err
	}
	grpcOptions = append(grpcOptions, mutualTLS)

	return grpc.NewServer(grpcOptions...), lis, nil
}

// GrpcServe starts the gRPC server passed.
func GrpcServe(ctx context.Context, grpcServer *grpc.Server, lis net.Listener, cancel context.CancelFunc, serverType string, errorCh chan interface{}) {
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Error().Err(err).Msgf("[grpc][%s] Error serving gRPC request", serverType)
			if errorCh != nil {
				errorCh <- err
			}
		}
		cancel()
	}()
	log.Info().Msgf("[grpc][%s] Started server on: %s", serverType, lis.Addr())

	<-ctx.Done()

	log.Info().Msgf("[grpc][%s] stopping %s server", serverType, serverType)

	if grpcServer != nil {
		log.Info().Msgf("[grpc][%s] Gracefully stopping %s gRPC server", serverType, serverType)
		grpcServer.GracefulStop()
		log.Info().Msgf("[grpc][%s] gRPC Server stopped", serverType)
	}
	log.Info().Msgf("[grpc][%s] exiting %s gRPC server", serverType, serverType)
}

func setupMutualTLS(insecure bool, serverName string, certPem []byte, keyPem []byte, ca []byte) (grpc.ServerOption, error) {
	certif, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil, errors.Errorf("[grpc][mTLS][%s] Failed loading Certificate (%+v) and Key (%+v) PEM files", serverName, certPem, keyPem)
	}

	certPool := x509.NewCertPool()

	// Load the set of Root CAs
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.Errorf("[grpc][mTLS][%s] Failed to append client certs", serverName)
	}

	// #nosec G402
	tlsConfig := tls.Config{
		InsecureSkipVerify: insecure,
		ServerName:         serverName,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		Certificates:       []tls.Certificate{certif},
		ClientCAs:          certPool,
	}
	return grpc.Creds(credentials.NewTLS(&tlsConfig)), nil
}
