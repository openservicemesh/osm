package main

import (
	"context"
	"fmt"
	"io"
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
	defaultChartPath         = ""
	defaultMeshName          = "osm"
	defaultEnforceSingleMesh = false
)

// chartTGZSource is a base64-encoded, gzipped tarball of the default Helm chart.
// Its value is initialized at build time.
var chartTGZSource string

type installCmd struct {
	out            io.Writer
	chartPath      string
	meshName       string
	timeout        time.Duration
	clientSet      kubernetes.Interface
	chartRequested *chart.Chart
	setOptions     []string
	atomic         bool
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
	f.StringVar(&inst.chartPath, "osm-chart-path", defaultChartPath, "path to osm chart to override default chart")
	f.StringVar(&inst.meshName, "mesh-name", defaultMeshName, "name for the new control plane instance")
	f.BoolVar(&inst.enforceSingleMesh, "enforce-single-mesh", defaultEnforceSingleMesh, "Enforce only deploying one mesh in the cluster")
	f.DurationVar(&inst.timeout, "timeout", 5*time.Minute, "Time to wait for installation and resources in a ready state, zero means no timeout")
	f.StringArrayVar(&inst.setOptions, "set", nil, "Set arbitrary chart values (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	f.BoolVar(&inst.atomic, "atomic", false, "Automatically clean up resources if installation fails")

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
	installClient.Wait = true
	installClient.Atomic = i.atomic
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

	if err := parseVal(i.setOptions, finalValues); err != nil {
		return nil, errors.Wrap(err, "invalid format for --set")
	}

	valuesConfig := []string{
		fmt.Sprintf("OpenServiceMesh.meshName=%s", i.meshName),
		fmt.Sprintf("OpenServiceMesh.enforceSingleMesh=%t", i.enforceSingleMesh),
	}

	if err := parseVal(valuesConfig, finalValues); err != nil {
		return nil, err
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
		return annotateErrorMessageWithOsmNamespace("Error ensuring no osm-controller running in OSM namespace [%s]: %s", settings.Namespace(), err)
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
			return errAlreadyExists
		}
	}

	s := map[string]interface{}{}
	if err := parseVal(i.setOptions, s); err != nil {
		return errors.Wrap(err, "invalid format for --set")
	}

	if setOptions, ok := s["OpenServiceMesh"].(map[string]interface{}); ok {
		// if deployPrometheus is true, make sure enablePrometheusScraping is not disabled
		if setOptions["deployPrometheus"] == true {
			if setOptions["enablePrometheusScraping"] == false {
				_, _ = fmt.Fprintf(i.out, "Prometheus scraping is disabled. To enable it, set prometheus_scraping in %s/%s to true.\n", settings.Namespace(), constants.OSMMeshConfig)
			}
		}

		// if certificateManager is vault, ensure all relevant information (vault-host, vault-token) is available
		if setOptions["certificateManager"] == "vault" {
			var missingFields []string
			vaultOptions, ok := setOptions["vault"].(map[string]interface{})
			if !ok {
				missingFields = append(missingFields, "OpenServiceMesh.vault.host", "OpenServiceMesh.vault.token")
			} else {
				if vaultOptions["host"] == nil || vaultOptions["host"] == "" {
					missingFields = append(missingFields, "OpenServiceMesh.vault.host")
				}
				if vaultOptions["token"] == nil || vaultOptions["token"] == "" {
					missingFields = append(missingFields, "OpenServiceMesh.vault.token")
				}
			}

			if len(missingFields) != 0 {
				return errors.Errorf("Missing arguments for certificate-manager vault: %v", missingFields)
			}
		}
	}

	return nil
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
	return annotateErrorMessageWithOsmNamespace("Namespace [%s] already has an osm controller.", namespace)
}

// parses Helm strvals line and merges into a map
func parseVal(vals []string, parsedVals map[string]interface{}) error {
	for _, v := range vals {
		if err := strvals.ParseInto(v, parsedVals); err != nil {
			return err
		}
	}
	return nil
}
