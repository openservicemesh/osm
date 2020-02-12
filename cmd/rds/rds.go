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

	envoyControlPlane "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/constants"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy/rds"
	"github.com/deislabs/smc/pkg/log"
	"github.com/deislabs/smc/pkg/providers/azure"
	azureResource "github.com/deislabs/smc/pkg/providers/azure/kubernetes"
	"github.com/deislabs/smc/pkg/providers/kube"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/tresor"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	serverType = "RDS"
)

var (
	flags          = pflag.NewFlagSet(`rds`, pflag.ExitOnError)
	kubeConfigFile = flags.String("kubeconfig", "", "Path to Kubernetes config file.")
	azureAuthFile  = flags.String("azureAuthFile", "", "Path to Azure Auth File")
	subscriptionID = flags.String("subscriptionID", "", "Azure Subscription")
	verbosity      = flags.Int("verbosity", int(log.LvlInfo), "Set log verbosity level")
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

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		glog.Fatalf("[RDS] Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", *kubeConfigFile, err)
	}

	observeNamespaces := getNamespaces()

	stop := make(chan struct{})
	meshSpecClient := smi.NewMeshSpecClient(kubeConfig, observeNamespaces, stop)
	certManager, err := tresor.NewCertManagerWithCAFromFile(*rootCertPem, *keyPem, "Acme", 1*time.Hour)
	if err != nil {
		glog.Fatal("Could not instantiate Certificate Manager: ", err)
	}
	azureResourceClient := azureResource.NewClient(kubeConfig, observeNamespaces, stop)
	endpointsProviders := []endpoint.Provider{
		azure.NewProvider(*subscriptionID, *azureAuthFile, stop, meshSpecClient, azureResourceClient, constants.AzureProviderName),
		kube.NewProvider(kubeConfig, observeNamespaces, stop, constants.KubeProviderName),
	}

	meshCatalog := catalog.NewMeshCatalog(meshSpecClient, certManager, stop, endpointsProviders...)
	rdsServer := rds.NewRDSServer(ctx, meshCatalog, meshSpecClient)

	grpcServer, lis := utils.NewGrpc(serverType, *port, *certPem, *keyPem, *rootCertPem)
	envoyControlPlane.RegisterRouteDiscoveryServiceServer(grpcServer, rdsServer)
	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	close(stop)
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
		defaultNS := constants.DefaultKubeNamespace
		namespaces = []string{defaultNS}
	} else {
		namespaces = []string{*namespace}
	}
	glog.Infof("[%s] Observing namespaces: %s", serverType, strings.Join(namespaces, ","))
	return namespaces
}
