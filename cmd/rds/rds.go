package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	envoyControlPlane "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/endpoint"
	rdsServer "github.com/deislabs/smc/pkg/envoy/rds"
	"github.com/deislabs/smc/pkg/providers/azure"
	azureResource "github.com/deislabs/smc/pkg/providers/azure/kubernetes"
	"github.com/deislabs/smc/pkg/providers/kube"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	serverType = "RDS"

	defaultKubernetesNamespace = "default"

	// These strings identify the participating clusters / endpoint providers.
	// Ideally these should be not only the type of compute but also a unique identifier, like the FQDN of the cluster,
	// or the subscription within the cloud vendor.
	azureProviderName      = "Azure"
	kubernetesProviderName = "Kubernetes"
)

var (
	flags          = pflag.NewFlagSet(`diplomat-rdsServer`, pflag.ExitOnError)
	kubeConfigFile = flags.String("kubeconfig", "", "Path to Kubernetes config file.")
	azureAuthFile  = flags.String("azureAuthFile", "", "Path to Azure Auth File")
	subscriptionID = flags.String("subscriptionID", "", "Azure Subscription")
	verbosity      = flags.Int("verbosity", 1, "Set logging verbosity level")
	namespace      = flags.String("namespace", "default", "Kubernetes namespace to watch.")
	port           = flags.Int("port", 15126, "Route Discovery Services port number. (Default: 15126)")
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
	// This is a signalling mechanism to notify SMC of a service mesh topology change which triggers Envoy updates.
	announcements := make(chan interface{})

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		glog.Fatalf("[RDS] Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", *kubeConfigFile, err)
	}

	observeNamespaces := getNamespaces()

	stopChan := make(chan struct{})
	meshTopologyClient := smi.NewMeshSpecClient(kubeConfig, observeNamespaces, announcements, stopChan)
	azureResourceClient := azureResource.NewClient(kubeConfig, observeNamespaces, announcements, stopChan)

	endpointsProviders := []endpoint.Provider{
		azure.NewProvider(*subscriptionID, *azureAuthFile, announcements, stopChan, meshTopologyClient, azureResourceClient, azureProviderName),
		kube.NewProvider(kubeConfig, observeNamespaces, announcements, stopChan, kubernetesProviderName),
	}

	serviceCatalog := catalog.NewServiceCatalog(meshTopologyClient, endpointsProviders...)

	grpcServer, lis := utils.NewGrpc(serverType, *port, *certPem, *keyPem, *rootCertPem)
	rds := rdsServer.NewRDSServer(ctx, serviceCatalog, meshTopologyClient, announcements)
	envoyControlPlane.RegisterRouteDiscoveryServiceServer(grpcServer, rds)
	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	close(stopChan)
	glog.Info("[RDS] Goodbye!")
}

func parseFlags() {
	// TODO: consolidate parseFlags - shared between sds.go, eds.go and rds.go
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
		defaultNS := defaultKubernetesNamespace
		namespaces = []string{defaultNS}
	} else {
		namespaces = []string{*namespace}
	}
	glog.Infof("[%s] Observing namespaces: %s", serverType, strings.Join(namespaces, ","))
	return namespaces
}
