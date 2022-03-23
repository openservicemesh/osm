// Package main implements the main entrypoint for osm-injector and utility routines to
// bootstrap the various internal components of osm-injector.
// osm-injector provides the automatic sidecar injection capability in OSM.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/reconciler"

	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/httpserver"
	httpserverconstants "github.com/openservicemesh/osm/pkg/httpserver/constants"
	"github.com/openservicemesh/osm/pkg/injector"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/version"
)

var (
	verbosity          string
	meshName           string // An ID that uniquely identifies an OSM instance
	kubeConfigFile     string
	osmNamespace       string
	webhookConfigName  string
	caBundleSecretName string
	osmMeshConfigName  string
	webhookTimeout     int32
	osmVersion         string

	injectorConfig injector.Config

	certProviderKind string

	enableReconciler bool

	tresorOptions      providers.TresorOptions
	vaultOptions       providers.VaultOptions
	certManagerOptions providers.CertManagerOptions

	osmContainerPullPolicy string

	scheme = runtime.NewScheme()
)

var (
	flags = pflag.NewFlagSet(`osm-injector`, pflag.ExitOnError)
	log   = logger.New("osm-injector/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "OSM mesh name")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&osmNamespace, "osm-namespace", "", "Namespace to which OSM belongs to.")
	flags.StringVar(&webhookConfigName, "webhook-config-name", "", "Name of the MutatingWebhookConfiguration to be configured by osm-injector")
	flags.Int32Var(&webhookTimeout, "webhook-timeout", int32(20), "Timeout of the MutatingWebhookConfiguration")
	flags.StringVar(&osmMeshConfigName, "osm-config-name", "osm-mesh-config", "Name of the OSM MeshConfig")
	flags.StringVar(&osmVersion, "osm-version", "", "Version of OSM")

	// sidecar injector options
	flags.IntVar(&injectorConfig.ListenPort, "webhook-port", constants.InjectorWebhookPort, "Webhook port for sidecar-injector")

	// Generic certificate manager/provider options
	flags.StringVar(&certProviderKind, "certificate-manager", providers.TresorKind.String(), fmt.Sprintf("Certificate manager, one of [%v]", providers.ValidCertificateProviders))
	flags.StringVar(&caBundleSecretName, "ca-bundle-secret-name", "", "Name of the Kubernetes Secret for the OSM CA bundle")

	// Vault certificate manager/provider options
	flags.StringVar(&vaultOptions.VaultProtocol, "vault-protocol", "http", "Host name of the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultHost, "vault-host", "vault.default.svc.cluster.local", "Host name of the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultToken, "vault-token", "", "Secret token for the the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultRole, "vault-role", "openservicemesh", "Name of the Vault role dedicated to Open Service Mesh")
	flags.IntVar(&vaultOptions.VaultPort, "vault-port", 8200, "Port of the Hashi Vault")

	// Cert-manager certificate manager/provider options
	flags.StringVar(&certManagerOptions.IssuerName, "cert-manager-issuer-name", "osm-ca", "cert-manager issuer name")
	flags.StringVar(&certManagerOptions.IssuerKind, "cert-manager-issuer-kind", "Issuer", "cert-manager issuer kind")
	flags.StringVar(&certManagerOptions.IssuerGroup, "cert-manager-issuer-group", "cert-manager.io", "cert-manager issuer group")

	// Reconciler options
	flags.BoolVar(&enableReconciler, "enable-reconciler", false, "Enable reconciler for CDRs, mutating webhook and validating webhook")

	flags.StringVar(&osmContainerPullPolicy, "osm-container-pull-policy", "", "The pullPolicy to use for injected init and healthcheck containers")

	_ = clientgoscheme.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting osm-injector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}
	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", kubeConfigFile)
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	policyClient := policyClientset.NewForConfigOrDie(kubeConfig)

	// Initialize the generic Kubernetes event recorder and associate it with the osm-injector pod resource
	injectorPod, err := getInjectorPod(kubeClient)
	if err != nil {
		log.Fatal().Msg("Error fetching osm-injector pod")
	}
	eventRecorder := events.GenericEventRecorder()
	if err := eventRecorder.Initialize(injectorPod, kubeClient, osmNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	stop := signals.RegisterExitHandlers()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the default metrics store
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.CertIssuedCount,
		metricsstore.DefaultMetricsStore.CertIssuedTime,
		metricsstore.DefaultMetricsStore.ErrCodeCounter,
		metricsstore.DefaultMetricsStore.HTTPResponseTotal,
		metricsstore.DefaultMetricsStore.HTTPResponseDuration,
		metricsstore.DefaultMetricsStore.AdmissionWebhookResponseTotal,
	)

	msgBroker := messaging.NewBroker(stop)

	// Initialize Configurator to retrieve mesh specific config
	cfg := configurator.NewConfigurator(configClientset.NewForConfigOrDie(kubeConfig), stop, osmNamespace, osmMeshConfigName, msgBroker)

	// Initialize kubernetes.Controller to watch kubernetes resources
	kubeController, err := k8s.NewKubernetesController(kubeClient, policyClient, meshName, stop, msgBroker, k8s.Namespaces)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Kubernetes Controller")
	}

	// Intitialize certificate manager/provider
	certProviderConfig := providers.NewCertificateProviderConfig(kubeClient, kubeConfig, cfg, providers.Kind(certProviderKind), osmNamespace,
		caBundleSecretName, tresorOptions, vaultOptions, certManagerOptions, msgBroker)

	certManager, _, err := certProviderConfig.GetCertificateManager()
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCertificateManager,
			"Error initializing certificate manager of kind %s", certProviderKind)
	}

	// Initialize the sidecar injector webhook
	if err := injector.NewMutatingWebhook(injectorConfig, kubeClient, certManager, kubeController, meshName, osmNamespace, webhookConfigName, osmVersion, webhookTimeout, enableReconciler, stop, cfg, corev1.PullPolicy(osmContainerPullPolicy)); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating sidecar injector webhook")
	}

	/*
	 * Initialize osm-injector's HTTP server
	 */
	httpServer := httpserver.NewHTTPServer(constants.OSMHTTPServerPort)
	// Metrics
	httpServer.AddHandler(httpserverconstants.MetricsPath, metricsstore.DefaultMetricsStore.Handler())
	// Version
	httpServer.AddHandler(httpserverconstants.VersionPath, version.GetVersionHandler())
	// Start HTTP server
	err = httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start OSM metrics/probes HTTP server")
	}

	// Start the global log level watcher that updates the log level dynamically
	go k8s.WatchAndUpdateLogLevel(msgBroker, stop)

	if enableReconciler {
		log.Info().Msgf("OSM reconciler enabled for sidecar injector webhook")
		err = reconciler.NewReconcilerClient(kubeClient, nil, meshName, osmVersion, stop, reconciler.MutatingWebhookInformerKey)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating reconciler client to reconcile sidecar injector webhook")
		}
	}

	<-stop
	log.Info().Msgf("Stopping osm-injector %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// getInjectorPod returns the osm-injector pod spec.
// The pod name is inferred from the 'INJECTOR_POD_NAME' env variable which is set during deployment.
func getInjectorPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podName := os.Getenv("INJECTOR_POD_NAME")
	if podName == "" {
		return nil, errors.New("INJECTOR_POD_NAME env variable cannot be empty")
	}

	pod, err := kubeClient.CoreV1().Pods(osmNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingInjectorPod)).
			Msgf("Error retrieving osm-injector pod %s", podName)
		return nil, err
	}

	return pod, nil
}

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if meshName == "" {
		return errors.New("Please specify the mesh name using --mesh-name")
	}

	if osmNamespace == "" {
		return errors.New("Please specify the OSM namespace using --osm-namespace")
	}

	if webhookConfigName == "" {
		return errors.Errorf("Please specify the mutatingwebhookconfiguration name using --webhook-config-name value")
	}

	if caBundleSecretName == "" {
		return errors.Errorf("Please specify the CA bundle secret name using --ca-bundle-secret-name")
	}

	return nil
}
