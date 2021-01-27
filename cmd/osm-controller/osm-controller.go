package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"

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
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	reconciler "github.com/openservicemesh/osm/pkg/reconciler/mutatingwebhook"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/version"
)

const (
	caBundleSecretNameCLIParam     = "ca-bundle-secret-name"
	xdsServerCertificateCommonName = "ads"
)

var (
	verbosity            string
	meshName             string // An ID that uniquely identifies an OSM instance
	kubeConfigFile       string
	osmNamespace         string
	webhookConfigName    string
	caBundleSecretName   string
	osmConfigMapName     string
	metricsAddr          string
	enableLeaderElection bool

	injectorConfig injector.Config

	// feature flag options
	optionalFeatures featureflags.OptionalFeatures

	scheme = runtime.NewScheme()
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
	flags.StringVar(&caBundleSecretName, caBundleSecretNameCLIParam, "", "Name of the Kubernetes Secret for the OSM CA bundle")
	flags.StringVar(&osmConfigMapName, "osm-configmap-name", "osm-config", "Name of the OSM ConfigMap")

	// sidecar injector options
	flags.BoolVar(&injectorConfig.DefaultInjection, "default-injection", true, "Enable sidecar injection by default")
	flags.IntVar(&injectorConfig.ListenPort, "webhook-port", constants.InjectorWebhookPort, "Webhook port for sidecar-injector")
	flags.StringVar(&injectorConfig.InitContainerImage, "init-container-image", "", "InitContainer image")
	flags.StringVar(&injectorConfig.SidecarImage, "sidecar-image", "", "Sidecar proxy Container image")

	// feature flags
	flags.BoolVar(&optionalFeatures.Backpressure, "enable-backpressure-experimental", false, "Enable experimental backpressure feature")
	flags.BoolVar(&optionalFeatures.RoutesV2, "enable-routes-v2-experimental", false, "Enable experimental routes v2 feature")

	// k8s controller manager options
	// a k8s controller provided by the package "sigs.k8s.io/controller-runtime" helps to ensure that the state of a given k8s object is as per its desired state
	// a controller manager is responsible for running controllers, managing the life cycle of the controller and setting up common dependencies
	// metrics-addr is the endpoint for the performance metrics generated by the controller
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	// enable-leader-election is a flag to ensure there's only one manager for the controller
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting osm-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}
	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	if featureFlagsJSON, err := json.Marshal(featureflags.Features); err != nil {
		log.Error().Err(err).Msgf("Error marshaling feature flags struct: %+v", featureflags.Features)
	} else {
		log.Info().Msgf("Feature flags: %s", string(featureFlagsJSON))
	}

	featureflags.Initialize(optionalFeatures)
	events.GetPubSubInstance() // Just to generate the interface, single routine context

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", kubeConfigFile)
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	// Initialize the generic Kubernetes event recorder and associate it with the osm-controller pod resource
	controllerPod, err := getOSMControllerPod(kubeClient)
	if err != nil {
		log.Fatal().Msg("Error fetching osm-controller pod")
	}
	eventRecorder := events.GenericEventRecorder()
	if err := eventRecorder.Initialize(controllerPod, kubeClient, osmNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	stop := signals.RegisterExitHandlers()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the default metrics store and start it.
	metricsstore.DefaultMetricsStore.Start()

	// This component will be watching the OSM ConfigMap and will make it
	// to the rest of the components.
	cfg := configurator.NewConfigurator(kubernetes.NewForConfigOrDie(kubeConfig), stop, osmNamespace, osmConfigMapName)
	configMap, err := cfg.GetConfigMap()
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing ConfigMap %s", osmConfigMapName)
	}
	log.Info().Msgf("Initial ConfigMap %s: %s", osmConfigMapName, string(configMap))

	kubernetesClient, err := k8s.NewKubernetesController(kubeClient, meshName, stop)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Kubernetes Controller")
	}

	meshSpec, err := smi.NewMeshSpecClient(kubeConfig, kubeClient, osmNamespace, kubernetesClient, stop)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating MeshSpec")
	}

	certManager, certDebugger, err := getCertificateManager(kubeClient, kubeConfig, cfg)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCertificateManager,
			"Error fetching certificate manager of kind %s", *osmCertificateManagerKind)
	}

	if caBundleSecretName == "" {
		log.Info().Msgf("CA bundle will not be exported to a k8s secret (no --%s provided)", caBundleSecretNameCLIParam)
	} else {
		if err := createOrUpdateCABundleKubernetesSecret(kubeClient, certManager, osmNamespace, caBundleSecretName); err != nil {
			log.Error().Err(err).Msgf("Error exporting CA bundle into Kubernetes secret with name %s", caBundleSecretName)
		}
	}

	kubeProvider, err := kube.NewProvider(kubeClient, kubernetesClient, constants.KubeProviderName, cfg)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Kubernetes endpoints provider")
	}

	endpointsProviders := []endpoint.Provider{kubeProvider}

	ingressClient, err := ingress.NewIngressClient(kubeClient, kubernetesClient, stop, cfg)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Ingress monitor client")
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
	if err := injector.NewMutatingWebhook(injectorConfig, kubeClient, certManager, meshCatalog, kubernetesClient, meshName, osmNamespace, webhookConfigName, stop, cfg); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating sidecar injector webhook")
	}

	// Create the configMap validating webhook
	if err := configurator.NewValidatingWebhook(kubeClient, certManager, osmNamespace, webhookConfigName, stop); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating osm-config validating webhook")
	}

	adsCert, err := certManager.IssueCertificate(xdsServerCertificateCommonName, constants.XDSCertificateValidityPeriod)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.CertificateIssuanceFailure, "Error issuing XDS certificate to ADS server")
	}

	// Create and start the ADS gRPC service
	xdsServer := ads.NewADSServer(meshCatalog, cfg.IsDebugServerEnabled(), osmNamespace, cfg, certManager)
	if err := xdsServer.Start(ctx, cancel, *port, adsCert); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error initializing ADS server")
	}

	if err := createControllerManagerForOSMResources(certManager); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating controller manager to reconcile OSM resources")
	}

	// Initialize OSM's http service server
	httpServer := httpserver.NewHTTPServer(constants.OSMServicePort)

	// Health/Liveness probes
	funcProbes := []health.Probes{xdsServer}
	httpServer.AddHandlers(map[string]http.Handler{
		"/health/ready": health.ReadinessHandler(funcProbes, getHTTPHealthProbes()),
		"/health/alive": health.LivenessHandler(funcProbes, getHTTPHealthProbes()),
	})
	// Metrics
	httpServer.AddHandler("/metrics", metricsstore.DefaultMetricsStore.Handler())
	// Version
	httpServer.AddHandler("/version", version.GetVersionHandler())

	// Start HTTP server
	err = httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start OSM metrics/probes HTTP server")
	}

	// Create DebugServer and start its config event listener.
	// Listener takes care to start and stop the debug server as appropriate
	debugConfig := debugger.NewDebugConfig(certDebugger, xdsServer, meshCatalog, kubeConfig, kubeClient, cfg, kubernetesClient)
	debugConfig.StartDebugServerConfigListener()

	<-stop
	log.Info().Msgf("Stopping osm-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
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

	return saveOrUpdateSecretToKubernetes(kubeClient, ca, namespace, caBundleSecretName)
}

func saveOrUpdateSecretToKubernetes(kubeClient clientset.Interface, ca certificate.Certificater, namespace, caBundleSecretName string) error {
	secretData := map[string][]byte{
		constants.KubernetesOpaqueSecretCAKey:             ca.GetCertificateChain(),
		constants.KubernetesOpaqueSecretCAExpiration:      []byte(ca.GetExpiration().Format(constants.TimeDateLayout)),
		constants.KubernetesOpaqueSecretRootPrivateKeyKey: ca.GetPrivateKey(),
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

func getCertificateManager(kubeClient kubernetes.Interface, kubeConfig *rest.Config, cfg configurator.Configurator) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	switch *osmCertificateManagerKind {
	case tresorKind:
		return getTresorOSMCertificateManager(kubeClient, cfg)
	case vaultKind:
		return getHashiVaultOSMCertificateManager(cfg)
	case certmanagerKind:
		return getCertManagerOSMCertificateManager(kubeClient, kubeConfig, cfg)
	default:
		return nil, nil, fmt.Errorf("Unsupported Certificate Manager %s", *osmCertificateManagerKind)
	}
}

func joinURL(baseURL string, paths ...string) string {
	p := path.Join(paths...)
	return fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), strings.TrimLeft(p, "/"))
}

// getOSMControllerPod returns the osm-controller pod.
// The pod name is inferred from the 'CONTROLLER_POD_NAME' env variable which is set during deployment.
func getOSMControllerPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podName := os.Getenv("CONTROLLER_POD_NAME")
	if podName == "" {
		return nil, errors.New("CONTROLLER_POD_NAME env variable cannot be empty")
	}

	pod, err := kubeClient.CoreV1().Pods(osmNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving osm-controller pod %s", podName)
		return nil, err
	}

	return pod, nil
}

//Setting up k8s controller manager to reconcile OSM resources
func createControllerManagerForOSMResources(certManager certificate.Manager) error {
	log.Info().Msg("Setting up controller manager to reconcile OSM resources")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		Namespace:          osmNamespace,
	})
	if err != nil {
		log.Error().Err(err).Msg("Error starting up controller manager")
		return err
	}

	log.Info().Msg("Successfully setup controller for resource reconciliation")
	log.Info().Msg("Setting up mutatingWebhookConfiguration reconciler")

	// controller logic is implemented by reconciler
	// Adding a reconciler for OSM's mutatingwehbookconfiguration
	if err = (&reconciler.MutatingWebhookConfigurationReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		OsmWebhook:   fmt.Sprintf("osm-webhook-%s", meshName),
		OsmNamespace: osmNamespace,
		CertManager:  certManager,
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Msg("Error creating reconcile controller for MutatingWebhookConfiguration")
		return err
	}

	go func() {
		// mgr.Start() below will block until stopped
		// See: https://github.com/kubernetes-sigs/controller-runtime/blob/release-0.6/pkg/manager/internal.go#L507-L514
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			log.Error().Err(err).Msg("problem running manager for controller")
		}
	}()

	return nil
}
