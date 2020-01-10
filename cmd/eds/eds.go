package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/eapache/channels"
	envoyControlPlane "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/endpoint"
	edsServer "github.com/deislabs/smc/pkg/envoy/eds"
	"github.com/deislabs/smc/pkg/providers/azure"
	azureResource "github.com/deislabs/smc/pkg/providers/azure/kubernetes"
	"github.com/deislabs/smc/pkg/providers/kube"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	serverType = "EDS"

	defaultNamespace = "default"

	// These strings identify the participating clusters / endpoint providers.
	// Ideally these should be not only the type of compute but also a unique identifier, like the FQDN of the cluster,
	// or the subscription within the cloud vendor.
	azureProviderName      = "Azure"
	kubernetesProviderName = "Kubernetes"
)

var (
	flags          = pflag.NewFlagSet(`diplomat-edsServer`, pflag.ExitOnError)
	kubeConfigFile = flags.String("kubeconfig", "", "Path to Kubernetes config file.")
	azureAuthFile  = flags.String("azureAuthFile", "", "Path to Azure Auth File")
	subscriptionID = flags.String("subscriptionID", "", "Azure Subscription")
	verbosity      = flags.Int("verbosity", 1, "Set logging verbosity level")
	namespace      = flags.String("namespace", "default", "Kubernetes namespace to watch.")
	port           = flags.Int("port", 15124, "Endpoint Discovery Services port number. (Default: 15124)")
)

func main() {
	defer glog.Flush()
	parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// SMI Informers will write to this channel when they notice changes.
	// This channel will be consumed by the ServiceName Mesh Controller.
	// This is a signalling mechanism to notify SMC of a service mesh topology change which triggers Envoy updates.
	announceChan := channels.NewRingChannel(1024)

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		glog.Fatalf("[EDS] Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", *kubeConfigFile, err)
	}

	observeNamespaces := getNamespaces()

	stopChan := make(chan struct{})
	meshTopologyClient := smi.NewMeshTopologyClient(kubeConfig, observeNamespaces, announceChan, stopChan)
	azureResourceClient := azureResource.NewClient(kubeConfig, observeNamespaces, announceChan, stopChan)

	endpointsProviders := []endpoint.Provider{
		azure.NewProvider(*subscriptionID, *azureAuthFile, announceChan, stopChan, meshTopologyClient, azureResourceClient, azureProviderName),
		kube.NewProvider(kubeConfig, observeNamespaces, announceChan, stopChan, kubernetesProviderName),
	}

	serviceCatalog := catalog.NewServiceCatalog(meshTopologyClient, endpointsProviders...)

	grpcServer, lis := utils.NewGrpc(serverType, *port)
	eds := edsServer.NewEDSServer(ctx, serviceCatalog, meshTopologyClient, announceChan)
	envoyControlPlane.RegisterEndpointDiscoveryServiceServer(grpcServer, eds)
	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	close(stopChan)
	glog.Info("[EDS] Goodbye!")
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
		defaultNS := defaultNamespace
		namespaces = []string{defaultNS}
	} else {
		namespaces = []string{*namespace}
	}
	glog.Infof("[EDS] Observing namespaces: %s", strings.Join(namespaces, ","))
	return namespaces
}
