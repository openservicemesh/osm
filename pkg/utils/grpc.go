package utils

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	maxStreams = 100000
)

// NewGrpc creates a new gRPC server
func NewGrpc(serverType string, port int, certPem string, keyPem string, rootCertPem string) (*grpc.Server, net.Listener) {
	glog.Infof("Setting up %s gRPC server...", serverType)
	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		glog.Fatalf("Could not start %s gRPC server on %s: %s", serverType, addr, err)
	}

	keepAlive := 60 * time.Second
	glog.Infof("Parameters for %s gRPC server: MaxConcurrentStreams=%d;  KeepAlive=%+v", serverType, maxStreams, keepAlive)

	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(maxStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: keepAlive,
		}),
		setupMutualTLS(false, serverType, certPem, keyPem, rootCertPem),
	}

	return grpc.NewServer(grpcOptions...), lis
}

// GrpcServe starts the gRPC server passed.
func GrpcServe(ctx context.Context, grpcServer *grpc.Server, lis net.Listener, cancel context.CancelFunc, serverType string) {
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			glog.Errorf("[grpc][%s] Error serving: %+v", serverType, err)
		}
		cancel()
	}()
	glog.Infof("[grpc][%s] Started server on: %s", serverType, lis.Addr().String())

	<-ctx.Done()

	glog.Infof("[grpc][%s] stopping %s server", serverType, serverType)

	if grpcServer != nil {
		glog.Infof("[grpc][%s] Gracefully stopping %s gRPC server", serverType, serverType)
		grpcServer.GracefulStop()
		glog.Infof("[grpc][%s] gRPC Server stopped", serverType)
	}
	glog.Infof("[grpc][%s] exiting %s gRPC server", serverType, serverType)
}
