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

	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/openservicemesh/osm/pkg/constants"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const installDesc = `
This command installs an osm control plane on the Kubernetes cluster.

An osm control plane is comprised of namespaced Kubernetes resources
that get installed into the osm-system namespace as well as cluster
wide Kubernetes resources.

The default Kubernetes namespace that gets created on install is called
osm-system. To create an install control plane components in a different
namespace, use the --namespace flag.

Example:
  $ osm install --namespace hello-world

Multiple control plane installations can exist within a cluster. Each
control plane is given a cluster-wide unqiue identifier called mesh name.
A mesh name can be passed in via the --mesh-name flag. By default, the
mesh-name name will be set to "osm." The mesh name must conform to same
guidlines as a valid Kubernetes label value. Must be 63 characters or
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
	defaultCertManager        = "tresor"
	defaultVaultProtocol      = "http"
	defaultMeshName           = "osm"
	defaultOsmImagePullPolicy = "IfNotPresent"
)

// chartTGZSource is a base64-encoded, gzipped tarball of the default Helm chart.
// Its value is initialized at build time.
var chartTGZSource string

type installCmd struct {
	out                           io.Writer
	certificateManager            string
	certmanagerIssuerGroup        string
	certmanagerIssuerKind         string
	certmanagerIssuerName         string
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
	enableDebugServer             bool
	enableEgress                  bool
	enablePermissiveTrafficPolicy bool
	clientSet                     kubernetes.Interface
	chartRequested                *chart.Chart

	// This is an experimental flag, which will eventually
	// become part of SMI Spec.
	enableBackpressureExperimental bool

	// Toggle to enable/disable Prometheus installation
	enablePrometheus bool

	// Toggle to enable/disable Grafana installation
	enableGrafana bool

	// Toggle this to enable/disable the automatic deployment of Jaeger
	deployJaeger bool
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
				return errors.Errorf("Error fetching kubeconfig")
			}

			clientset, err := kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster. Check kubeconfig, %s", err)
			}
			inst.clientSet = clientset
			return inst.run(config)
		},
	}

	f := cmd.Flags()
	f.StringVar(&inst.containerRegistry, "container-registry", "openservicemesh", "container registry that hosts control plane component images")
	f.StringVar(&inst.chartPath, "osm-chart-path", "", "path to osm chart to override default chart")
	f.StringVar(&inst.certificateManager, "certificate-manager", defaultCertManager, "certificate manager to use one of (tresor, vault, cert-manager)")
	f.StringVar(&inst.osmImageTag, "osm-image-tag", "v0.4.2", "osm image tag")
	f.StringVar(&inst.osmImagePullPolicy, "osm-image-pull-policy", defaultOsmImagePullPolicy, "osm image pull policy")
	f.StringVar(&inst.containerRegistrySecret, "container-registry-secret", "", "name of Kubernetes secret for container registry credentials to be created if it doesn't already exist")
	f.StringVar(&inst.vaultHost, "vault-host", "", "Hashicorp Vault host/service - where Vault is installed")
	f.StringVar(&inst.vaultProtocol, "vault-protocol", defaultVaultProtocol, "protocol to use to connect to Vault")
	f.StringVar(&inst.vaultToken, "vault-token", "", "token that should be used to connect to Vault")
	f.StringVar(&inst.vaultRole, "vault-role", "openservicemesh", "Vault role to be used by Open Service Mesh")
	f.StringVar(&inst.certmanagerIssuerName, "cert-manager-issuer-name", "osm-ca", "cert-manager issuer name")
	f.StringVar(&inst.certmanagerIssuerKind, "cert-manager-issuer-kind", "Issuer", "cert-manager issuer kind")
	f.StringVar(&inst.certmanagerIssuerGroup, "cert-manager-issuer-group", "cert-manager.io", "cert-manager issuer group")
	f.StringVar(&inst.serviceCertValidityDuration, "service-cert-validity-duration", "24h", "Service certificate validity duration, represented as a sequence of decimal numbers each with optional fraction and a unit suffix")
	f.StringVar(&inst.prometheusRetentionTime, "prometheus-retention-time", constants.PrometheusDefaultRetentionTime, "Duration for which data will be retained in prometheus")
	f.BoolVar(&inst.enableDebugServer, "enable-debug-server", false, "Enable the debug HTTP server")
	f.BoolVar(&inst.enablePermissiveTrafficPolicy, "enable-permissive-traffic-policy", false, "Enable permissive traffic policy mode")
	f.BoolVar(&inst.enableEgress, "enable-egress", false, "Enable egress in the mesh")
	f.BoolVar(&inst.enableBackpressureExperimental, "enable-backpressure-experimental", false, "Enable experimental backpressure feature")
	f.BoolVar(&inst.enablePrometheus, "enable-prometheus", true, "Enable Prometheus installation and deployment")
	f.BoolVar(&inst.enableGrafana, "enable-grafana", false, "Enable Grafana installation and deployment")
	f.StringVar(&inst.meshName, "mesh-name", defaultMeshName, "name for the new control plane instance")
	f.BoolVar(&inst.deployJaeger, "deploy-jaeger", true, "Deploy Jaeger in the namespace of the OSM controller")
	f.StringVar(&inst.envoyLogLevel, "envoy-log-level", "error", "Envoy log level is used to specify the level of logs collected from envoy and needs to be one of these (trace, debug, info, warning, warn, error, critical, off)")

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
	valuesConfig := []string{
		fmt.Sprintf("OpenServiceMesh.image.registry=%s", i.containerRegistry),
		fmt.Sprintf("OpenServiceMesh.image.tag=%s", i.osmImageTag),
		fmt.Sprintf("OpenServiceMesh.image.pullPolicy=%s", i.osmImagePullPolicy),
		fmt.Sprintf("OpenServiceMesh.certificateManager=%s", i.certificateManager),
		fmt.Sprintf("OpenServiceMesh.vault.host=%s", i.vaultHost),
		fmt.Sprintf("OpenServiceMesh.vault.protocol=%s", i.vaultProtocol),
		fmt.Sprintf("OpenServiceMesh.vault.token=%s", i.vaultToken),
		fmt.Sprintf("OpenServiceMesh.vault.role=%s", i.vaultRole),
		fmt.Sprintf("OpenServiceMesh.certmanager.issuerName=%s", i.certmanagerIssuerName),
		fmt.Sprintf("OpenServiceMesh.certmanager.issuerKind=%s", i.certmanagerIssuerKind),
		fmt.Sprintf("OpenServiceMesh.certmanager.issuerGroup=%s", i.certmanagerIssuerGroup),
		fmt.Sprintf("OpenServiceMesh.serviceCertValidityDuration=%s", i.serviceCertValidityDuration),
		fmt.Sprintf("OpenServiceMesh.prometheus.retention.time=%s", i.prometheusRetentionTime),
		fmt.Sprintf("OpenServiceMesh.enableDebugServer=%t", i.enableDebugServer),
		fmt.Sprintf("OpenServiceMesh.enablePermissiveTrafficPolicy=%t", i.enablePermissiveTrafficPolicy),
		fmt.Sprintf("OpenServiceMesh.enableBackpressureExperimental=%t", i.enableBackpressureExperimental),
		fmt.Sprintf("OpenServiceMesh.enablePrometheus=%t", i.enablePrometheus),
		fmt.Sprintf("OpenServiceMesh.enableGrafana=%t", i.enableGrafana),
		fmt.Sprintf("OpenServiceMesh.meshName=%s", i.meshName),
		fmt.Sprintf("OpenServiceMesh.enableEgress=%t", i.enableEgress),
		fmt.Sprintf("OpenServiceMesh.deployJaeger=%t", i.deployJaeger),
		fmt.Sprintf("OpenServiceMesh.envoyLogLevel=%s", strings.ToLower(i.envoyLogLevel)),
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
	list, err := deploymentsClient.List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	if len(list.Items) != 0 {
		return errMeshAlreadyExists(i.meshName)
	}

	// ensure no osm-controller is running in the same namespace
	deploymentsClient = i.clientSet.AppsV1().Deployments(settings.Namespace()) // Get deployments for specified namespace
	labelSelector = metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.OSMControllerName}}
	listOptions = metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	list, err = deploymentsClient.List(context.TODO(), listOptions)
	if len(list.Items) != 0 {
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
	return errors.Errorf("Namespace %s has an osm controller. Please specify a new namespace using --namespace", namespace)
}
