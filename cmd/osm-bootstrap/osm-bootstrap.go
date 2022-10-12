// Package main implements the main entrypoint for osm-bootstrap and utility routines to
// bootstrap the various internal components of osm-bootstrap.
// osm-bootstrap provides crd conversion capability in OSM.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/util"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/health"

	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/httpserver"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/reconciler"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/version"
)

const (
	meshConfigName                   = "osm-mesh-config"
	presetMeshConfigName             = "preset-mesh-config"
	presetMeshConfigJSONKey          = "preset-mesh-config.json"
	presetMeshRootCertificateName    = "preset-mesh-root-certificate"
	presetMeshRootCertificateJSONKey = "preset-mesh-root-certificate.json"
)

var (
	verbosity          string
	osmNamespace       string
	caBundleSecretName string
	osmMeshConfigName  string
	meshName           string
	osmVersion         string
	trustDomain        string

	certProviderKind          string
	enableMeshRootCertificate bool

	vaultOptions       providers.VaultOptions
	certManagerOptions providers.CertManagerOptions

	enableReconciler bool

	scheme = runtime.NewScheme()
)

var (
	flags = pflag.NewFlagSet(`osm-bootstrap`, pflag.ExitOnError)
	log   = logger.New(constants.OSMBootstrapName)
)

type bootstrap struct {
	kubeClient   kubernetes.Interface
	configClient configClientset.Interface
	namespace    string
}

func init() {
	flags.StringVar(&meshName, "mesh-name", "", "OSM mesh name")
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&osmNamespace, "osm-namespace", "", "Namespace to which OSM belongs to.")
	flags.StringVar(&osmMeshConfigName, "osm-config-name", "osm-mesh-config", "Name of the OSM MeshConfig")
	flags.StringVar(&osmVersion, "osm-version", "", "Version of OSM")

	// Generic certificate manager/provider options
	flags.StringVar(&certProviderKind, "certificate-manager", providers.TresorKind.String(), fmt.Sprintf("Certificate manager, one of [%v]", providers.ValidCertificateProviders))
	flags.BoolVar(&enableMeshRootCertificate, "enable-mesh-root-certificate", false, "Enable unsupported MeshRootCertificate to create the OSM Certificate Manager")
	flags.StringVar(&caBundleSecretName, "ca-bundle-secret-name", "", "Name of the Kubernetes Secret for the OSM CA bundle")

	// TODO (#4502): Remove when we add full MRC support
	flags.StringVar(&trustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")

	// Vault certificate manager/provider options
	flags.StringVar(&vaultOptions.VaultProtocol, "vault-protocol", "http", "Host name of the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultHost, "vault-host", "vault.default.svc.cluster.local", "Host name of the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultToken, "vault-token", "", "Secret token for the the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultRole, "vault-role", "openservicemesh", "Name of the Vault role dedicated to Open Service Mesh")
	flags.IntVar(&vaultOptions.VaultPort, "vault-port", 8200, "Port of the Hashi Vault")
	flags.StringVar(&vaultOptions.VaultTokenSecretName, "vault-token-secret-name", "", "Name of the secret storing the Vault token used in OSM")
	flags.StringVar(&vaultOptions.VaultTokenSecretKey, "vault-token-secret-key", "", "Key for the vault token used in OSM")

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

	bootstrap := bootstrap{
		kubeClient:   kubeClient,
		configClient: configClient,
		namespace:    osmNamespace,
	}

	applyOrUpdateCRDs(crdClient)

	err = bootstrap.ensureMeshConfig()
	if err != nil {
		log.Fatal().Err(err).Msgf("Error setting up default MeshConfig %s from ConfigMap %s", meshConfigName, presetMeshConfigName)
		return
	}

	if enableMeshRootCertificate {
		err = bootstrap.ensureMeshRootCertificate()
		if err != nil {
			log.Fatal().Err(err).Msgf("Error setting up default MeshRootCertificate %s from ConfigMap %s", constants.DefaultMeshRootCertificateName, presetMeshRootCertificateName)
			return
		}
	}

	err = bootstrap.initiatilizeKubernetesEventsRecorder()
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing Kubernetes events recorder")
	}

	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := signals.RegisterExitHandlers(cancel)

	// Start the default metrics store
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.ErrCodeCounter,
		metricsstore.DefaultMetricsStore.HTTPResponseTotal,
		metricsstore.DefaultMetricsStore.HTTPResponseDuration,
		metricsstore.DefaultMetricsStore.ReconciliationTotal,
	)

	version.SetMetric()
	/*
	 * Initialize osm-bootstrap's HTTP server
	 */
	if enableReconciler {
		log.Info().Msgf("OSM reconciler enabled for custom resource definitions")
		err = reconciler.NewReconcilerClient(kubeClient, apiServerClient, meshName, osmVersion, stop, reconciler.CrdInformerKey)
		if err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating reconciler client for custom resource definitions")
			log.Fatal().Err(err).Msgf("Failed to create reconcile client for custom resource definitions")
		}
	}

	/*
	 * Initialize osm-bootstrap's HTTP server
	 */
	httpServer := httpserver.NewHTTPServer(constants.OSMHTTPServerPort)
	// Metrics
	httpServer.AddHandler(constants.MetricsPath, metricsstore.DefaultMetricsStore.Handler())
	// Version
	httpServer.AddHandler(constants.VersionPath, version.GetVersionHandler())

	httpServer.AddHandler(constants.WebhookHealthPath, http.HandlerFunc(health.SimpleHandler))

	// Start HTTP server
	err = httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start OSM metrics/probes HTTP server")
	}

	<-stop
	cancel()
	log.Info().Msgf("Stopping osm-bootstrap %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func applyOrUpdateCRDs(crdClient *apiclient.ApiextensionsV1Client) {
	crdFiles, err := filepath.Glob("/osm-crds/*.yaml")

	if err != nil {
		log.Fatal().Err(err).Msgf("error reading files from /osm-crds")
	}

	scheme = runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	decode := codecs.UniversalDeserializer().Decode

	for _, file := range crdFiles {
		yaml, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			log.Fatal().Err(err).Msgf("Error reading CRD file %s", file)
		}

		crd := &apiv1.CustomResourceDefinition{}
		_, _, err = decode(yaml, nil, crd)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error decoding CRD file %s", file)
		}

		crd.Labels[constants.ReconcileLabel] = strconv.FormatBool(enableReconciler)

		crdExisting, err := crdClient.CustomResourceDefinitions().Get(context.Background(), crd.Name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			log.Fatal().Err(err).Msgf("error getting CRD %s", crd.Name)
		}

		if apierrors.IsNotFound(err) {
			log.Info().Msgf("crds %s not found, creating CRD", crd.Name)
			if err := util.CreateApplyAnnotation(crd, unstructured.UnstructuredJSONScheme); err != nil {
				log.Fatal().Err(err).Msgf("Error applying annotation to CRD %s", crd.Name)
			}
			if _, err = crdClient.CustomResourceDefinitions().Create(context.Background(), crd, metav1.CreateOptions{}); err != nil {
				log.Fatal().Err(err).Msgf("Error creating crd : %s", crd.Name)
			}
			log.Info().Msgf("Successfully created crd: %s", crd.Name)
		} else {
			log.Info().Msgf("Patching conversion webhook configuration for crd: %s, setting to \"None\"", crd.Name)

			crdExisting.Labels[constants.ReconcileLabel] = strconv.FormatBool(enableReconciler)
			crdExisting.Spec = crd.Spec
			crdExisting.Spec.Conversion = &apiv1.CustomResourceConversion{
				Strategy: apiv1.NoneConverter,
			}
			if _, err = crdClient.CustomResourceDefinitions().Update(context.Background(), crdExisting, metav1.UpdateOptions{}); err != nil {
				log.Fatal().Err(err).Msgf("Error updating conversion webhook configuration for crd : %s", crd.Name)
			}
			log.Info().Msgf("successfully set conversion webhook configuration for crd : %s to \"None\"", crd.Name)
		}
	}
}

func (b *bootstrap) createDefaultMeshConfig() error {
	// find presets config map to build the default MeshConfig from that
	presetsConfigMap, err := b.kubeClient.CoreV1().ConfigMaps(b.namespace).Get(context.TODO(), presetMeshConfigName, metav1.GetOptions{})

	// If the presets MeshConfig could not be loaded return the error
	if err != nil {
		return err
	}

	// Create a default meshConfig
	defaultMeshConfig, err := buildDefaultMeshConfig(presetsConfigMap)
	if err != nil {
		return err
	}
	if _, err := b.configClient.ConfigV1alpha2().MeshConfigs(b.namespace).Create(context.TODO(), defaultMeshConfig, metav1.CreateOptions{}); err == nil {
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
	config, err := b.configClient.ConfigV1alpha2().MeshConfigs(b.namespace).Get(context.TODO(), meshConfigName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// create a default mesh config since it was not found
		return b.createDefaultMeshConfig()
	}
	if err != nil {
		return err
	}

	if _, exists := config.Annotations[corev1.LastAppliedConfigAnnotation]; !exists {
		// Mesh was found, but may not have the last applied annotation.
		if err := util.CreateApplyAnnotation(config, unstructured.UnstructuredJSONScheme); err != nil {
			return err
		}
		if _, err := b.configClient.ConfigV1alpha2().MeshConfigs(b.namespace).Update(context.TODO(), config, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// initiatilizeKubernetesEventsRecorder initializes the generic Kubernetes event recorder and associates it with
// the osm-bootstrap pod resource. The events recorder allows the osm-bootstap to publish Kubernets events to
// report fatal errors with initializing this application. These events will show up in the output of `kubectl get events`
func (b *bootstrap) initiatilizeKubernetesEventsRecorder() error {
	bootstrapPod, err := b.getBootstrapPod()
	if err != nil {
		return fmt.Errorf("Error fetching osm-bootstrap pod: %w", err)
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
	if osmNamespace == "" {
		return errors.New("Please specify the OSM namespace using --osm-namespace")
	}

	return nil
}

func buildDefaultMeshConfig(presetMeshConfigMap *corev1.ConfigMap) (*configv1alpha2.MeshConfig, error) {
	presetMeshConfig := presetMeshConfigMap.Data[presetMeshConfigJSONKey]
	presetMeshConfigSpec := configv1alpha2.MeshConfigSpec{}
	err := json.Unmarshal([]byte(presetMeshConfig), &presetMeshConfigSpec)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error converting preset-mesh-config json string to meshConfig object")
	}

	config := &configv1alpha2.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MeshConfig",
			APIVersion: "config.openservicemesh.io/configv1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: meshConfigName,
		},
		Spec: presetMeshConfigSpec,
	}

	return config, util.CreateApplyAnnotation(config, unstructured.UnstructuredJSONScheme)
}

func (b *bootstrap) ensureMeshRootCertificate() error {
	meshRootCertificateList, err := b.configClient.ConfigV1alpha2().MeshRootCertificates(b.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(meshRootCertificateList.Items) != 0 {
		return nil
	}

	// create a MeshRootCertificate since none were found
	return b.createMeshRootCertificate()
}

func (b *bootstrap) createMeshRootCertificate() error {
	// TODO(5046): create the mrc per https://gist.github.com/keithmattix/9ac73abbcb5721b0ce58102ac3ebff29
	// find preset config map to build the MeshRootCertificate from
	presetMeshRootCertificate, err := b.kubeClient.CoreV1().ConfigMaps(b.namespace).Get(context.TODO(), presetMeshRootCertificateName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Create a MeshRootCertificate
	defaultMeshRootCertificate, err := buildMeshRootCertificate(presetMeshRootCertificate)
	if err != nil {
		return err
	}
	createdMRC, err := b.configClient.ConfigV1alpha2().MeshRootCertificates(b.namespace).Create(context.TODO(), defaultMeshRootCertificate, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		log.Info().Msgf("MeshRootCertificate already exists in %s. Skip creating.", b.namespace)
		return nil
	}
	if err != nil {
		return err
	}

	_, err = b.configClient.ConfigV1alpha2().MeshRootCertificates(b.namespace).UpdateStatus(context.Background(), createdMRC, metav1.UpdateOptions{})
	if apierrors.IsAlreadyExists(err) {
		log.Info().Msgf("MeshRootCertificate status already exists in %s. Skip creating.", b.namespace)
	}

	if err != nil {
		return err
	}

	log.Info().Msgf("Successfully created MeshRootCertificate %s in %s.", constants.DefaultMeshRootCertificateName, b.namespace)
	return nil
}

func buildMeshRootCertificate(presetMeshRootCertificateConfigMap *corev1.ConfigMap) (*configv1alpha2.MeshRootCertificate, error) {
	presetMeshRootCertificate := presetMeshRootCertificateConfigMap.Data[presetMeshRootCertificateJSONKey]
	presetMeshRootCertificateSpec := configv1alpha2.MeshRootCertificateSpec{}
	err := json.Unmarshal([]byte(presetMeshRootCertificate), &presetMeshRootCertificateSpec)
	if err != nil {
		return nil, fmt.Errorf("error converting preset-mesh-root-certificate json string to MeshRootCertificate object: %w", err)
	}

	mrc := &configv1alpha2.MeshRootCertificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MeshRootCertificate",
			APIVersion: "config.openservicemesh.io/configv1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.DefaultMeshRootCertificateName,
		},
		Spec: presetMeshRootCertificateSpec,
	}

	return mrc, util.CreateApplyAnnotation(mrc, unstructured.UnstructuredJSONScheme)
}
