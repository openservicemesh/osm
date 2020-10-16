package utils

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	maxStreams              = 100000
	streamKeepAliveDuration = 60 * time.Second
)

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
	log.Info().Msgf("[grpc][%s] Started server on: %s", serverType, lis.Addr().String())

	<-ctx.Done()

	log.Info().Msgf("[grpc][%s] stopping %s server", serverType, serverType)

	if grpcServer != nil {
		log.Info().Msgf("[grpc][%s] Gracefully stopping %s gRPC server", serverType, serverType)
		grpcServer.GracefulStop()
		log.Info().Msgf("[grpc][%s] gRPC Server stopped", serverType)
	}
	log.Info().Msgf("[grpc][%s] exiting %s gRPC server", serverType, serverType)
}
