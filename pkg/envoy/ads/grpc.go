package ads

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
)

const (
	maxStreams              = 100000
	streamKeepAliveDuration = 60 * time.Second
)

// GRPCServer is a construct to run gRPC servers
type GRPCServer struct {
	name           string
	cm             *certificate.Manager
	server         *grpc.Server
	certCommonName string

	mu     sync.Mutex
	config tls.Config
}

// NewGrpc creates a new gRPC server
func NewGrpc(serverName string, port int, certCommonName string, cm *certificate.Manager) (*GRPCServer, net.Listener, error) {
	log.Info().Msgf("Setting up %s gRPC server...", serverName)
	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error().Err(err).Msgf("Error starting %s gRPC server on %s", serverName, addr)
		return nil, nil, err
	}

	log.Debug().Msgf("Parameters for %s gRPC server: MaxConcurrentStreams=%d;  KeepAlive=%+v", serverName, maxStreams, streamKeepAliveDuration)

	s := &GRPCServer{
		name:           serverName,
		cm:             cm,
		certCommonName: certCommonName,
	}

	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(maxStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: streamKeepAliveDuration,
		}),
	}

	// #nosec G402: TLS MinVersion too low
	tlsConfig := tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.name,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		MinVersion:         constants.MinTLSVersion,
		GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
			// use lock to prevent concurrent updates and reads to the tls config
			s.mu.Lock()
			defer s.mu.Unlock()
			return &s.config, nil
		},
	}
	mutualTLS := grpc.Creds(credentials.NewTLS(&tlsConfig))
	grpcOptions = append(grpcOptions, mutualTLS)

	s.server = grpc.NewServer(grpcOptions...)
	return s, lis, nil
}

// GetServer returns the gRPC server
func (s *GRPCServer) GetServer() *grpc.Server {
	return s.server
}

func (s *GRPCServer) updateConfig() error {
	grpcCert, err := s.cm.IssueCertificate(
		s.certCommonName,
		certificate.Internal)
	if err != nil {
		return err
	}

	certif, err := tls.X509KeyPair(grpcCert.GetCertificateChain(), grpcCert.GetPrivateKey())
	if err != nil {
		return fmt.Errorf("failed loading Certificate (%+v) and Key (%+v) PEM files", grpcCert.GetCertificateChain(), grpcCert.GetPrivateKey())
	}

	certPool := x509.NewCertPool()

	// Load the set of Root CAs
	if ok := certPool.AppendCertsFromPEM(grpcCert.GetTrustedCAs()); !ok {
		return fmt.Errorf("failed to append client certs")
	}

	// use lock to prevent concurrent updates and reads to the tls config
	s.mu.Lock()
	// #nosec G402: TLS MinVersion too low
	s.config = tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.name,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		Certificates:       []tls.Certificate{certif},
		ClientCAs:          certPool,
		MinVersion:         constants.MinTLSVersion,
	}
	s.mu.Unlock()

	return nil
}

func (s *GRPCServer) watchCertRotations(ctx context.Context) error {
	// listen for certificate rotation first, so we don't miss any events
	certRotationChan, unsubscribeRotation := s.cm.SubscribeRotations(s.certCommonName)

	// initial updateConfig call creates the server certificate
	if err := s.updateConfig(); err != nil {
		// this is a fatal error on start, we can't continue without a cert
		unsubscribeRotation()
		return err
	}

	// Handle the rotations until the context is cancelled
	go func() {
		log.Info().Str("grpc", s.name).Str("cn", s.certCommonName).Msg("Listening for certificate rotations")
		defer unsubscribeRotation()
		for {
			select {
			case <-certRotationChan:
				log.Debug().Str("grpc", s.name).Str("cn", s.certCommonName).Msg("Certificate rotation was initiated for grpc server")
				if err := s.updateConfig(); err != nil {
					events.GenericEventRecorder().ErrorEvent(err, events.CertificateIssuanceFailure, "Error rotating the certificate for grpc server")
					continue
				}
				log.Info().Str("grpc", s.name).Str("cn", s.certCommonName).Msg("Certificate rotated for grpc")
			case <-ctx.Done():
				log.Info().Str("grpc", s.name).Str("cn", s.certCommonName).Msg("Stop listening for certificate rotations")
				return
			}
		}
	}()
	return nil
}

// GrpcServe starts the gRPC server passed.
func (s *GRPCServer) GrpcServe(ctx context.Context, cancel context.CancelFunc, lis net.Listener, errorCh chan interface{}) error {
	if err := s.watchCertRotations(ctx); err != nil {
		return err
	}

	log.Info().Str("grpc", s.name).Msgf("Starting server on: %s", lis.Addr())
	go func() {
		if err := s.server.Serve(lis); err != nil {
			log.Error().Str("grpc", s.name).Err(err).Msg("error serving gRPC request")
			if errorCh != nil {
				errorCh <- err
			}
		}
		cancel()
	}()

	go func() {
		<-ctx.Done()

		log.Info().Str("grpc", s.name).Msg("gracefully stopping gRPC server")
		s.server.GracefulStop()
		log.Info().Str("grpc", s.name).Msgf("exiting gRPC server")
	}()
	return nil
}
