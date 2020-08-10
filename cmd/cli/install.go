package main

import (
	"context"
	"fmt"
	"io"

	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/openservicemesh/osm/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
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
	defaultCertManager   = "tresor"
	defaultVaultProtocol = "http"
	defaultMeshName      = "osm"

	defaultCertValidityMinutes = int(1440) // 24 hours
)

// chartTGZSource is a base64-encoded, gzipped tarball of the default Helm chart.
// Its value is initialized at build time.
var chartTGZSource string

type installCmd struct {
	out       io.Writer
	clientSet kubernetes.Interface
	// action config is used to resolve the helm chart values
	cfg actionConfig
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
				return errors.Errorf("Could not access Kubernetes cluster. Check kubeconfig")
			}
			inst.clientSet = clientset
			return inst.run(config)
		},
	}

	f := cmd.Flags()
	f.StringVar(&inst.cfg.containerRegistry, "container-registry", "openservicemesh", "container registry that hosts control plane component images")
	f.StringVar(&inst.cfg.osmImageTag, "osm-image-tag", "v0.2.0", "osm image tag")
	f.StringVar(&inst.cfg.containerRegistrySecret, "container-registry-secret", "acr-creds", "name of Kubernetes secret for container registry credentials to be created if it doesn't already exist")
	f.StringVar(&inst.cfg.chartPath, "osm-chart-path", "", "path to osm chart to override default chart")
	f.StringVar(&inst.cfg.certManager, "certificate-manager", defaultCertManager, "certificate manager to use (tresor or vault)")
	f.StringVar(&inst.cfg.vaultHost, "vault-host", "", "Hashicorp Vault host/service - where Vault is installed")
	f.StringVar(&inst.cfg.vaultProtocol, "vault-protocol", defaultVaultProtocol, "protocol to use to connect to Vault")
	f.StringVar(&inst.cfg.vaultToken, "vault-token", "", "token that should be used to connect to Vault")
	f.StringVar(&inst.cfg.vaultRole, "vault-role", "openservicemesh", "Vault role to be used by Open Service Mesh")
	f.IntVar(&inst.cfg.serviceCertValidityMinutes, "service-cert-validity-minutes", defaultCertValidityMinutes, "Certificate TTL in minutes")
	f.StringVar(&inst.cfg.prometheusRetentionTime, "prometheus-retention-time", constants.PrometheusDefaultRetentionTime, "Duration for which data will be retained in prometheus")
	f.BoolVar(&inst.cfg.enableDebugServer, "enable-debug-server", false, "Enable the debug HTTP server")
	f.BoolVar(&inst.cfg.enablePermissiveTrafficPolicy, "enable-permissive-traffic-policy", false, "Enable permissive traffic policy mode")
	f.BoolVar(&inst.cfg.enableEgress, "enable-egress", false, "Enable egress in the mesh")
	f.StringSliceVar(&inst.cfg.meshCIDRRanges, "mesh-cidr", []string{}, "mesh CIDR range, accepts multiple CIDRs, required if enable-egress option is true")
	f.BoolVar(&inst.cfg.enableBackpressureExperimental, "enable-backpressure-experimental", false, "Enable experimental backpressure feature")
	f.BoolVar(&inst.cfg.enableMetricsStack, "enable-metrics-stack", true, "Enable metrics (Prometheus and Grafana) deployment")
	f.StringVar(&inst.cfg.meshName, "mesh-name", defaultMeshName, "name for the new control plane instance")
	f.BoolVar(&inst.cfg.deployZipkin, "deploy-zipkin", true, "Deploy Zipkin in the namespace of the OSM controller")

	return cmd
}

func (i *installCmd) run(config *helm.Configuration) error {
	var chartRequested *chart.Chart
	var err error
	if i.cfg.chartPath != "" {
		chartRequested, err = loader.Load(i.cfg.chartPath)
	} else {
		chartRequested, err = cli.LoadChart(chartTGZSource)
	}
	if err != nil {
		return err
	}

	err = i.cfg.validate()
	if err != nil {
		return err
	}

	deploymentsClient := i.clientSet.AppsV1().Deployments("") // Get deployments from all namespaces

	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"meshName": i.cfg.meshName}}

	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	list, err := deploymentsClient.List(context.TODO(), listOptions)
	if err != nil {
		return err
	}

	if len(list.Items) != 0 {
		return errMeshAlreadyExists(i.cfg.meshName)
	}

	deploymentsClient = i.clientSet.AppsV1().Deployments(settings.Namespace()) // Get deployments for specified namespace
	labelSelector = metav1.LabelSelector{MatchLabels: map[string]string{"app": "osm-controller"}}

	listOptions = metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	list, err = deploymentsClient.List(context.TODO(), listOptions)
	if len(list.Items) != 0 {
		return errNamespaceAlreadyHasController(settings.Namespace())
	}

	installClient := helm.NewInstall(config)
	installClient.ReleaseName = i.cfg.meshName
	installClient.Namespace = settings.Namespace()
	installClient.CreateNamespace = true

	values, err := i.cfg.resolveValues()
	if err != nil {
		return err
	}

	if _, err = installClient.Run(chartRequested, values); err != nil {
		return err
	}

	fmt.Fprintf(i.out, "OSM installed successfully in namespace [%s] with mesh name [%s]\n", settings.Namespace(), i.cfg.meshName)
	return nil
}

func errMeshAlreadyExists(name string) error {
	return errors.Errorf("Mesh %s already exists in cluster. Please specify a new mesh name using --mesh-name", name)
}

func errNamespaceAlreadyHasController(namespace string) error {
	return errors.Errorf("Namespace %s has an osm controller. Please specify a new namespace using --namespace", namespace)
}
