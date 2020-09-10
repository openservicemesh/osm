package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/debugger"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/endpoint/providers/kube"
	"github.com/openservicemesh/osm/pkg/envoy/ads"
	"github.com/openservicemesh/osm/pkg/featureflags"
	"github.com/openservicemesh/osm/pkg/health"
	"github.com/openservicemesh/osm/pkg/httpserver"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/injector"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/version"
)

const (
	defaultServiceCertValidityMinutes = 60 // 1 hour
	caBundleSecretNameCLIParam        = "ca-bundle-secret-name"
	xdsServerCertificateCommonName    = "ads"
)

var (
	verbosity                  string
	meshName                   string // An ID that uniquely identifies an OSM instance
	kubeConfigFile             string
	osmNamespace               string
	webhookConfigName          string
	serviceCertValidityMinutes int
	caBundleSecretName         string
	enableDebugServer          bool
	osmConfigMapName           string

	injectorConfig injector.Config

	// feature flag options
	optionalFeatures featureflags.OptionalFeatures
)

var (
	flags = pflag.NewFlagSet(`osm-controller`, pflag.ExitOnError)
	port  = flags.Int("port", constants.OSMControllerPort, "Aggregated Discovery Service port number.")
	log   = logger.New("osm-controller/main")

	// What is the Certification Authority to be used
	osmCertificateManagerKind = flags.String("certificate-manager", "tresor", fmt.Sprintf("Certificate manager [%v]", strings.Join(validCertificateManagerOptions, "|")))

	// When certmanager == "vault"
	vaultProtocol = flags.String("vault-protocol", "http", "Host name of the Hashi Vault")
	vaultHost     = flags.String("vault-host", "vault.default.svc.cluster.local", "Host name of the Hashi Vault")
	vaultPort     = flags.Int("vault-port", 8200, "Port of the Hashi Vault")
	vaultToken    = flags.String("vault-token", "", "Secret token for the the Hashi Vault")
	vaultRole     = flags.String("vault-role", "openservicemesh", "Name of the Vault role dedicated to Open Service Mesh")

	certmanagerIssuerName  = flags.String("cert-manager-issuer-name", "osm-ca", "cert-manager issuer name")
	certmanagerIssuerKind  = flags.String("cert-manager-issuer-kind", "Issuer", "cert-manager issuer kind")
	certmanagerIssuerGroup = flags.String("cert-manager-issuer-group", "cert-manager.io", "cert-manager issuer group")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "OSM mesh name")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&osmNamespace, "osm-namespace", "", "Namespace to which OSM belongs to.")
	flags.StringVar(&webhookConfigName, "webhook-config-name", "", "Name of the MutatingWebhookConfiguration to be configured by osm-controller")
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
	log.Info().Msgf("Starting osm-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
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

	kubernetesClient := k8s.NewKubernetesClient(kubeClient, meshName, stop)
	meshSpec, err := smi.NewMeshSpecClient(*smiKubeConfig, kubeClient, osmNamespace, kubernetesClient, stop)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create new mesh spec client")
	}

	certManager, certDebugger, err := getCertificateManager(kubeClient, kubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to get certificate manager based on CLI argument: %s", *osmCertificateManagerKind)
	}

	log.Info().Msgf("Service certificates will be valid for %+v", getServiceCertValidityPeriod())

	if caBundleSecretName == "" {
		log.Info().Msgf("CA bundle will not be exported to a k8s secret (no --%s provided)", caBundleSecretNameCLIParam)
	} else {
		if err := createOrUpdateCABundleKubernetesSecret(kubeClient, certManager, osmNamespace, caBundleSecretName); err != nil {
			log.Error().Err(err).Msgf("Error exporting CA bundle into Kubernetes secret with name %s", caBundleSecretName)
		}
	}

	provider, err := kube.NewProvider(kubeClient, kubernetesClient, stop, constants.KubeProviderName, cfg)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to get endpoint provider")
	}

	endpointsProviders := []endpoint.Provider{provider}

	ingressClient, err := ingress.NewIngressClient(kubeClient, kubernetesClient, stop, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize ingress client")
	}

	meshCatalog := catalog.NewMeshCatalog(
		kubernetesClient,
		kubeClient,
		meshSpec,
		certManager,
		ingressClient,
		stop,
		cfg,
		endpointsProviders...)

	// Create the sidecar-injector webhook
	if err := injector.NewWebhook(injectorConfig, kubeClient, certManager, meshCatalog, kubernetesClient, meshName, osmNamespace, webhookConfigName, stop, cfg); err != nil {
		log.Fatal().Err(err).Msg("Error creating mutating webhook")
	}

	// TODO(draychev): we need to pass this hard-coded string is a CLI argument (https://github.com/openservicemesh/osm/issues/542)
	validityPeriod := constants.XDSCertificateValidityPeriod
	adsCert, err := certManager.IssueCertificate(xdsServerCertificateCommonName, &validityPeriod)
	if err != nil {
		log.Fatal().Err(err)
	}

	// Create and start the ADS gRPC service
	xdsServer := ads.NewADSServer(meshCatalog, enableDebugServer, osmNamespace, cfg)
	xdsServer.Start(ctx, cancel, *port, adsCert)

	// initialize the http server and start it
	// TODO(draychev): figure out the NS and POD
	metricsStore := metricsstore.NewMetricStore("TBD_NameSpace", "TBD_PodName")

	// Expose /debug endpoints and data only if the enableDebugServer flag is enabled
	var debugServer debugger.DebugServer
	if enableDebugServer {
		debugServer = debugger.NewDebugServer(certDebugger, xdsServer, meshCatalog, kubeConfig, kubeClient, cfg)
	}

	funcProbes := []health.Probes{xdsServer}
	httpProbes := getHTTPHealthProbes()
	httpServer := httpserver.NewHTTPServer(funcProbes, httpProbes, metricsStore, constants.MetricsServerPort, debugServer)
	httpServer.Start()

	// Wait for exit handler signal
	<-stop

	log.Info().Msg("Goodbye!")
}

func getHTTPHealthProbes() []health.HTTPProbe {
	return []health.HTTPProbe{
		{
			// HTTP probe on the sidecar injector webhook's port
			URL: joinURL(fmt.Sprintf("https://%s:%d", constants.LocalhostIPAddress, injectorConfig.ListenPort),
				injector.WebhookHealthPath),
			Protocol: health.ProtocolHTTPS,
		},
	}
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

func createOrUpdateCABundleKubernetesSecret(kubeClient clientset.Interface, certManager certificate.Manager, namespace, caBundleSecretName string) error {
	if caBundleSecretName == "" {
		log.Info().Msg("No name provided for CA bundle k8s secret. Skip creation of secret")
		return nil
	}

	ca, err := certManager.GetRootCertificate()
	if err != nil {
		log.Error().Err(err).Msgf("Error getting root certificate")
		return nil
	}

	return saveOrUpdateSecretToKubernetes(kubeClient, ca, namespace, caBundleSecretName, nil)
}

func saveOrUpdateSecretToKubernetes(kubeClient clientset.Interface, ca certificate.Certificater, namespace, caBundleSecretName string, privKey []byte) error {
	secretData := map[string][]byte{
		constants.KubernetesOpaqueSecretCAKey:        ca.GetCertificateChain(),
		constants.KubernetesOpaqueSecretCAExpiration: []byte(ca.GetExpiration().Format(constants.TimeDateLayout)),
	}

	if privKey != nil {
		secretData[constants.KubernetesOpaqueSecretRootPrivateKeyKey] = privKey
	}

	existingSecret, getErr := kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), caBundleSecretName, metav1.GetOptions{})

	// If Kubernetes Secret doesn't exist, create a new one.
	if apierrors.IsNotFound(getErr) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      caBundleSecretName,
				Namespace: namespace,
			},
			Data: secretData,
		}

		if _, err := kubeClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{}); err != nil {
			log.Error().Err(err).Msgf("Error creating CA bundle Kubernetes secret %s in namespace %s", caBundleSecretName, namespace)
			return err
		}

		log.Info().Msgf("Created CA bundle Kubernetes secret %s in namespace %s", caBundleSecretName, namespace)

		return nil
	}

	if getErr != nil {
		log.Error().Err(getErr).Msgf("Error getting CA bundle Kubernetes secret %s in namespace %s", caBundleSecretName, namespace)
		return getErr
	}

	log.Info().Msgf("Updating existing CA bundle Kubernetes secret %s in namespace %s", caBundleSecretName, namespace)

	// Override or add CA bundle to existing Secret
	for key, datum := range secretData {
		existingSecret.Data[key] = datum
	}

	// Secret already exists, so update
	if _, err := kubeClient.CoreV1().Secrets(namespace).Update(context.Background(), existingSecret, metav1.UpdateOptions{}); err != nil {
		log.Error().Err(err).Msgf("Error updating CA bundle Kubernetes secret %s in namespace %s", caBundleSecretName, namespace)
		return err
	}

	return nil
}

func getCertificateManager(kubeClient kubernetes.Interface, kubeConfig *rest.Config) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	switch *osmCertificateManagerKind {
	case tresorKind:
		return getTresorOSMCertificateManager(kubeClient, enableDebugServer)
	case vaultKind:
		return getHashiVaultOSMCertificateManager(enableDebugServer)
	case certmanagerKind:
		return getCertManagerOSMCertificateManager(kubeClient, kubeConfig, enableDebugServer)
	default:
		return nil, nil, fmt.Errorf("Unsupported Certificate Manager %s", *osmCertificateManagerKind)
	}
}

func joinURL(baseURL string, paths ...string) string {
	p := path.Join(paths...)
	return fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), strings.TrimLeft(p, "/"))
}
