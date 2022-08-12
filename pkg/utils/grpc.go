package utils

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

// Server is a construct to run gRPC servers
type Server struct {
	name     string
	cm       *certificate.Manager
	server   *grpc.Server
	certName string

	mu     sync.Mutex
	config tls.Config
}

// NewGrpc creates a new gRPC server
func NewGrpc(serverType string, port int, certName string, cm *certificate.Manager) (*Server, net.Listener, error) {
	log.Info().Msgf("Setting up %s gRPC server...", serverType)
	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error().Err(err).Msgf("Error starting %s gRPC server on %s", serverType, addr)
		return nil, nil, err
	}

	log.Debug().Msgf("Parameters for %s gRPC server: MaxConcurrentStreams=%d;  KeepAlive=%+v", serverType, maxStreams, streamKeepAliveDuration)

	s := &Server{
		name:     serverType,
		cm:       cm,
		certName: certName,
	}

	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(maxStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: streamKeepAliveDuration,
		}),
	}

	tlsConfig := tls.Config{
		InsecureSkipVerify: false,
		ServerName:         serverType,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		MinVersion:         constants.MinTLSVersion,
		GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
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
func (s *Server) GetServer() *grpc.Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.server
}

func (s *Server) setConfig() error {
	grpcCert, err := s.cm.IssueCertificate(
		s.certName,
		certificate.Internal,
		certificate.FullCNProvided())
	if err != nil {
		return err
	}

	certif, err := tls.X509KeyPair(grpcCert.GetCertificateChain(), grpcCert.GetPrivateKey())
	if err != nil {
		return fmt.Errorf("[grpc][mTLS][%s] Failed loading Certificate (%+v) and Key (%+v) PEM files", s.name, grpcCert.GetCertificateChain(), grpcCert.GetPrivateKey())
	}

	certPool := x509.NewCertPool()

	// Load the set of Root CAs
	if ok := certPool.AppendCertsFromPEM(grpcCert.GetTrustedCAs()); !ok {
		return fmt.Errorf("[grpc][mTLS][%s] Failed to append client certs", s.name)
	}

	s.mu.Lock()
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

func (s *Server) configureConfigUpdates(ctx context.Context) error {
	// listen for certificate rotation first, so we don't miss any events
	certRotationChan, unsubscribeRotation := s.cm.SubscribeRotations(s.certName)

	if err := s.setConfig(); err != nil {
		// this is a fatal error on start, we can't continue without a cert
		unsubscribeRotation()
		return err
	}

	// Handle the rotations until the context is cancelled
	go func() {
		log.Info().Str("grpc", s.name).Str("cn", s.certName).Msg("Listening for certificate rotations")
		defer unsubscribeRotation()
		for {
			select {
			case <-certRotationChan:
				log.Debug().Str("grpc", s.name).Str("cn", s.certName).Msg("Certificate rotation was initiated for grpc server")
				if err := s.setConfig(); err != nil {
					events.GenericEventRecorder().ErrorEvent(err, events.CertificateIssuanceFailure, "Error rotating the certificate for grpc server")
					continue
				}
				log.Info().Str("grpc", s.name).Str("cn", s.certName).Msg("Certificate rotated for grpc")
			case <-ctx.Done():
				log.Info().Str("grpc", s.name).Str("cn", s.certName).Msg("Stop listening for certificate rotations")
				return
			}
		}
	}()
	return nil
}

// GrpcServe starts the gRPC server passed.
func (s *Server) GrpcServe(ctx context.Context, cancel context.CancelFunc, lis net.Listener, errorCh chan interface{}) error {
	if err := s.configureConfigUpdates(ctx); err != nil {
		return err
	}

	log.Info().Msgf("[grpc][%s] Starting server on: %s", s.name, lis.Addr())
	go func() {
		if err := s.server.Serve(lis); err != nil {
			log.Error().Err(err).Msgf("[grpc][%s] Error serving gRPC request", s.name)
			if errorCh != nil {
				errorCh <- err
			}
		}
		cancel()
	}()

	go func() {
		<-ctx.Done()

		log.Info().Msgf("[grpc][%s] stopping %s server", s.name, s.name)

		if s.server != nil {
			log.Info().Msgf("[grpc][%s] Gracefully stopping %s gRPC server", s.name, s.name)
			s.server.GracefulStop()
			log.Info().Msgf("[grpc][%s] gRPC Server stopped", s.name)
		}
		log.Info().Msgf("[grpc][%s] exiting %s gRPC server", s.name, s.name)
	}()
	return nil
}
