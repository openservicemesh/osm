package main

import (
	"context"
	"flag"
	"os"

	xds "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
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
	"github.com/open-service-mesh/osm/pkg/utils"
)

const (
	// TODO(draychev): pass this via CLI param (https://github.com/open-service-mesh/osm/issues/542)
	serverType = "ADS"

	defaultCertValidityMinutes = 525600 // 1 year
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
	log                 = logger.New("ads/main")
	validity            = flags.Int("validity", defaultCertValidityMinutes, "validity duration of a certificate in MINUTES")

	// What is the Certification Authority to be used
	certManagerKind = flags.String("certmanager", "tresor", "Certificate manager")

	// TODO(draychev): convert all these flags to spf13/cobra: https://github.com/open-service-mesh/osm/issues/576
	// When certmanager == "vault"
	vaultProtocol = flags.String("vaultProtocol", "http", "Host name of the Hashi Vault")
	vaultHost     = flags.String("vaultHost", "vault.default.svc.cluster.local", "Host name of the Hashi Vault")
	vaultPort     = flags.Int("vaultPort", 8200, "Port of the Hashi Vault")
	vaultToken    = flags.String("vaultToken", "", "Secret token for the the Hashi Vault")
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

	// This ensures CLI parameters (and dependent values) are correct.
	// Side effects: This will log.Fatal on error resulting in os.Exit(255)
	validateCLIParams()

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

	stop := signals.RegisterExitHandlers()

	namespaceController := namespace.NewNamespaceController(kubeConfig, osmID, stop)
	meshSpec := smi.NewMeshSpecClient(kubeConfig, osmNamespace, namespaceController, stop)

	certManager := certManagers[certificateManagerKind(*certManagerKind)]()

	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	if err := createCABundleKubernetesSecret(kubeClient, certManager, osmNamespace, *caBundleSecretName); err != nil {
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
	if err := injector.NewWebhook(injectorConfig, kubeConfig, certManager, meshCatalog, namespaceController, osmNamespace, stop); err != nil {
		log.Fatal().Err(err).Msg("Error creating mutating webhook")
	}

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

func createCABundleKubernetesSecret(kubeClient clientset.Interface, certManager certificate.Manager, namespace, caBundleSecretName string) error {
	if caBundleSecretName == "" {
		log.Info().Msg("No name provided for CA bundle k8s secret. Skip creation of secret")
		return nil
	}

	// the CN does not matter much - cert won't be used -- 'localhost' is used for throwaway certs.
	cn := certificate.CommonName("localhost")
	cert, err := certManager.IssueCertificate(cn)
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing %s certificate", cn)
		return nil
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caBundleSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			constants.KubernetesOpaqueSecretCAKey: cert.GetIssuingCA(),
		},
	}

	_, err = kubeClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error creating CA bundle Kubernetes secret %s in namespace %s", caBundleSecretName, namespace)
		return err
	}

	log.Info().Msgf("Created CA bundle Kubernetes secret %s in namespace %s", caBundleSecretName, namespace)
	return nil
}
