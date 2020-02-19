package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/deislabs/smc/pkg/constants"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/providers/azure"
	"github.com/deislabs/smc/pkg/providers/kube"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy/ads"
	"github.com/deislabs/smc/pkg/httpserver"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/metricsstore"
	azureResource "github.com/deislabs/smc/pkg/providers/azure/kubernetes"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/tresor"
	"github.com/deislabs/smc/pkg/utils"
)

const (
	serverType       = "ADS"
	defaultNamespace = "default"
)

var azureAuthFile string

var (
	flags          = pflag.NewFlagSet(`ads`, pflag.ExitOnError)
	kubeConfigFile = flags.String("kubeconfig", "", "Path to Kubernetes config file.")
	subscriptionID = flags.String("subscriptionID", "", "Azure Subscription")
	verbosity      = flags.Int("verbosity", int(level.Info), "Set log verbosity level")
	port           = flags.Int("port", 15128, "Clusters Discovery Service port number.")
	namespace      = flags.String("namespace", "default", "Kubernetes namespace to watch for SMI Spec.")
	certPem        = flags.String("certpem", "", fmt.Sprintf("Full path to the %s Certificate PEM file", serverType))
	keyPem         = flags.String("keypem", "", fmt.Sprintf("Full path to the %s Key PEM file", serverType))
	rootCertPem    = flags.String("rootcertpem", "", "Full path to the Root Certificate PEM file")
)

func init() {
	flags.StringVar(&azureAuthFile, "azureAuthFile", "", "Path to Azure Auth File")
}

func main() {
	defer glog.Flush()
	parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		glog.Fatalf("[%s] Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", serverType, *kubeConfigFile, err)
	}

	observeNamespaces := getNamespaces()
	stop := make(chan struct{})

	meshSpec := smi.NewMeshSpecClient(kubeConfig, observeNamespaces, stop)
	certManager, err := tresor.NewCertManagerWithCAFromFile(*rootCertPem, *keyPem, "Acme", 1*time.Hour)
	if err != nil {
		glog.Fatal("Could not instantiate Certificate Manager: ", err)
	}

	endpointsProviders := []endpoint.Provider{
		kube.NewProvider(kubeConfig, observeNamespaces, stop, constants.KubeProviderName),
	}

	if azureAuthFile != "" {
		azureResourceClient := azureResource.NewClient(kubeConfig, observeNamespaces, stop)
		endpointsProviders = append(endpointsProviders, azure.NewProvider(
			*subscriptionID, azureAuthFile, stop, meshSpec, azureResourceClient, constants.AzureProviderName))
	}
	meshCatalog := catalog.NewMeshCatalog(meshSpec, certManager, stop, endpointsProviders...)

	// TODO(draychev): there should be no need to pass meshSpec to the ADS - it is already in meshCatalog
	adsServer := ads.NewADSServer(ctx, meshCatalog, meshSpec)

	grpcServer, lis := utils.NewGrpc(serverType, *port, *certPem, *keyPem, *rootCertPem)
	xds.RegisterAggregatedDiscoveryServiceServer(grpcServer, adsServer)

	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	// initialize the http server and start it
	metricsStore := metricsstore.NewMetricStore("TBD_NameSpace", "TBD_PodName")
	httpServer := httpserver.NewHTTPServer(adsServer, metricsStore, "8888")
	httpServer.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	close(stop)
	glog.Infof("[%s] Goodbye!", serverType)
}

func parseFlags() {
	// TODO(draychev): consolidate parseFlags - shared between ads.go and eds.go
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
	glog.Infof("[%s] Observing namespaces: %s", serverType, strings.Join(namespaces, ","))
	return namespaces
}
