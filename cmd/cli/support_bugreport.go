package main

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/bugreport"
	"github.com/openservicemesh/osm/pkg/k8s"
)

const bugReportDescription = `
Generate a bug report.

*Note:
- Both 'osm' and 'kubectl' CLI must reside in the evironment's lookup path.
- If the environment includes sensitive information that should not be collected,
  please do not specify the associated resources.
`

const bugReportExample = `
# Generate a bug report for the given namespaces, deployments, and pods
osm support bug-report --app-namespaces bookbuyer,bookstore \
	--app-deployments bookbuyer/bookbuyer,bookstore/bookstore-v1 \
	--app-pods bookthief/bookthief-7bb7f9b98c-qplq4
`

type bugReportCmd struct {
	stdout         io.Writer
	stderr         io.Writer
	kubeClient     kubernetes.Interface
	appNamespaces  []string
	appDeployments []string
	appPods        []string
	outFile        string
}

func newSupportBugReportCmd(config *action.Configuration, stdout io.Writer, stderr io.Writer) *cobra.Command {
	bugReportCmd := &bugReportCmd{
		stdout: stdout,
		stderr: stderr,
	}

	cmd := &cobra.Command{
		Use:   "bug-report",
		Short: "generate bug report",
		Long:  bugReportDescription,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}
			bugReportCmd.kubeClient, err = kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			return bugReportCmd.run()
		},
		Example: bugReportExample,
	}

	f := cmd.Flags()
	f.StringSliceVar(&bugReportCmd.appNamespaces, "app-namespaces", nil, "Application namespaces")
	f.StringSliceVar(&bugReportCmd.appDeployments, "app-deployments", nil, "Application deployments: <namespace>/<deployment>")
	f.StringSliceVar(&bugReportCmd.appPods, "app-pods", nil, "Application pods: <namespace>/pod")
	f.StringVarP(&bugReportCmd.outFile, "out-file", "o", "", "Output file path")

	return cmd
}

func (cmd *bugReportCmd) run() error {
	var appPods, appDeployments []types.NamespacedName

	for _, pod := range cmd.appPods {
		p, err := k8s.NamespacedNameFrom(pod)
		if err != nil {
			fmt.Fprintf(cmd.stderr, "Pod name %s is not namespaced, skipping it", pod)
			continue
		}
		appPods = append(appPods, p)
	}

	for _, deployment := range cmd.appDeployments {
		d, err := k8s.NamespacedNameFrom(deployment)
		if err != nil {
			fmt.Fprintf(cmd.stderr, "Deployment name %s is not namespaced, skipping it", deployment)
			continue
		}
		appDeployments = append(appDeployments, d)
	}

	bugReportCfg := &bugreport.Config{
		Stdout:               cmd.stdout,
		Stderr:               cmd.stderr,
		KubeClient:           cmd.kubeClient,
		ControlPlaneNamepace: settings.Namespace(),
		AppNamespaces:        cmd.appNamespaces,
		AppDeployments:       appDeployments,
		AppPods:              appPods,
		OutFile:              cmd.outFile,
	}

	return bugReportCfg.Run()
}
