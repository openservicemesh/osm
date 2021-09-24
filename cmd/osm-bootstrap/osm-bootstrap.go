// Package main implements the main entrypoint for osm-bootstrap and utility routines to
// bootstrap the various internal components of osm-bootstrap.
// osm-bootstrap provides crd conversion capability in OSM.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/crdconversion"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/httpserver"
	httpserverconstants "github.com/openservicemesh/osm/pkg/httpserver/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/reconciler"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/version"
)

const (
	meshConfigName          = "osm-mesh-config"
	presetMeshConfigName    = "preset-mesh-config"
	presetMeshConfigJSONKey = "preset-mesh-config.json"
)

var (
	verbosity          string
	osmNamespace       string
	caBundleSecretName string
	osmMeshConfigName  string
	meshName           string

	crdConverterConfig crdconversion.Config

	certProviderKind string

	tresorOptions      providers.TresorOptions
	vaultOptions       providers.VaultOptions
	certManagerOptions providers.CertManagerOptions

	enableReconciler bool

	scheme = runtime.NewScheme()
)

var (
	flags = pflag.NewFlagSet(`osm-bootstrap`, pflag.ExitOnError)
	log   = logger.New(constants.OSMBootstrapName)
)

func init() {
	flags.StringVar(&meshName, "mesh-name", "", "OSM mesh name")
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&osmNamespace, "osm-namespace", "", "Namespace to which OSM belongs to.")
	flags.StringVar(&osmMeshConfigName, "osm-config-name", "osm-mesh-config", "Name of the OSM MeshConfig")

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

	_ = clientgoscheme.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting osm-bootstrap %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating kube configs using in-cluster config")
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	crdClient := apiclient.NewForConfigOrDie(kubeConfig)
	apiServerClient := clientset.NewForConfigOrDie(kubeConfig)
	configClient, err := configClientset.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not access Kubernetes cluster, check kubeconfig.")
		return
	}

	presetMeshConfigMap, presetConfigErr := kubeClient.CoreV1().ConfigMaps(osmNamespace).Get(context.TODO(), presetMeshConfigName, metav1.GetOptions{})
	_, meshConfigErr := configClient.ConfigV1alpha1().MeshConfigs(osmNamespace).Get(context.TODO(), meshConfigName, metav1.GetOptions{})

	// If the presetMeshConfig could not be loaded and a default meshConfig doesn't exist, return the error
	if presetConfigErr != nil && apierrors.IsNotFound(meshConfigErr) {
		log.Fatal().Err(err).Msgf("Unable to create default meshConfig, as %s could not be found", presetMeshConfigName)
		return
	}

	// Create a default meshConfig
	defaultMeshConfig := createDefaultMeshConfig(presetMeshConfigMap)
	if createdMeshConfig, err := configClient.ConfigV1alpha1().MeshConfigs(osmNamespace).Create(context.TODO(), defaultMeshConfig, metav1.CreateOptions{}); err == nil {
		log.Info().Msgf("MeshConfig created in %s, %v", osmNamespace, createdMeshConfig)
	} else if apierrors.IsAlreadyExists(err) {
		log.Info().Msgf("MeshConfig already exists in %s. Skip creating.", osmNamespace)
	} else {
		log.Fatal().Err(err).Msgf("Error creating default MeshConfig")
	}

	// Initialize the generic Kubernetes event recorder and associate it with the osm-bootstrap pod resource
	bootstrapPod, err := getBootstrapPod(kubeClient)
	if err != nil {
		log.Fatal().Msg("Error fetching osm-bootstrap pod")
	}
	eventRecorder := events.GenericEventRecorder()
	if err := eventRecorder.Initialize(bootstrapPod, kubeClient, osmNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	stop := signals.RegisterExitHandlers()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the default metrics store
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.ErrCodeCounter,
	)

	// Initialize Configurator to retrieve mesh specific config
	cfg := configurator.NewConfigurator(configClientset.NewForConfigOrDie(kubeConfig), stop, osmNamespace, osmMeshConfigName)

	// Intitialize certificate manager/provider
	certProviderConfig := providers.NewCertificateProviderConfig(kubeClient, kubeConfig, cfg, providers.Kind(certProviderKind), osmNamespace,
		caBundleSecretName, tresorOptions, vaultOptions, certManagerOptions)

	certManager, _, err := certProviderConfig.GetCertificateManager()
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCertificateManager,
			"Error initializing certificate manager of kind %s", certProviderKind)
	}

	// Initialize the crd conversion webhook server to support the conversion of OSM's CRDs
	crdConverterConfig.ListenPort = 443
	if err := crdconversion.NewConversionWebhook(crdConverterConfig, kubeClient, crdClient, certManager, osmNamespace, enableReconciler, stop); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating crd conversion webhook")
	}

	/*
	 * Initialize osm-bootstrap's HTTP server
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

	if enableReconciler {
		log.Info().Msgf("OSM reconciler enabled for custom resource definitions")
		err = reconciler.NewReconcilerClient(kubeClient, apiServerClient, meshName, stop, reconciler.CrdInformerKey)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating reconciler client for custom resource definitions")
		}
	}

	<-stop
	log.Info().Msgf("Stopping osm-bootstrap %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// getBootstrapPod returns the osm-bootstrap pod spec.
// The pod name is inferred from the 'BOOTSTRAP_POD_NAME' env variable which is set during deployment.
func getBootstrapPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podName := os.Getenv("BOOTSTRAP_POD_NAME")
	if podName == "" {
		return nil, errors.New("BOOTSTRAP_POD_NAME env variable cannot be empty")
	}

	pod, err := kubeClient.CoreV1().Pods(osmNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving osm-bootstrap pod %s", podName)
		return nil, err
	}

	return pod, nil
}

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if osmNamespace == "" {
		return errors.New("Please specify the OSM namespace using --osm-namespace")
	}

	if caBundleSecretName == "" {
		return errors.Errorf("Please specify the CA bundle secret name using --ca-bundle-secret-name")
	}

	return nil
}

func createDefaultMeshConfig(presetMeshConfigMap *corev1.ConfigMap) *v1alpha1.MeshConfig {
	presetMeshConfig := presetMeshConfigMap.Data[presetMeshConfigJSONKey]
	presetMeshConfigSpec := v1alpha1.MeshConfigSpec{}
	err := json.Unmarshal([]byte(presetMeshConfig), &presetMeshConfigSpec)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error converting preset-mesh-config json string to meshConfig object")
	}

	return &v1alpha1.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MeshConfig",
			APIVersion: "config.openservicemesh.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: meshConfigName,
		},
		Spec: presetMeshConfigSpec,
	}
}
