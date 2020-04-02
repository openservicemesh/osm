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
func NewGrpc(serverType string, port int, certPem string, keyPem string, rootCertPem string) (*grpc.Server, net.Listener) {
	log.Info().Msgf("Setting up %s gRPC server...", serverType)
	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not start %s gRPC server on %s", serverType, addr)
	}

	log.Info().Msgf("Parameters for %s gRPC server: MaxConcurrentStreams=%d;  KeepAlive=%+v", serverType, maxStreams, streamKeepAliveDuration)

	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(maxStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: streamKeepAliveDuration,
		}),
		setupMutualTLS(false, serverType, certPem, keyPem, rootCertPem),
	}

	return grpc.NewServer(grpcOptions...), lis
}

// GrpcServe starts the gRPC server passed.
func GrpcServe(ctx context.Context, grpcServer *grpc.Server, lis net.Listener, cancel context.CancelFunc, serverType string) {
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Error().Err(err).Msgf("[grpc][%s] Error serving gRPC request", serverType)
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
