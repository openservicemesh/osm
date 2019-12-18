package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	envoyControlPlane "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/deislabs/smc/cmd"
	sdsServer "github.com/deislabs/smc/pkg/envoy/sds"
)

const (
	serverType = "SDS"
	port       = 15123

	verbosityFlag = "verbosity"
)

var (
	flags         = pflag.NewFlagSet(`diplomat-sds`, pflag.ExitOnError)
	keysDirectory = flags.String("keys-directory", "", "Directory where the keys are stored")
	verbosity     = flags.Int(verbosityFlag, 1, "Set logging verbosity level")
)

func main() {
	defer glog.Flush()
	parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer, lis := cmd.NewGrpc(serverType, port)
	sds := sdsServer.NewSDSServer(keysDirectory)
	envoyControlPlane.RegisterSecretDiscoveryServiceServer(grpcServer, sds)
	cmd.GrpcServe(ctx, grpcServer, lis, cancel, serverType)
}

func parseFlags() {
	if err := flags.Parse(os.Args); err != nil {
		glog.Error(err)
	}
	_ = flag.CommandLine.Parse([]string{})
	_ = flag.Lookup("v").Value.Set(fmt.Sprintf("%d", *verbosity))
	_ = flag.Lookup("logtostderr").Value.Set("true")
}
