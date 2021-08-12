// Package main implements the main entrypoint for osm-controller and utility routines to
// bootstrap the various internal components of osm-controller.
// osm-controller is the core control plane component in OSM responsible for programming sidecar proxies.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsClientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/config"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/debugger"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy/ads"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/health"
	"github.com/openservicemesh/osm/pkg/httpserver"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/providers/kube"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/validator"
	"github.com/openservicemesh/osm/pkg/version"
)

const (
	xdsServerCertificateCommonName = "ads"
	validatorWebhookSvc            = "osm-validator"
)

var (
	verbosity                  string
	meshName                   string // An ID that uniquely identifies an OSM instance
	osmNamespace               string
	osmServiceAccount          string
	validatorWebhookConfigName string
	caBundleSecretName         string
	osmMeshConfigName          string

	certProviderKind string

	tresorOptions      providers.TresorOptions
	vaultOptions       providers.VaultOptions
	certManagerOptions providers.CertManagerOptions

	scheme = runtime.NewScheme()
)

var (
	flags = pflag.NewFlagSet(`osm-controller`, pflag.ExitOnError)
	log   = logger.New("osm-controller/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", constants.DefaultOSMLogLevel, "Set boot log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "OSM mesh name")
	flags.StringVar(&osmNamespace, "osm-namespace", "", "OSM controller's namespace")
	flags.StringVar(&osmServiceAccount, "osm-service-account", "", "OSM controller's service account")
	flags.StringVar(&validatorWebhookConfigName, "validator-webhook-config", "", "Name of the ValidatingWebhookConfiguration for the resource validator webhook")
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
	log.Info().Msgf("Starting osm-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Str(errcode.Kind, errcode.ErrInvalidCLIArgument.String()).Msg("Error parsing cmd line arguments")
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
	policyClient := policyClientset.NewForConfigOrDie(kubeConfig)

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

	// Start the default metrics store
	startMetricsStore()

	// This component will be watching the OSM MeshConfig and will make it available
	// to the rest of the components.
	cfg := configurator.NewConfigurator(configClientset.NewForConfigOrDie(kubeConfig), stop, osmNamespace, osmMeshConfigName)

	// Start Global log level handler, reads from configurator (meshconfig)
	StartGlobalLogLevelHandler(cfg, stop)

	k8sClient, err := k8s.NewKubernetesController(kubeClient, policyClient, meshName, stop)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Kubernetes Controller")
	}

	meshSpec, err := smi.NewMeshSpecClient(kubeConfig, kubeClient, osmNamespace, k8sClient, stop)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating MeshSpec")
	}

	certManager, certDebugger, _, err := providers.NewCertificateProvider(kubeClient, kubeConfig, cfg, providers.Kind(certProviderKind), osmNamespace,
		caBundleSecretName, tresorOptions, vaultOptions, certManagerOptions)

	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCertificateManager,
			"Error fetching certificate manager of kind %s", certProviderKind)
	}

	if cfg.GetFeatureFlags().EnableMulticlusterMode {
		log.Info().Msgf("Bootstrapping OSM multicluster gateway")
		if err := bootstrapOSMMulticlusterGateway(kubeClient, certManager, osmNamespace); err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError,
				"Error bootstraping OSM multicluster gateway")
		}
	}

	var configClient config.Controller

	if cfg.GetFeatureFlags().EnableMulticlusterMode {
		if configClient, err = config.NewConfigController(kubeConfig, k8sClient, stop); err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Kubernetes config client")
		}
	}

	// A nil configClient is passed in if multi cluster mode is not enabled.
	kubeProvider := kube.NewClient(k8sClient, configClient, constants.KubeProviderName, cfg)

	endpointsProviders := []endpoint.Provider{kubeProvider}
	serviceProviders := []service.Provider{kubeProvider}

	ingressClient, err := ingress.NewIngressClient(kubeClient, k8sClient, stop, cfg, certManager)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Ingress monitor client")
	}

	policyController, err := policy.NewPolicyController(k8sClient, policyClient, stop)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating controller for policy.openservicemesh.io")
	}

	meshCatalog := catalog.NewMeshCatalog(
		k8sClient,
		meshSpec,
		certManager,
		ingressClient,
		policyController,
		stop,
		cfg,
		serviceProviders,
		endpointsProviders,
	)

	var proxyMapper registry.ProxyServiceMapper
	if cfg.GetFeatureFlags().EnableAsyncProxyServiceMapping {
		m := registry.NewAsyncKubeProxyServiceMapper(k8sClient)
		m.Run(stop)
		proxyMapper = m
	} else {
		proxyMapper = &registry.KubeProxyServiceMapper{KubeController: k8sClient}
	}
	proxyRegistry := registry.NewProxyRegistry(proxyMapper)
	proxyRegistry.ReleaseCertificateHandler(certManager)

	adsCert, err := certManager.IssueCertificate(xdsServerCertificateCommonName, constants.XDSCertificateValidityPeriod)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.CertificateIssuanceFailure, "Error issuing XDS certificate to ADS server")
	}

	// Create and start the ADS gRPC service
	xdsServer := ads.NewADSServer(meshCatalog, proxyRegistry, cfg.IsDebugServerEnabled(), osmNamespace, cfg, certManager, k8sClient)
	if err := xdsServer.Start(ctx, cancel, constants.ADSServerPort, adsCert); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error initializing ADS server")
	}

	clientset := extensionsClientset.NewForConfigOrDie(kubeConfig)

	webhookHandlerCert, err := certManager.IssueCertificate(
		certificate.CommonName(fmt.Sprintf("%s.%s.svc", validatorWebhookSvc, osmNamespace)),
		constants.XDSCertificateValidityPeriod)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.CertificateIssuanceFailure, "Error issuing certificate for the validating webhook")
	}

	if err := validator.NewValidatingWebhook(validatorWebhookConfigName, constants.ValidatorWebhookPort, webhookHandlerCert, kubeClient, stop); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error starting the validating webhook server")
	}

	// Initialize OSM's http service server
	httpServer := httpserver.NewHTTPServer(constants.OSMHTTPServerPort)
	// Health/Liveness probes
	funcProbes := []health.Probes{xdsServer, smi.HealthChecker{DiscoveryClient: clientset.Discovery()}}
	httpServer.AddHandlers(map[string]http.Handler{
		"/health/ready": health.ReadinessHandler(funcProbes, getHTTPHealthProbes()),
		"/health/alive": health.LivenessHandler(funcProbes, getHTTPHealthProbes()),
	})
	// Metrics
	httpServer.AddHandler("/metrics", metricsstore.DefaultMetricsStore.Handler())
	// Version
	httpServer.AddHandler("/version", version.GetVersionHandler())
	// Supported SMI Versions
	httpServer.AddHandler(constants.HTTPServerSmiVersionPath, smi.GetSmiClientVersionHTTPHandler())

	// Start HTTP server
	err = httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start OSM metrics/probes HTTP server")
	}

	// Create DebugServer and start its config event listener.
	// Listener takes care to start and stop the debug server as appropriate
	debugConfig := debugger.NewDebugConfig(certDebugger, xdsServer, meshCatalog, proxyRegistry, kubeConfig, kubeClient, cfg, k8sClient)
	debugConfig.StartDebugServerConfigListener()

	k8s.PatchSecretHandler(kubeClient)

	<-stop
	log.Info().Msgf("Stopping osm-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

// Start the metric store, register the metrics OSM will expose
func startMetricsStore() {
	metricsstore.DefaultMetricsStore.Start(
		metricsstore.DefaultMetricsStore.K8sAPIEventCounter,
		metricsstore.DefaultMetricsStore.ProxyConnectCount,
		metricsstore.DefaultMetricsStore.ProxyReconnectCount,
		metricsstore.DefaultMetricsStore.ProxyConfigUpdateTime,
		metricsstore.DefaultMetricsStore.ProxyBroadcastEventCount,
		metricsstore.DefaultMetricsStore.CertIssuedCount,
		metricsstore.DefaultMetricsStore.CertIssuedTime,
		metricsstore.DefaultMetricsStore.ErrCodeCounter,
	)
}

// getHTTPHealthProbes returns the HTTP health probes served by OSM controller
func getHTTPHealthProbes() []health.HTTPProbe {
	return []health.HTTPProbe{
		// Internal probe to validator's webhook port
		{
			URL:      joinURL(fmt.Sprintf("https://%s:%d", constants.LocalhostIPAddress, constants.ValidatorWebhookPort), validator.HealthAPIPath),
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
		// TODO: Need to push metric?
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingControllerPod)).
			Msgf("Error retrieving osm-controller pod %s", podName)
		return nil, err
	}

	return pod, nil
}
