package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/deislabs/smc/pkg/utils"

	envoyControlPlane "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	sdsServer "github.com/deislabs/smc/pkg/envoy/sds"
)

const (
	serverType = "SDS"
)

var (
	flags         = pflag.NewFlagSet(`diplomat-sds`, pflag.ExitOnError)
	keysDirectory = flags.String("keys-directory", "", "Directory where the keys are stored")
	verbosity     = flags.Int("verbosity", 1, "Set logging verbosity level")
	port          = flags.Int("port", 15123, "Services Discovery Services port number. (Default: 15123)")
)

func main() {
	defer glog.Flush()
	parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer, lis := utils.NewGrpc(serverType, *port)
	sds := sdsServer.NewSDSServer(keysDirectory)
	envoyControlPlane.RegisterSecretDiscoveryServiceServer(grpcServer, sds)
	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	glog.Info("Goodbye!")
}

func parseFlags() {
	// TODO(draychev): consolidate parseFlags - shared between sds.go and eds.go
	if err := flags.Parse(os.Args); err != nil {
		glog.Error(err)
	}
	_ = flag.CommandLine.Parse([]string{})
	_ = flag.Lookup("v").Value.Set(fmt.Sprintf("%d", *verbosity))
	_ = flag.Lookup("logtostderr").Value.Set("true")
}
