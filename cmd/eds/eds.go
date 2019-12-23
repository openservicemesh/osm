package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/eapache/channels"
	envoyControlPlane "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/deislabs/smc/cmd"
	edsServer "github.com/deislabs/smc/pkg/envoy/eds"
)

const (
	serverType = "EDS"
	port       = 15124

	verbosityFlag    = "verbosity"
	defaultNamespace = "default"

	maxAuthRetryCount = 10
	retryPause        = 10 * time.Second
)

var (
	flags          = pflag.NewFlagSet(`diplomat-edsServer`, pflag.ExitOnError)
	kubeConfigFile = flags.String("kubeconfig", "", "Path to Kubernetes config file.")
	azureAuthFile  = flags.String("azureAuthFile", "", "Path to Azure Auth File")
	resourceGroup  = flags.String("resource-group", "", "Azure Resource Group")
	subscriptionID = flags.String("subscriptionID", "", "Azure Subscription")
	verbosity      = flags.Int(verbosityFlag, 1, "Set logging verbosity level")
	namespace      = flags.String("namespace", "default", "Kubernetes namespace to watch.")
)

func main() {
	defer glog.Flush()
	parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// This channel will be read by ServiceName Mesh Controller, and written to by the compute and SMI observers.
	// This is a signalling mechanism to notify SMC of a service mesh topology change,
	// which would trigger Envoy updates.
	announceChan := channels.NewRingChannel(1024)

	computeProviders, meshSpec, serviceCatalog := setupClients(announceChan)
	grpcServer, lis := cmd.NewGrpc(serverType, port)
	eds := edsServer.NewEDSServer(ctx, computeProviders, serviceCatalog, meshSpec, announceChan)
	envoyControlPlane.RegisterEndpointDiscoveryServiceServer(grpcServer, eds)
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

func getNamespaces() []string {
	var namespaces []string
	if namespace == nil {
		defaultNS := defaultNamespace
		namespaces = []string{defaultNS}
	}
	namespaces = []string{*namespace}
	glog.Infof("Observing namespaces: %s", strings.Join(namespaces, ","))
	return namespaces
}
