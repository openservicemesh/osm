package main

import (
	"context"
	"flag"
	"os"
	"time"

	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy/ads"
	"github.com/open-service-mesh/osm/pkg/featureflags"
	"github.com/open-service-mesh/osm/pkg/httpserver"
	"github.com/open-service-mesh/osm/pkg/ingress"
	"github.com/open-service-mesh/osm/pkg/injector"
	"github.com/open-service-mesh/osm/pkg/logger"
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
	// TODO(draychev): pass this via CLI param (https://github.com/open-service-mesh/osm/issues/542)
	serverType = "ADS"

	defaultCertValidityMinutes = 525600 // 1 year

	tlsCAKey = "ca.crt"
)

var (
	verbosity      string
	osmID          string // An ID that uniquely identifies an OSM instance
	azureAuthFile  string
	kubeConfigFile string
	osmNamespace   string
	injectorConfig injector.Config

	// feature flag options
	optionalFeatures featureflags.OptionalFeatures
)

var (
	flags               = pflag.NewFlagSet(`ads`, pflag.ExitOnError)
	caBundleSecretName  = flags.String("caBundleSecretName", "", "Name of the Kubernetes Secret for the OSM CA bundle")
	azureSubscriptionID = flags.String("azureSubscriptionID", "", "Azure Subscription ID")
	port                = flags.Int("port", constants.AggregatedDiscoveryServicePort, "Aggregated Discovery Service port number.")
	certPem             = flags.String("certpem", "", "Full path to the xDS Certificate PEM file")
	keyPem              = flags.String("keypem", "", "Full path to the xDS Key PEM file")
	rootCertPem         = flags.String("rootcertpem", "", "Full path to the Root Certificate PEM file")
	rootKeyPem          = flags.String("rootkeypem", "", "Full path to the Root Key PEM file")
	log                 = logger.New("ads/main")
	validity            = flags.Int("validity", defaultCertValidityMinutes, "validity duration of a certificate in MINUTES")
)

func init() {
	flags.StringVar(&verbosity, "verbosity", "info", "Set log verbosity level")
	flags.StringVar(&osmID, "osmID", "", "OSM instance ID")
	flags.StringVar(&azureAuthFile, "azureAuthFile", "", "Path to Azure Auth File")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&osmNamespace, "osmNamespace", "", "Namespace to which OSM belongs to.")

	// sidecar injector options
	flags.BoolVar(&injectorConfig.EnableTLS, "enable-tls", true, "Enable TLS")
	flags.BoolVar(&injectorConfig.DefaultInjection, "default-injection", true, "Enable sidecar injection by default")
	flags.IntVar(&injectorConfig.ListenPort, "webhook-port", constants.InjectorWebhookPort, "Webhook port for sidecar-injector")
	flags.StringVar(&injectorConfig.InitContainerImage, "init-container-image", "", "InitContainer image")
	flags.StringVar(&injectorConfig.SidecarImage, "sidecar-image", "", "Sidecar proxy Container image")

	// feature flags
	flags.BoolVar(&optionalFeatures.Ingress, "enable-ingress", false, "Enable ingress in OSM")
}

func main() {
	log.Trace().Msg("Starting ADS")
	parseFlags()
	logger.SetLogLevel(verbosity)
	featureflags.Initialize(optionalFeatures)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var kubeConfig *rest.Config
	var err error
	if kubeConfigFile != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			log.Fatal().Err(err).Msgf("[%s] Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s'", serverType, kubeConfigFile)
		}
	} else {
		// creates the in-cluster config
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Fatal().Err(err).Msg("Error generating Kubernetes config")
		}
	}

	if osmID == "" {
		log.Fatal().Msg("Please specify the OSM instance ID using --osmID")
	}
	if osmNamespace == "" {
		log.Fatal().Msg("Please specify the OSM namespace using --osmNamespace")
	}
	if injectorConfig.InitContainerImage == "" {
		log.Fatal().Msg("Please specify the init container image using --init-container-image ")
	}
	if injectorConfig.SidecarImage == "" {
		log.Fatal().Msg("Please specify the sidecar image using --sidecar-image ")
	}

	stop := signals.RegisterExitHandlers()

	namespaceController := namespace.NewNamespaceController(kubeConfig, osmID, stop)
	meshSpec := smi.NewMeshSpecClient(kubeConfig, osmNamespace, namespaceController, stop)

	certValidityPeriod := time.Duration(*validity) * time.Minute

	var certManager certificate.Manager

	{
		// TODO(draychev): save/load root cert from persistent storage:  https://github.com/open-service-mesh/osm/issues/541
		certManagerKind := "tresor"
		rootCert, err := tresor.NewCA(constants.CertificationAuthorityCommonName, certValidityPeriod)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to create new Certificate Authority with cert issuer %s", certManagerKind)
		}

		if rootCert == nil {
			log.Fatal().Msgf("Invalid root certificate created by cert issuer %s", certManagerKind)
		}

		certManager, err = tresor.NewCertManager(rootCert, certValidityPeriod)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to instantiate certificate manager")
		}
	}

	if err := createCABundleKubernetesSecret(kubeConfig, certManager, caBundleSecretName); err != nil {
		log.Error().Err(err).Msgf("Error exporting CA bundle into Kubernetes secret with name %s", *caBundleSecretName)
	}

	endpointsProviders := []endpoint.Provider{
		kube.NewProvider(kubeConfig, namespaceController, stop, constants.KubeProviderName),
	}

	if azureAuthFile != "" {
		azureResourceClient := azureResource.NewClient(kubeConfig, namespaceController, stop)
		endpointsProviders = append(endpointsProviders, azure.NewProvider(
			*azureSubscriptionID, azureAuthFile, stop, meshSpec, azureResourceClient, constants.AzureProviderName))
	}

	ingressClient, err := ingress.NewIngressClient(kubeConfig, namespaceController, stop)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize ingress client")
	}
	meshCatalog := catalog.NewMeshCatalog(meshSpec, certManager, ingressClient, stop, endpointsProviders...)

	// Create the sidecar-injector webhook
	webhook := injector.NewWebhook(injectorConfig, kubeConfig, certManager, meshCatalog, namespaceController, osmNamespace)
	go webhook.ListenAndServe(stop)

	// TODO(draychev): there should be no need to pass meshSpec to the ADS - it is already in meshCatalog
	adsServer := ads.NewADSServer(ctx, meshCatalog, meshSpec)

	// TODO(draychev): we need to pass this hard-coded string is a CLI argument (https://github.com/open-service-mesh/osm/issues/542)
	adsCert, err := certManager.IssueCertificate("ads")
	if err != nil {
		log.Fatal().Err(err)
	}

	grpcServer, lis := utils.NewGrpc(serverType, *port, adsCert.GetCertificateChain(), adsCert.GetPrivateKey(), adsCert.GetIssuingCA())
	xds.RegisterAggregatedDiscoveryServiceServer(grpcServer, adsServer)

	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	// initialize the http server and start it
	// TODO(draychev): figure out the NS and POD
	metricsStore := metricsstore.NewMetricStore("TBD_NameSpace", "TBD_PodName")
	// TODO(draychev): the port number should be configurable
	httpServer := httpserver.NewHTTPServer(adsServer, metricsStore, constants.MetricsServerPort, meshCatalog.GetDebugInfo)
	httpServer.Start()

	// Wait for exit handler signal
	<-stop

	log.Info().Msgf("[%s] Goodbye!", serverType)
}

func parseFlags() {
	// TODO(draychev): consolidate parseFlags - shared between ads.go and eds.go
	if err := flags.Parse(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}
	_ = flag.CommandLine.Parse([]string{})
}

func createCABundleKubernetesSecret(kubeConfig *rest.Config, certManager certificate.Manager, caBundleSecretName *string) error {
	if caBundleSecretName == nil || *caBundleSecretName == "" {
		return nil
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	cn := "localhost" // the CN does not matter much - cert won't be used -- 'localhost' is used for throwaway certs.
	cert, err := certManager.IssueCertificate("localhost")
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing %s certificate", cn)
		return nil
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: *caBundleSecretName,
		},
		Data: map[string][]byte{
			tlsCAKey: cert.GetIssuingCA(),
		},
	}

	log.Info().Msgf("Creating bootstrap config for Envoy: name=%s, namespace=%s", *caBundleSecretName, osmNamespace)
	_, err = kubeClient.CoreV1().Secrets(osmNamespace).Create(context.Background(), secret, metav1.CreateOptions{})
	return err
}
