package main

import (
	"bytes"
	"context"

	_ "embed" // required to embed resources
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
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
	defaultEnforceSingleMesh = true
)

// chartTGZSource is the `helm package`d representation of the default Helm chart.
// Its value is embedded at build time.
//go:embed chart.tgz
var chartTGZSource []byte

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
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}

			clientset, err := kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
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
	if err := i.loadOSMChart(); err != nil {
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

	debug("Beginning OSM installation")
	if _, err = installClient.Run(i.chartRequested, values); err != nil {
		if !settings.Verbose() {
			return err
		}

		pods, _ := i.clientSet.CoreV1().Pods(settings.Namespace()).List(context.Background(), metav1.ListOptions{})

		for _, pod := range pods.Items {
			fmt.Fprintf(i.out, "Status for pod %s in namespace %s:\n %v\n\n", pod.Name, pod.Namespace, pod.Status)
		}
		return err
	}

	fmt.Fprintf(i.out, "OSM installed successfully in namespace [%s] with mesh name [%s]\n", settings.Namespace(), i.meshName)
	return nil
}

func (i *installCmd) loadOSMChart() error {
	debug("Loading OSM helm chart")
	var err error
	if i.chartPath != "" {
		i.chartRequested, err = loader.Load(i.chartPath)
	} else {
		i.chartRequested, err = loader.LoadArchive(bytes.NewReader(chartTGZSource))
	}

	if err != nil {
		return fmt.Errorf("error loading chart for installation: %w", err)
	}

	return nil
}

func (i *installCmd) resolveValues() (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}

	if err := parseVal(i.setOptions, finalValues); err != nil {
		return nil, fmt.Errorf("invalid format for --set: %w", err)
	}

	valuesConfig := []string{
		fmt.Sprintf("osm.meshName=%s", i.meshName),
		fmt.Sprintf("osm.enforceSingleMesh=%t", i.enforceSingleMesh),
	}

	if err := parseVal(valuesConfig, finalValues); err != nil {
		return nil, err
	}

	return finalValues, nil
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
