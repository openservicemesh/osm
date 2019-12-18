package cmd

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

func NewGrpc(serverType string, port int) (*grpc.Server, net.Listener) {
	glog.Infof("Setting up %s gRPC server...", serverType)
	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		glog.Fatalf("Could not start %s gRPC server on %s: %s", serverType, addr, err)
	}

	keepAlive := 60 * time.Second
	glog.Infof("Parameters for %s gRPC server: MaxConcurrentStreams=%d;  KeepAlive=%+v", serverType, maxStreams, keepAlive)

	// TODO(draychev): Use TLS to protect gRPC connections
	grpcOptions := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(maxStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: keepAlive,
		}),
	}

	return grpc.NewServer(grpcOptions...), lis
}

func GrpcServe(grpcServer *grpc.Server, lis net.Listener, ctx context.Context, cancel context.CancelFunc, serverType string) {
	go func() {
		err := grpcServer.Serve(lis)
		glog.Error("error in %s gRPC server", serverType, err)
		cancel()
	}()
	glog.Infof("Started %s on: %s", serverType, lis.Addr().String())

	<-ctx.Done()
	glog.Info("stopping %s server", serverType)

	if grpcServer != nil {
		glog.Info("gracefully stopping %s gRPC server", serverType)
		grpcServer.GracefulStop()
		glog.Info("%s gRPC Server stopped", serverType)
	}
	glog.Info("exiting %s gRPC server", serverType)
}
