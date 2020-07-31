package main

import (
	"context"
	"flag"
	"os"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/debugger"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/endpoint/providers/azure"
	azureResource "github.com/openservicemesh/osm/pkg/endpoint/providers/azure/kubernetes"
	"github.com/openservicemesh/osm/pkg/endpoint/providers/kube"
	"github.com/openservicemesh/osm/pkg/envoy/ads"
	"github.com/openservicemesh/osm/pkg/featureflags"
	"github.com/openservicemesh/osm/pkg/httpserver"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/injector"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/utils"
)

const (
	serverType                        = "ADS"
	defaultServiceCertValidityMinutes = 60 // 1 hour
	caBundleSecretNameCLIParam        = "ca-bundle-secret-name"
	xdsServerCertificateCommonName    = "ads"
)

var (
	verbosity                  string
	meshName                   string // An ID that uniquely identifies an OSM instance
	azureAuthFile              string
	kubeConfigFile             string
	osmNamespace               string
	webhookName                string
	serviceCertValidityMinutes int
	caBundleSecretName         string
	enableDebugServer          bool
	osmConfigMapName           string

	injectorConfig injector.Config

	// feature flag options
	optionalFeatures featureflags.OptionalFeatures
)

var (
	flags               = pflag.NewFlagSet(`ads`, pflag.ExitOnError)
	azureSubscriptionID = flags.String("azure-subscription-id", "", "Azure Subscription ID")
	port                = flags.Int("port", constants.OSMControllerPort, "Aggregated Discovery MeshService port number.")
	log                 = logger.New("ads/main")

	// What is the Certification Authority to be used
	certManagerKind = flags.String("cert-manager", "tresor", "Certificate manager")

	// TODO(draychev): convert all these flags to spf13/cobra: https://github.com/openservicemesh/osm/issues/576
	// When certmanager == "vault"
	vaultProtocol = flags.String("vault-protocol", "http", "Host name of the Hashi Vault")
	vaultHost     = flags.String("vault-host", "vault.default.svc.cluster.local", "Host name of the Hashi Vault")
	vaultPort     = flags.Int("vault-port", 8200, "Port of the Hashi Vault")
	vaultToken    = flags.String("vault-token", "", "Secret token for the the Hashi Vault")
	vaultRole     = flags.String("vault-role", "openservicemesh", "Name of the Vault role dedicated to Open MeshService Mesh")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "OSM mesh name")
	flags.StringVar(&azureAuthFile, "azure-auth-file", "", "Path to Azure Auth File")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&osmNamespace, "osm-namespace", "", "Namespace to which OSM belongs to.")
	flags.StringVar(&webhookName, "webhook-name", "", "Name of the MutatingWebhookConfiguration to be configured by ADS")
	flags.IntVar(&serviceCertValidityMinutes, "service-cert-validity-minutes", defaultServiceCertValidityMinutes, "Certificate validityPeriod duration in minutes")
	flags.StringVar(&caBundleSecretName, caBundleSecretNameCLIParam, "", "Name of the Kubernetes Secret for the OSM CA bundle")
	flags.BoolVar(&enableDebugServer, "enable-debug-server", false, "Enable OSM debug HTTP server")
	flags.StringVar(&osmConfigMapName, "osm-configmap-name", "osm-config", "Name of the OSM ConfigMap")

	// sidecar injector options
	flags.BoolVar(&injectorConfig.DefaultInjection, "default-injection", true, "Enable sidecar injection by default")
	flags.IntVar(&injectorConfig.ListenPort, "webhook-port", constants.InjectorWebhookPort, "Webhook port for sidecar-injector")
	flags.StringVar(&injectorConfig.InitContainerImage, "init-container-image", "", "InitContainer image")
	flags.StringVar(&injectorConfig.SidecarImage, "sidecar-image", "", "Sidecar proxy Container image")

	// feature flags
	flags.BoolVar(&optionalFeatures.Backpressure, "enable-backpressure-experimental", false, "Enable experimental backpressure feature")
}

func main() {
	log.Trace().Msg("Starting ADS")
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}
	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}
	featureflags.Initialize(optionalFeatures)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// This ensures CLI parameters (and dependent values) are correct.
	// Side effects: This will log.Fatal on error resulting in os.Exit(255)
	if err := validateCLIParams(); err != nil {
		log.Fatal().Err(err).Msg("Failed to validate CLI parameters")
	}

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)

	smiKubeConfig := &kubeConfig
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", kubeConfigFile)
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	stop := signals.RegisterExitHandlers()

	// This component will be watching the OSM ConfigMap and will make it
	// to the rest of the components.
	cfg := configurator.NewConfigurator(kubernetes.NewForConfigOrDie(kubeConfig), stop, osmNamespace, osmConfigMapName)
	configMap, err := cfg.GetConfigMap()
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing ConfigMap %s", osmConfigMapName)
	}
	log.Info().Msgf("Initial ConfigMap %s: %s", osmConfigMapName, string(configMap))

	namespaceController := namespace.NewNamespaceController(kubeClient, meshName, stop)
	meshSpec, err := smi.NewMeshSpecClient(*smiKubeConfig, kubeClient, osmNamespace, namespaceController, stop)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create new mesh spec client")
	}

	// Get the Certificate Manager based on the CLI argument passed to this module.
	certManager, certDebugger, err := certManagers[certificateManagerKind(*certManagerKind)](kubeClient, enableDebugServer)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to get certificate manager based on CLI argument")
	}

	log.Info().Msgf("MeshService certificates will be valid for %+v", getServiceCertValidityPeriod())

	if caBundleSecretName == "" {
		log.Info().Msgf("CA bundle will not be exported to a k8s secret (no --%s provided)", caBundleSecretNameCLIParam)
	} else {
		if err := createCABundleKubernetesSecret(kubeClient, certManager, osmNamespace, caBundleSecretName); err != nil {
			log.Error().Err(err).Msgf("Error exporting CA bundle into Kubernetes secret with name %s", caBundleSecretName)
		}
	}

	provider, err := kube.NewProvider(kubeClient, namespaceController, stop, constants.KubeProviderName, cfg)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to get endpoint provider")
	}

	endpointsProviders := []endpoint.Provider{provider}

	if azureAuthFile != "" {
		azureResourceClient, err := azureResource.NewClient(kubeClient, kubeConfig, namespaceController, stop, cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize azure resource client")
		}
		azureProvider, err := azure.NewProvider(*azureSubscriptionID, azureAuthFile, stop, meshSpec, azureResourceClient, constants.AzureProviderName)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize azure provider")
		}
		endpointsProviders = append(endpointsProviders, azureProvider)
	}

	ingressClient, err := ingress.NewIngressClient(kubeClient, namespaceController, stop, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize ingress client")
	}

	meshCatalog := catalog.NewMeshCatalog(
		kubeClient,
		meshSpec,
		certManager,
		ingressClient,
		stop,
		cfg,
		endpointsProviders...)

	// Create the sidecar-injector webhook
	if err := injector.NewWebhook(injectorConfig, kubeClient, certManager, meshCatalog, namespaceController, meshName, osmNamespace, webhookName, stop, cfg); err != nil {
		log.Fatal().Err(err).Msg("Error creating mutating webhook")
	}

	// TODO(draychev): there should be no need to pass meshSpec to the ADS - it is already in meshCatalog
	xdsServer := ads.NewADSServer(ctx, meshCatalog, meshSpec, enableDebugServer, osmNamespace, cfg)

	// TODO(draychev): we need to pass this hard-coded string is a CLI argument (https://github.com/openservicemesh/osm/issues/542)
	validityPeriod := constants.XDSCertificateValidityPeriod
	adsCert, err := certManager.IssueCertificate(xdsServerCertificateCommonName, &validityPeriod)
	if err != nil {
		log.Fatal().Err(err)
	}

	grpcServer, lis := utils.NewGrpc(serverType, *port, adsCert.GetCertificateChain(), adsCert.GetPrivateKey(), adsCert.GetIssuingCA())
	xds_discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, xdsServer)

	go utils.GrpcServe(ctx, grpcServer, lis, cancel, serverType)

	// initialize the http server and start it
	// TODO(draychev): figure out the NS and POD
	metricsStore := metricsstore.NewMetricStore("TBD_NameSpace", "TBD_PodName")

	// Expose /debug endpoints and data only if the enableDebugServer flag is enabled
	var debugServer debugger.DebugServer
	if enableDebugServer {
		debugServer = debugger.NewDebugServer(certDebugger, xdsServer, meshCatalog, kubeConfig, kubeClient, cfg)
	}
	httpServer := httpserver.NewHTTPServer(xdsServer, metricsStore, constants.MetricsServerPort, debugServer)
	httpServer.Start()

	// Wait for exit handler signal
	<-stop

	log.Info().Msgf("[%s] Goodbye!", serverType)
}

func parseFlags() error {
	// TODO(draychev): consolidate parseFlags - shared between ads.go and eds.go
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

func createCABundleKubernetesSecret(kubeClient clientset.Interface, certManager certificate.Manager, namespace, caBundleSecretName string) error {
	if caBundleSecretName == "" {
		log.Info().Msg("No name provided for CA bundle k8s secret. Skip creation of secret")
		return nil
	}

	ca, err := certManager.GetRootCertificate()
	if err != nil {
		log.Error().Err(err).Msgf("Error getting root certificate")
		return nil
	}

	return saveSecretToKubernetes(kubeClient, ca, namespace, caBundleSecretName, nil)
}

func saveSecretToKubernetes(kubeClient clientset.Interface, ca certificate.Certificater, namespace, caBundleSecretName string, privKey []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caBundleSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			constants.KubernetesOpaqueSecretCAKey:        ca.GetCertificateChain(),
			constants.KubernetesOpaqueSecretCAExpiration: encodeExpiration(ca.GetExpiration()),
		},
	}

	if privKey != nil {
		secret.Data[constants.KubernetesOpaqueSecretRootPrivateKeyKey] = privKey
	}

	if _, err := kubeClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{}); err != nil {
		log.Error().Err(err).Msgf("Error creating CA bundle Kubernetes secret %s in namespace %s", caBundleSecretName, namespace)
		return err
	}

	log.Info().Msgf("Created CA bundle Kubernetes secret %s in namespace %s", caBundleSecretName, namespace)
	return nil
}
