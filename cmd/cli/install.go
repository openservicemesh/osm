package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/openservicemesh/osm/pkg/constants"
)

const installDesc = `
This command installs an osm control plane on the Kubernetes cluster.

An osm control plane is comprised of namespaced Kubernetes resources
that get installed into the osm-system namespace as well as cluster
wide Kubernetes resources.

The default Kubernetes namespace that gets created on install is called
osm-system. To create an install control plane components in a different
namespace, use the global --osm-namespace flag.

Example:
  $ osm install --osm-namespace hello-world

Multiple control plane installations can exist within a cluster. Each
control plane is given a cluster-wide unqiue identifier called mesh name.
A mesh name can be passed in via the --mesh-name flag. By default, the
mesh-name name will be set to "osm." The mesh name must conform to same
guidelines as a valid Kubernetes label value. Must be 63 characters or
less and must be empty or begin and end with an alphanumeric character
([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and
alphanumerics between.

Example:
  $ osm install --mesh-name "hello-osm"

The mesh name is used in various ways like for naming Kubernetes resources as
well as for adding a Kubernetes Namespace to the list of Namespaces a control
plane should watch for sidecar injection of Envoy proxies.
`
const (
	defaultCertificateManager             = "tresor"
	defaultCertManagerIssuerGroup         = "cert-manager.io"
	defaultCertManagerIssuerKind          = "Issuer"
	defaultCertManagerIssuerName          = "osm-ca"
	defaultChartPath                      = ""
	defaultContainerRegistry              = "openservicemesh"
	defaultContainerRegistrySecret        = ""
	defaultMeshName                       = "osm"
	defaultOsmImagePullPolicy             = "IfNotPresent"
	defaultOsmImageTag                    = "v0.6.1"
	defaultPrometheusRetentionTime        = constants.PrometheusDefaultRetentionTime
	defaultVaultHost                      = ""
	defaultVaultProtocol                  = "http"
	defaultVaultToken                     = ""
	defaultVaultRole                      = "openservicemesh"
	defaultEnvoyLogLevel                  = "error"
	defaultServiceCertValidityDuration    = "24h"
	defaultEnableDebugServer              = false
	defaultEnableEgress                   = false
	defaultEnablePermissiveTrafficPolicy  = false
	defaultEnableBackpressureExperimental = false
	defaultEnableRoutesV2Experimental     = false
	defaultDeployPrometheus               = false
	defaultEnablePrometheusScraping       = true
	defaultDeployGrafana                  = false
	defaultEnableFluentbit                = false
	defaultDeployJaeger                   = false
	defaultEnforceSingleMesh              = false
)

// chartTGZSource is a base64-encoded, gzipped tarball of the default Helm chart.
// Its value is initialized at build time.
var chartTGZSource string

type installCmd struct {
	out                           io.Writer
	certificateManager            string
	certManagerIssuerGroup        string
	certManagerIssuerKind         string
	certManagerIssuerName         string
	chartPath                     string
	containerRegistry             string
	containerRegistrySecret       string
	meshName                      string
	osmImagePullPolicy            string
	osmImageTag                   string
	prometheusRetentionTime       string
	vaultHost                     string
	vaultProtocol                 string
	vaultToken                    string
	vaultRole                     string
	envoyLogLevel                 string
	serviceCertValidityDuration   string
	timeout                       time.Duration
	enableDebugServer             bool
	enableEgress                  bool
	enablePermissiveTrafficPolicy bool
	clientSet                     kubernetes.Interface
	chartRequested                *chart.Chart
	setOptions                    []string

	// This is an experimental flag, which will eventually
	// become part of SMI Spec.
	enableBackpressureExperimental bool

	// This is an experimental flag, which results in using
	// 	the experimental routes v2 feature
	enableRoutesV2Experimental bool

	// Toggle to enable/disable Prometheus installation
	deployPrometheus bool

	// Toggle to enable/disable Prometheus scraping
	enablePrometheusScraping bool

	// Toggle to enable/disable Grafana installation
	deployGrafana bool

	// Toggle to enable/disable FluentBit sidecar
	enableFluentbit bool

	// Toggle this to enable/disable the automatic deployment of Jaeger
	deployJaeger bool

	// Toggle this to enforce only one mesh in this cluster
	enforceSingleMesh bool
}

func newInstallCmd(config *helm.Configuration, out io.Writer) *cobra.Command {
	inst := &installCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "install osm control plane",
		Long:  installDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}

			clientset, err := kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			inst.clientSet = clientset
			return inst.run(config)
		},
	}

	f := cmd.Flags()
	f.StringVar(&inst.containerRegistry, "container-registry", defaultContainerRegistry, "container registry that hosts control plane component images")
	f.StringVar(&inst.chartPath, "osm-chart-path", defaultChartPath, "path to osm chart to override default chart")
	f.StringVar(&inst.certificateManager, "certificate-manager", defaultCertificateManager, "certificate manager to use one of (tresor, vault, cert-manager)")
	f.StringVar(&inst.osmImageTag, "osm-image-tag", defaultOsmImageTag, "osm image tag")
	f.StringVar(&inst.osmImagePullPolicy, "osm-image-pull-policy", defaultOsmImagePullPolicy, "osm image pull policy")
	f.StringVar(&inst.containerRegistrySecret, "container-registry-secret", defaultContainerRegistrySecret, "name of Kubernetes secret for container registry credentials to be created if it doesn't already exist")
	f.StringVar(&inst.vaultHost, "vault-host", defaultVaultHost, "Hashicorp Vault host/service - where Vault is installed")
	f.StringVar(&inst.vaultProtocol, "vault-protocol", defaultVaultProtocol, "protocol to use to connect to Vault")
	f.StringVar(&inst.vaultToken, "vault-token", defaultVaultToken, "token that should be used to connect to Vault")
	f.StringVar(&inst.vaultRole, "vault-role", defaultVaultRole, "Vault role to be used by Open Service Mesh")
	f.StringVar(&inst.certManagerIssuerName, "cert-manager-issuer-name", defaultCertManagerIssuerName, "cert-manager issuer name")
	f.StringVar(&inst.certManagerIssuerKind, "cert-manager-issuer-kind", defaultCertManagerIssuerKind, "cert-manager issuer kind")
	f.StringVar(&inst.certManagerIssuerGroup, "cert-manager-issuer-group", defaultCertManagerIssuerGroup, "cert-manager issuer group")
	f.StringVar(&inst.serviceCertValidityDuration, "service-cert-validity-duration", defaultServiceCertValidityDuration, "Service certificate validity duration, represented as a sequence of decimal numbers each with optional fraction and a unit suffix")
	f.StringVar(&inst.prometheusRetentionTime, "prometheus-retention-time", defaultPrometheusRetentionTime, "Duration for which data will be retained in prometheus")
	f.BoolVar(&inst.enableDebugServer, "enable-debug-server", defaultEnableDebugServer, "Enable the debug HTTP server")
	f.BoolVar(&inst.enablePermissiveTrafficPolicy, "enable-permissive-traffic-policy", defaultEnablePermissiveTrafficPolicy, "Enable permissive traffic policy mode")
	f.BoolVar(&inst.enableEgress, "enable-egress", defaultEnableEgress, "Enable egress in the mesh")
	f.BoolVar(&inst.enableBackpressureExperimental, "enable-backpressure-experimental", defaultEnableBackpressureExperimental, "Enable experimental backpressure feature")
	f.BoolVar(&inst.enableRoutesV2Experimental, "enable-routes-v2-experimental", defaultEnableRoutesV2Experimental, "Enable experimental routes v2 feature")
	f.BoolVar(&inst.deployPrometheus, "deploy-prometheus", defaultDeployPrometheus, "Install and deploy Prometheus")
	f.BoolVar(&inst.enablePrometheusScraping, "enable-prometheus-scraping", defaultEnablePrometheusScraping, "Enable Prometheus metrics scraping on sidecar proxies")
	f.BoolVar(&inst.deployGrafana, "deploy-grafana", defaultDeployGrafana, "Install and deploy Grafana")
	f.BoolVar(&inst.enableFluentbit, "enable-fluentbit", defaultEnableFluentbit, "Enable Fluentbit sidecar deployment")
	f.StringVar(&inst.meshName, "mesh-name", defaultMeshName, "name for the new control plane instance")
	f.BoolVar(&inst.deployJaeger, "deploy-jaeger", defaultDeployJaeger, "Deploy Jaeger in the namespace of the OSM controller")
	f.StringVar(&inst.envoyLogLevel, "envoy-log-level", defaultEnvoyLogLevel, "Envoy log level is used to specify the level of logs collected from envoy and needs to be one of these (trace, debug, info, warning, warn, error, critical, off)")
	f.BoolVar(&inst.enforceSingleMesh, "enforce-single-mesh", defaultEnforceSingleMesh, "Enforce only deploying one mesh in the cluster")
	f.DurationVar(&inst.timeout, "timeout", 5*time.Minute, "Time to wait for installation and resources in a ready state, zero means no timeout")
	f.StringArrayVar(&inst.setOptions, "set", nil, "Set arbitrary chart values values not settable by another flag (can specify multiple or separate values with commas: key1=val1,key2=val2)")

	return cmd
}

func (i *installCmd) run(config *helm.Configuration) error {
	if err := i.validateOptions(); err != nil {
		return err
	}

	// values represents the overrides for the OSM chart's values.yaml file
	values, err := i.resolveValues()
	if err != nil {
		return err
	}

	installClient := helm.NewInstall(config)
	installClient.ReleaseName = i.meshName
	installClient.Namespace = settings.Namespace()
	installClient.CreateNamespace = true
	installClient.Atomic = true
	installClient.Timeout = i.timeout
	if _, err = installClient.Run(i.chartRequested, values); err != nil {
		return err
	}

	fmt.Fprintf(i.out, "OSM installed successfully in namespace [%s] with mesh name [%s]\n", settings.Namespace(), i.meshName)
	return nil
}
func (i *installCmd) loadOSMChart() error {
	var err error
	if i.chartPath != "" {
		i.chartRequested, err = loader.Load(i.chartPath)
	} else {
		i.chartRequested, err = cli.LoadChart(chartTGZSource)
	}

	if err != nil {
		return fmt.Errorf("Error loading chart for installation: %s", err)
	}

	return nil
}

func (i *installCmd) resolveValues() (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	for _, val := range i.setOptions {
		// parses Helm strvals line and merges into a map for the final overrides for values.yaml
		if err := strvals.ParseInto(val, finalValues); err != nil {
			return nil, errors.Wrap(err, "invalid format for --set")
		}
	}

	valuesConfig := []string{
		fmt.Sprintf("OpenServiceMesh.image.registry=%s", i.containerRegistry),
		fmt.Sprintf("OpenServiceMesh.image.tag=%s", i.osmImageTag),
		fmt.Sprintf("OpenServiceMesh.image.pullPolicy=%s", i.osmImagePullPolicy),
		fmt.Sprintf("OpenServiceMesh.certificateManager=%s", i.certificateManager),
		fmt.Sprintf("OpenServiceMesh.vault.host=%s", i.vaultHost),
		fmt.Sprintf("OpenServiceMesh.vault.protocol=%s", i.vaultProtocol),
		fmt.Sprintf("OpenServiceMesh.vault.token=%s", i.vaultToken),
		fmt.Sprintf("OpenServiceMesh.vault.role=%s", i.vaultRole),
		fmt.Sprintf("OpenServiceMesh.certmanager.issuerName=%s", i.certManagerIssuerName),
		fmt.Sprintf("OpenServiceMesh.certmanager.issuerKind=%s", i.certManagerIssuerKind),
		fmt.Sprintf("OpenServiceMesh.certmanager.issuerGroup=%s", i.certManagerIssuerGroup),
		fmt.Sprintf("OpenServiceMesh.serviceCertValidityDuration=%s", i.serviceCertValidityDuration),
		fmt.Sprintf("OpenServiceMesh.prometheus.retention.time=%s", i.prometheusRetentionTime),
		fmt.Sprintf("OpenServiceMesh.enableDebugServer=%t", i.enableDebugServer),
		fmt.Sprintf("OpenServiceMesh.enablePermissiveTrafficPolicy=%t", i.enablePermissiveTrafficPolicy),
		fmt.Sprintf("OpenServiceMesh.enableBackpressureExperimental=%t", i.enableBackpressureExperimental),
		fmt.Sprintf("OpenServiceMesh.enableRoutesV2Experimental=%t", i.enableRoutesV2Experimental),
		fmt.Sprintf("OpenServiceMesh.deployPrometheus=%t", i.deployPrometheus),
		fmt.Sprintf("OpenServiceMesh.enablePrometheusScraping=%t", i.enablePrometheusScraping),
		fmt.Sprintf("OpenServiceMesh.deployGrafana=%t", i.deployGrafana),
		fmt.Sprintf("OpenServiceMesh.enableFluentbit=%t", i.enableFluentbit),
		fmt.Sprintf("OpenServiceMesh.meshName=%s", i.meshName),
		fmt.Sprintf("OpenServiceMesh.enableEgress=%t", i.enableEgress),
		fmt.Sprintf("OpenServiceMesh.deployJaeger=%t", i.deployJaeger),
		fmt.Sprintf("OpenServiceMesh.envoyLogLevel=%s", strings.ToLower(i.envoyLogLevel)),
		fmt.Sprintf("OpenServiceMesh.enforceSingleMesh=%t", i.enforceSingleMesh),
	}

	if i.containerRegistrySecret != "" {
		valuesConfig = append(valuesConfig, fmt.Sprintf("OpenServiceMesh.imagePullSecrets[0].name=%s", i.containerRegistrySecret))
	}

	for _, val := range valuesConfig {
		// parses Helm strvals line and merges into a map for the final overrides for values.yaml
		if err := strvals.ParseInto(val, finalValues); err != nil {
			return nil, err
		}
	}

	return finalValues, nil
}

func (i *installCmd) validateOptions() error {
	if err := i.loadOSMChart(); err != nil {
		return err
	}

	if err := isValidMeshName(i.meshName); err != nil {
		return err
	}

	// if certificateManager is vault, ensure all relevant information (vault-host, vault-token) is available
	if strings.EqualFold(i.certificateManager, "vault") {
		var missingFields []string
		if i.vaultHost == "" {
			missingFields = append(missingFields, "vault-host")
		}
		if i.vaultToken == "" {
			missingFields = append(missingFields, "vault-token")
		}
		if len(missingFields) != 0 {
			return errors.Errorf("Missing arguments for certificate-manager vault: %v", missingFields)
		}
	}

	// ensure no control plane exists in cluster with the same meshName
	deploymentsClient := i.clientSet.AppsV1().Deployments("") // Get deployments from all namespaces
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"meshName": i.meshName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	osmControllerDeployments, err := deploymentsClient.List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	if len(osmControllerDeployments.Items) != 0 {
		return errMeshAlreadyExists(i.meshName)
	}

	// ensure no osm-controller is running in the same namespace
	deploymentsClient = i.clientSet.AppsV1().Deployments(settings.Namespace()) // Get deployments for specified namespace
	labelSelector = metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.OSMControllerName}}
	listOptions = metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	osmControllerDeployments, err = deploymentsClient.List(context.TODO(), listOptions)
	if osmControllerDeployments != nil && len(osmControllerDeployments.Items) > 0 {
		return errNamespaceAlreadyHasController(settings.Namespace())
	} else if err != nil {
		return fmt.Errorf("Error ensuring no osm-controller running in namespace %s:%s", settings.Namespace(), err)
	}

	// validate the envoy log level type
	if err := isValidEnvoyLogLevel(i.envoyLogLevel); err != nil {
		return err
	}

	// validate certificate validity duration
	if _, err := time.ParseDuration(i.serviceCertValidityDuration); err != nil {
		return err
	}

	osmControllerDeployments, err = getControllerDeployments(i.clientSet)
	if err != nil {
		return err
	}

	// Check if single mesh cluster is already specified
	for _, deployment := range osmControllerDeployments.Items {
		singleMeshEnforced := deployment.ObjectMeta.Labels["enforceSingleMesh"] == "true"
		name := deployment.ObjectMeta.Labels["meshName"]
		if singleMeshEnforced {
			return errors.Errorf("Cannot install mesh [%s]. Existing mesh [%s] enforces single mesh cluster.", i.meshName, name)
		}
	}

	// Enforce single mesh cluster if needed
	if i.enforceSingleMesh {
		if len(osmControllerDeployments.Items) != 0 {
			return errors.Errorf("Meshes already exist in cluster. Cannot enforce single mesh cluster.")
		}
	}

	if i.deployPrometheus {
		if !i.enablePrometheusScraping {
			_, _ = fmt.Fprintf(i.out, "Prometheus scraping is disabled. To enable it, set prometheus_scraping in %s/%s to true.\n", settings.Namespace(), constants.OSMConfigMap)
		}
	}

	return nil
}

func isValidEnvoyLogLevel(envoyLogLevel string) error {
	// allowedLogLevels referenced from : https://github.com/envoyproxy/envoy/blob/release/v1.15/test/server/options_impl_test.cc#L373
	allowedLogLevels := []string{"trace", "debug", "info", "warning", "warn", "error", "critical", "off"}
	for _, logLevel := range allowedLogLevels {
		if strings.EqualFold(envoyLogLevel, logLevel) {
			return nil
		}
	}
	return errors.Errorf("Invalid envoy log level.\n A valid envoy log level must be one from the following : %v", allowedLogLevels)
}

func isValidMeshName(meshName string) error {
	meshNameErrs := validation.IsValidLabelValue(meshName)
	if len(meshNameErrs) != 0 {
		return errors.Errorf("Invalid mesh-name.\nValid mesh-name:\n- must be no longer than 63 characters\n- must consist of alphanumeric characters, '-', '_' or '.'\n- must start and end with an alphanumeric character\nregex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?'")
	}
	return nil
}

func errMeshAlreadyExists(name string) error {
	return errors.Errorf("Mesh %s already exists in cluster. Please specify a new mesh name using --mesh-name", name)
}

func errNamespaceAlreadyHasController(namespace string) error {
	return errors.Errorf("Namespace %s has an osm controller. Please specify a new namespace using --osm-namespace", namespace)
}
