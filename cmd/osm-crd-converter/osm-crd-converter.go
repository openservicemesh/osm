// Package main implements the main entrypoint for osm-crd-converter and utility routines to
// bootstrap the various internal components of osm-crd-converter.
// osm-crd-converter provides crd conversion capability in OSM.
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
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/crdconversion"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/httpserver"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/version"
)

var (
	verbosity          string
	osmNamespace       string
	caBundleSecretName string
	osmMeshConfigName  string

	crdConverterConfig crdconversion.Config

	certProviderKind string

	tresorOptions      providers.TresorOptions
	vaultOptions       providers.VaultOptions
	certManagerOptions providers.CertManagerOptions

	scheme = runtime.NewScheme()
)

var (
	flags = pflag.NewFlagSet(`osm-crd-converter`, pflag.ExitOnError)
	log   = logger.New("osm-crd-converter/main")
)

func init() {
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

	_ = clientgoscheme.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting osm-crd-converter %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
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

	// Initialize the generic Kubernetes event recorder and associate it with the osm-crd-converter pod resource
	crdConverterPod, err := getCrdConverterPod(kubeClient)
	if err != nil {
		log.Fatal().Msg("Error fetching osm-crd-converter pod")
	}
	eventRecorder := events.GenericEventRecorder()
	if err := eventRecorder.Initialize(crdConverterPod, kubeClient, osmNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	stop := signals.RegisterExitHandlers()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Initialize the crd conversion webhook
	crdConverterConfig.ListenPort = 443
	if err := crdconversion.NewConversionWebhook(crdConverterConfig, kubeClient, crdClient, certManager, osmNamespace, stop); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating crd conversion webhook")
	}

	/*
	 * Initialize osm-crd-converter's HTTP server
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

	<-stop
	log.Info().Msgf("Stopping osm-crd-converter %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// getCrdConverterPod returns the osm-crd-converter pod spec.
// The pod name is inferred from the 'CRD_CONVERTER_POD_NAME' env variable which is set during deployment.
func getCrdConverterPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podName := os.Getenv("CRD_CONVERTER_POD_NAME")
	if podName == "" {
		return nil, errors.New("CRD_CONVERTER_POD_NAME env variable cannot be empty")
	}

	pod, err := kubeClient.CoreV1().Pods(osmNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving osm-crd-converter pod %s", podName)
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
