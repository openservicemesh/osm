package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/constants"
	"github.com/deislabs/smc/pkg/envoy/sds"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	serverType = "SDS"
)

var (
	flags          = pflag.NewFlagSet(`diplomat-sds`, pflag.ExitOnError)
	kubeConfigFile = flags.String("kubeconfig", "", "Path to Kubernetes config file.")
	verbosity      = flags.Int("verbosity", 1, "Set logging verbosity level")
	port           = flags.Int("port", 15123, "Secrets Discovery Service port number. (Default: 15123)")
	namespace      = flags.String("namespace", "default", "Kubernetes namespace to watch for SMI Spec.")
	certPem        = flags.String("certpem", "", fmt.Sprintf("Full path to the %s Certificate PEM file", serverType))
	keyPem         = flags.String("keypem", "", fmt.Sprintf("Full path to the %s Key PEM file", serverType))
	rootCertPem    = flags.String("rootcertpem", "", "Full path to the Root Certificate PEM file")
)

func main() {
	defer glog.Flush()
	parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// SMI Informers will write to this channel when they notice changes.
	// This channel will be consumed by the ServiceName Mesh Controller.
	// This is a signalling mechanism to notify SMC of a service mesh spec change which triggers Envoy updates.
	announcements := make(chan interface{})

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		glog.Fatalf("[SDS] Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", *kubeConfigFile, err)
	}

	observeNamespaces := getNamespaces()

	stop := make(chan struct{})
	meshSpecClient := smi.NewMeshSpecClient(kubeConfig, observeNamespaces, announcements, stop)
	certManager := certificate.NewManager(stop)
	meshCatalog := catalog.NewMeshCatalog(meshSpecClient, certManager, stop)
	sdsServer := sds.NewSDSServer(meshCatalog)

	grpcServer, lis := utils.NewGrpc(serverType, *port, *certPem, *keyPem, *rootCertPem)
	xds.RegisterSecretDiscoveryServiceServer(grpcServer, sdsServer)
	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	close(stop)
	glog.Info("[SDS] Goodbye!")
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

func getNamespaces() []string {
	var namespaces []string
	if namespace == nil {
		defaultNS := constants.DefaultKubeNamespace
		namespaces = []string{defaultNS}
	} else {
		namespaces = []string{*namespace}
	}
	glog.Infof("[SDS] Observing namespaces: %s", strings.Join(namespaces, ","))
	return namespaces
}
