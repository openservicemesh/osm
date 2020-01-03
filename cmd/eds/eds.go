package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/deislabs/smc/pkg/utils"

	"github.com/eapache/channels"
	envoyControlPlane "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/deislabs/smc/pkg/catalog"
	edsServer "github.com/deislabs/smc/pkg/envoy/eds"
	"github.com/deislabs/smc/pkg/providers/azure"
	"github.com/deislabs/smc/pkg/providers/kube"
)

const (
	serverType = "EDS"

	defaultNamespace = "default"

	maxAuthRetryCount = 10
	retryPause        = 10 * time.Second
)

var (
	flags          = pflag.NewFlagSet(`diplomat-edsServer`, pflag.ExitOnError)
	kubeConfigFile = flags.String("kubeconfig", "", "Path to Kubernetes config file.")
	azureAuthFile  = flags.String("azureAuthFile", "", "Path to Azure Auth File")
	subscriptionID = flags.String("subscriptionID", "", "Azure Subscription")
	verbosity      = flags.Int("verbosity", 1, "Set logging verbosity level")
	namespace      = flags.String("namespace", "default", "Kubernetes namespace to watch.")
	port           = flags.Int("port", 15124, "Endpoint Discovery Service port number. (Default: 15124)")
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

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		glog.Fatalf("Error gathering Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", *kubeConfigFile, err)
	}

	stopChan := make(chan struct{})
	meshSpec := kube.NewMeshSpecClient(kubeConfig, getNamespaces(), 1*time.Second, announceChan, stopChan)
	kubernetesProvider := kube.NewProvider(kubeConfig, getNamespaces(), 1*time.Second, announceChan, "Kubernetes")
	azureProvider := azure.NewProvider(*subscriptionID, *namespace, *azureAuthFile, maxAuthRetryCount, retryPause, announceChan, meshSpec, "Azure")

	// ServiceName Catalog is the facility, which we query to get the list of services, weights for traffic split etc.
	serviceCatalog := catalog.NewServiceCatalog(meshSpec, stopChan, kubernetesProvider, azureProvider)

	grpcServer, lis := utils.NewGrpc(serverType, *port)
	eds := edsServer.NewEDSServer(ctx, serviceCatalog, meshSpec, announceChan)
	envoyControlPlane.RegisterEndpointDiscoveryServiceServer(grpcServer, eds)
	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	close(stopChan)
	glog.Info("Goodbye!")
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
	} else {
		namespaces = []string{*namespace}
	}
	glog.Infof("Observing namespaces: %s", strings.Join(namespaces, ","))
	return namespaces
}
