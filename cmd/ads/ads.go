package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/open-service-mesh/osm/demo/cmd/common"
	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy/ads"
	"github.com/open-service-mesh/osm/pkg/envoy/metrics"
	"github.com/open-service-mesh/osm/pkg/httpserver"
	"github.com/open-service-mesh/osm/pkg/injector"
	"github.com/open-service-mesh/osm/pkg/log/level"
	"github.com/open-service-mesh/osm/pkg/metricsstore"
	"github.com/open-service-mesh/osm/pkg/namespace"
	"github.com/open-service-mesh/osm/pkg/providers/azure"
	azureResource "github.com/open-service-mesh/osm/pkg/providers/azure/kubernetes"
	"github.com/open-service-mesh/osm/pkg/providers/kube"
	"github.com/open-service-mesh/osm/pkg/signals"
	"github.com/open-service-mesh/osm/pkg/smi"
	"github.com/open-service-mesh/osm/pkg/tresor"
	"github.com/open-service-mesh/osm/pkg/utils"
)

const (
	serverType = "ADS"
)

var (
	osmID          string // An ID that uniquely identifies an OSM instance
	azureAuthFile  string
	kubeConfigFile string
	appNamespaces  string // comma separated list of namespaces to observe
	osmNamespace   string
	injectorConfig injector.Config
)

var (
	flags          = pflag.NewFlagSet(`ads`, pflag.ExitOnError)
	subscriptionID = flags.String("subscriptionID", "", "Azure Subscription")
	verbosity      = flags.Int("verbosity", int(level.Info), "Set log verbosity level")
	port           = flags.Int("port", 15128, "Clusters Discovery Service port number.")
	certPem        = flags.String("certpem", "", "Full path to the xDS Certificate PEM file")
	keyPem         = flags.String("keypem", "", "Full path to the xDS Key PEM file")
	rootCertPem    = flags.String("rootcertpem", "", "Full path to the Root Certificate PEM file")
	rootKeyPem     = flags.String("rootkeypem", "", "Full path to the Root Key PEM file")
)

func init() {
	flags.StringVar(&osmID, "osmID", "", "OSM instance ID")
	flags.StringVar(&azureAuthFile, "azureAuthFile", "", "Path to Azure Auth File")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&appNamespaces, "appNamespaces", "", "List of comma separated application namespaces OSM should manage.")
	flags.StringVar(&osmNamespace, "osmNamespace", "", "Namespace to which OSM belongs to.")

	// sidecar injector options
	flags.BoolVar(&injectorConfig.EnableTLS, "enable-tls", true, "Enable TLS")
	flags.BoolVar(&injectorConfig.DefaultInjection, "default-injection", true, "Enable sidecar injection by default")
	flags.IntVar(&injectorConfig.ListenPort, "webhook-port", constants.InjectorWebhookPort, "Webhook port for sidecar-injector")
	flags.StringVar(&injectorConfig.InitContainerImage, "init-container-image", "", "InitContainer image")
	flags.StringVar(&injectorConfig.SidecarImage, "sidecar-image", "", "Sidecar proxy Container image")
}

func main() {
	defer glog.Flush()
	parseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var kubeConfig *rest.Config
	var err error
	if kubeConfigFile != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			glog.Fatalf("[%s] Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", serverType, kubeConfigFile, err)
		}
	} else {
		// creates the in-cluster config
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("[RDS] Error generating Kubernetes config: %s", err)
		}
	}

	if osmID == "" {
		glog.Fatal("Please specify the OSM instance ID using --osmID")
	}
	if osmNamespace == "" {
		glog.Fatal("Please specify the OSM namespace using --osmNamespace")
	}
	if injectorConfig.InitContainerImage == "" {
		glog.Fatal("Please specify the init container image using --init-container-image ")
	}
	if injectorConfig.SidecarImage == "" {
		glog.Fatal("Please specify the sidecar image using --sidecar-image ")
	}

	stop := signals.RegisterExitHandlers()

	namespaceController := namespace.NewNamespaceController(kubeConfig, osmID, stop)
	meshSpec := smi.NewMeshSpecClient(kubeConfig, osmNamespace, namespaceController, stop)
	certManager, err := tresor.NewCertManagerWithCAFromFile(*rootCertPem, *rootKeyPem, "Acme", 1*time.Hour)
	if err != nil {
		glog.Fatal("Could not instantiate Certificate Manager: ", err)
	}

	endpointsProviders := []endpoint.Provider{
		kube.NewProvider(kubeConfig, namespaceController, stop, constants.KubeProviderName),
	}

	if azureAuthFile != "" {
		azureResourceClient := azureResource.NewClient(kubeConfig, namespaceController, stop)
		endpointsProviders = append(endpointsProviders, azure.NewProvider(
			*subscriptionID, azureAuthFile, stop, meshSpec, azureResourceClient, constants.AzureProviderName))
	}
	meshCatalog := catalog.NewMeshCatalog(meshSpec, certManager, stop, endpointsProviders...)

	// Create the sidecar-injector webhook
	webhook := injector.NewWebhook(injectorConfig, kubeConfig, certManager, meshCatalog, namespaceController, osmNamespace)
	go webhook.ListenAndServe(stop)

	// TODO(draychev): there should be no need to pass meshSpec to the ADS - it is already in meshCatalog
	adsServer := ads.NewADSServer(ctx, meshCatalog, meshSpec)

	grpcServer, lis := utils.NewGrpc(serverType, *port, *certPem, *keyPem, *rootCertPem)
	xds.RegisterAggregatedDiscoveryServiceServer(grpcServer, adsServer)

	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	// initialize the http server and start it
	// TODO(draychev): figure out the NS and POD
	metricsStore := metricsstore.NewMetricStore("TBD_NameSpace", "TBD_PodName")

	metrics.Start(common.MetricsServicePort, certPem, keyPem, rootCertPem, stop)

	// TODO(draychev): the port number should be configurable
	httpServer := httpserver.NewHTTPServer(adsServer, metricsStore, "15000", meshCatalog.GetDebugInfo)
	httpServer.Start()

	// Wait for exit handler signal
	<-stop

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
