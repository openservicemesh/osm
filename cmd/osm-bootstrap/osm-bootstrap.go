// Package main implements the main entrypoint for osm-bootstrap and utility routines to
// bootstrap the various internal components of osm-bootstrap.
// osm-bootstrap provides crd conversion capability in OSM.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiScheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/crdconversion"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/httpserver"
	httpserverconstants "github.com/openservicemesh/osm/pkg/httpserver/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
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
	osmVersion         string

	crdConverterConfig crdconversion.Config

	certProviderKind string

	tresorOptions      providers.TresorOptions
	vaultOptions       providers.VaultOptions
	certManagerOptions providers.CertManagerOptions

	enableReconciler bool
	onlyCRDs         bool

	scheme  = runtime.NewScheme()
	crdPath = "/osm-crds"
)

var (
	flags = pflag.NewFlagSet(`osm-bootstrap`, pflag.ExitOnError)
	log   = logger.New(constants.OSMBootstrapName)
)

type bootstrap struct {
	kubeClient       kubernetes.Interface
	meshConfigClient configClientset.Interface
	namespace        string
}

func init() {
	flags.StringVar(&meshName, "mesh-name", "", "OSM mesh name")
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&osmNamespace, "osm-namespace", "", "Namespace to which OSM belongs to.")
	flags.StringVar(&osmMeshConfigName, "osm-config-name", "osm-mesh-config", "Name of the OSM MeshConfig")
	flags.StringVar(&osmVersion, "osm-version", "", "Version of OSM")

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
	flags.BoolVar(&onlyCRDs, "only-crds", false, "Only install CRDs and then exit")

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

	if onlyCRDs { // whether there's an error or not, we return here
		err := installCRDs(crdClient)
		if err != nil {
			log.Fatal().Err(err).Msg("Error installing CRDs in the cluster")
		}
		return
	}

	apiServerClient := clientset.NewForConfigOrDie(kubeConfig)
	configClient, err := configClientset.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not access Kubernetes cluster, check kubeconfig.")
		return
	}

	bootstrap := bootstrap{
		kubeClient:       kubeClient,
		meshConfigClient: configClient,
		namespace:        osmNamespace,
	}

	err = bootstrap.ensureMeshConfig()
	if err != nil {
		log.Fatal().Err(err).Msgf("Error setting up default MeshConfig %s from ConfigMap %s", meshConfigName, presetMeshConfigName)
		return
	}

	err = bootstrap.initiatilizeKubernetesEventsRecorder()
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing Kubernetes events recorder")
	}

	stop := signals.RegisterExitHandlers()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the default metrics store
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.ErrCodeCounter,
		metricsstore.DefaultMetricsStore.HTTPResponseTotal,
		metricsstore.DefaultMetricsStore.HTTPResponseDuration,
		metricsstore.DefaultMetricsStore.ConversionWebhookResourceTotal,
	)

	msgBroker := messaging.NewBroker(stop)

	// Initialize Configurator to retrieve mesh specific config
	cfg := configurator.NewConfigurator(configClient, stop, osmNamespace, osmMeshConfigName, msgBroker)

	// Intitialize certificate manager/provider
	certProviderConfig := providers.NewCertificateProviderConfig(kubeClient, kubeConfig, cfg, providers.Kind(certProviderKind), osmNamespace,
		caBundleSecretName, tresorOptions, vaultOptions, certManagerOptions, msgBroker)

	certManager, _, err := certProviderConfig.GetCertificateManager()
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCertificateManager,
			"Error initializing certificate manager of kind %s", certProviderKind)
	}

	// Initialize the crd conversion webhook server to support the conversion of OSM's CRDs
	crdConverterConfig.ListenPort = constants.CRDConversionWebhookPort
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
		err = reconciler.NewReconcilerClient(kubeClient, apiServerClient, meshName, osmVersion, stop, reconciler.CrdInformerKey)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating reconciler client for custom resource definitions")
		}
	}

	<-stop
	log.Info().Msgf("Stopping osm-bootstrap %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func (b *bootstrap) createDefaultMeshConfig() error {
	// find presets config map to build the default MeshConfig from that
	presetsConfigMap, err := b.kubeClient.CoreV1().ConfigMaps(b.namespace).Get(context.TODO(), presetMeshConfigName, metav1.GetOptions{})

	// If the presets MeshConfig could not be loaded return the error
	if err != nil {
		return err
	}

	// Create a default meshConfig
	defaultMeshConfig := buildDefaultMeshConfig(presetsConfigMap)
	if _, err := b.meshConfigClient.ConfigV1alpha2().MeshConfigs(b.namespace).Create(context.TODO(), defaultMeshConfig, metav1.CreateOptions{}); err == nil {
		log.Info().Msgf("MeshConfig (%s) created in namespace %s", meshConfigName, b.namespace)
		return nil
	}

	if apierrors.IsAlreadyExists(err) {
		log.Info().Msgf("MeshConfig already exists in %s. Skip creating.", b.namespace)
		return nil
	}

	return err
}

func (b *bootstrap) ensureMeshConfig() error {
	_, err := b.meshConfigClient.ConfigV1alpha2().MeshConfigs(b.namespace).Get(context.TODO(), meshConfigName, metav1.GetOptions{})
	if err == nil {
		return nil // default meshConfig was found
	}

	if apierrors.IsNotFound(err) {
		// create a default mesh config since it was not found
		if err = b.createDefaultMeshConfig(); err != nil {
			return err
		}
	}

	return err
}

// initiatilizeKubernetesEventsRecorder initializes the generic Kubernetes event recorder and associates it with
//	the osm-bootstrap pod resource. The events recorder allows the osm-bootstap to publish Kubernets events to
// 	report fatal errors with initializing this application. These events will show up in the output of `kubectl get events`
func (b *bootstrap) initiatilizeKubernetesEventsRecorder() error {
	bootstrapPod, err := b.getBootstrapPod()
	if err != nil {
		return errors.Errorf("Error fetching osm-bootstrap pod: %s", err)
	}
	eventRecorder := events.GenericEventRecorder()
	return eventRecorder.Initialize(bootstrapPod, b.kubeClient, osmNamespace)
}

// getBootstrapPod returns the osm-bootstrap pod spec.
// The pod name is inferred from the 'BOOTSTRAP_POD_NAME' env variable which is set during deployment.
func (b *bootstrap) getBootstrapPod() (*corev1.Pod, error) {
	podName := os.Getenv("BOOTSTRAP_POD_NAME")
	if podName == "" {
		return nil, errors.New("BOOTSTRAP_POD_NAME env variable cannot be empty")
	}

	pod, err := b.kubeClient.CoreV1().Pods(b.namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error retrieving osm-bootstrap pod %s", podName)
		return nil, err
	}

	return pod, nil
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if onlyCRDs {
		return nil
	}

	if osmNamespace == "" {
		return errors.New("Please specify the OSM namespace using --osm-namespace")
	}

	if caBundleSecretName == "" {
		return errors.Errorf("Please specify the CA bundle secret name using --ca-bundle-secret-name")
	}

	return nil
}

func buildDefaultMeshConfig(presetMeshConfigMap *corev1.ConfigMap) *configv1alpha2.MeshConfig {
	presetMeshConfig := presetMeshConfigMap.Data[presetMeshConfigJSONKey]
	presetMeshConfigSpec := configv1alpha2.MeshConfigSpec{}
	err := json.Unmarshal([]byte(presetMeshConfig), &presetMeshConfigSpec)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error converting preset-mesh-config json string to meshConfig object")
	}

	return &configv1alpha2.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MeshConfig",
			APIVersion: "config.openservicemesh.io/configv1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: meshConfigName,
		},
		Spec: presetMeshConfigSpec,
	}
}

func installCRDs(crdClient apiclient.ApiextensionsV1Interface) error {
	tmpl, err := template.ParseGlob(fmt.Sprintf("%s/*.tpl", crdPath))
	if err != nil {
		log.Error().Err(err).Msg("Error parsing OSM CRD templates")
		return err
	}

	crdSettings := struct {
		EnableReconciler bool
		ReconcileLabel   string
	}{
		EnableReconciler: enableReconciler,
		ReconcileLabel:   constants.ReconcileLabel,
	}

	err = filepath.Walk(crdPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Error().Str("file", info.Name()).Err(err).Msg("Error walking crd directory path")
			return err
		}

		if !strings.Contains(info.Name(), ".tpl") {
			return nil
		}

		// We found a template; execute it
		var buffer bytes.Buffer

		err = tmpl.ExecuteTemplate(&buffer, info.Name(), crdSettings)
		if err != nil {
			return err
		}

		obj, gvk, err := apiScheme.Codecs.UniversalDeserializer().Decode(buffer.Bytes(), nil, nil)
		if err != nil {
			return err
		}

		switch crd := obj.(type) {
		case *v1.CustomResourceDefinitionList:
			if len(crd.Items) == 0 {
				log.Debug().Str("crd", info.Name()).Msg("skipping CRD definition with 0 YAML documents")
				return nil
			}

			for i := range crd.Items {
				r, err := crdClient.CustomResourceDefinitions().Create(context.Background(), &crd.Items[i], metav1.CreateOptions{})
				if err != nil {
					log.Error().Str("crd", info.Name()).Err(err).Msg("Error applying CRD to cluster")
					return err
				}
				log.Info().Str("crd", info.Name()).Msgf("%s.%s/%s created", strings.ToLower(r.GroupVersionKind().Kind), r.GroupVersionKind().Group, r.GetObjectMeta().GetName())
			}
		case *v1.CustomResourceDefinition:
			r, err := crdClient.CustomResourceDefinitions().Create(context.Background(), crd, metav1.CreateOptions{})
			if err != nil {
				log.Error().Str("crd", info.Name()).Err(err).Msg("Error applying CRD to cluster")
				return err
			}
			log.Info().Str("crd", info.Name()).Msgf("%s.%s/%s created", strings.ToLower(r.GroupVersionKind().Kind), r.GroupVersionKind().Group, r.GetObjectMeta().GetName())
		default:
			log.Error().Str("crd", info.Name()).Msgf("Invalid YAML manifest encountered in CRD directory: %q", gvk)
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Error executing CRD templates")
		return err
	}

	return nil
}
