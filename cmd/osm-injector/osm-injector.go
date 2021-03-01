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
	"k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/httpserver"
	"github.com/openservicemesh/osm/pkg/injector"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
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
	osmConfigMapName   string

	injectorConfig injector.Config

	certProviderKind string

	tresorOptions      providers.TresorOptions
	vaultOptions       providers.VaultOptions
	certManagerOptions providers.CertManagerOptions

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
	flags.StringVar(&osmConfigMapName, "osm-configmap-name", "osm-config", "Name of the OSM ConfigMap")

	// sidecar injector options
	flags.IntVar(&injectorConfig.ListenPort, "webhook-port", constants.InjectorWebhookPort, "Webhook port for sidecar-injector")
	flags.StringVar(&injectorConfig.InitContainerImage, "init-container-image", "", "InitContainer image")
	flags.StringVar(&injectorConfig.SidecarImage, "sidecar-image", "", "Sidecar proxy Container image")

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

	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
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
		metricsstore.DefaultMetricsStore.InjectorRqTime,
		metricsstore.DefaultMetricsStore.InjectorSidecarCount,
		metricsstore.DefaultMetricsStore.CertIssuedCount,
		metricsstore.DefaultMetricsStore.CertIssuedTime,
	)

	// Initialize Configurator to watch osm-config ConfigMap
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
	configMap, err := cfg.GetConfigMap()
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing ConfigMap %s", osmConfigMapName)
	}
	log.Debug().Msgf("Initial ConfigMap %s: %s", osmConfigMapName, string(configMap))

	// Initialize kubernetes.Controller to watch kubernetes resources
	kubeController, err := k8s.NewKubernetesController(kubeClient, meshName, stop, k8s.Namespaces)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Kubernetes Controller")
	}

	// Intitialize certificate manager/provider
	certProviderConfig := providers.NewCertificateProviderConfig(kubeClient, kubeConfig, cfg, providers.Kind(certProviderKind), osmNamespace,
		caBundleSecretName, tresorOptions, vaultOptions, certManagerOptions)

	certManager, _, err := certProviderConfig.GetCertificateManager()
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCertificateManager,
			"Error initializing certificate manager of kind %s", certProviderKind)
	}

	// Initialize the sidecar injector webhook
	if err := injector.NewMutatingWebhook(injectorConfig, kubeClient, certManager, kubeController, meshName, osmNamespace, webhookConfigName, stop, cfg); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating sidecar injector webhook")
	}

	/*
	 * Initialize osm-injector's HTTP server
	 */
	httpServer := httpserver.NewHTTPServer(constants.OSMHTTPServerPort)
	// Metrics
	httpServer.AddHandler("/metrics", metricsstore.DefaultMetricsStore.Handler())
	// Version
	httpServer.AddHandler("/version", version.GetVersionHandler())
	// Start HTTP server
	err = httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start OSM metrics/probes HTTP server")
	}

	// Initialize the reconciler for the injector's MutatingWebhookConfiguration
	if err := createReconciler(kubeClient); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating controller manager to reconcile sidecar injector webhook config")
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
		log.Error().Err(err).Msgf("Error retrieving osm-injector pod %s", podName)
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

	if injectorConfig.InitContainerImage == "" {
		return errors.New("Please specify the init container image using --init-container-image")
	}

	if injectorConfig.SidecarImage == "" {
		return errors.Errorf("Please specify the sidecar image using --sidecar-image")
	}

	if webhookConfigName == "" {
		return errors.Errorf("Please specify the mutatingwebhookconfiguration name using --webhook-config-name value")
	}

	if caBundleSecretName == "" {
		return errors.Errorf("Please specify the CA bundle secret name using --ca-bundle-secret-name")
	}

	return nil
}
