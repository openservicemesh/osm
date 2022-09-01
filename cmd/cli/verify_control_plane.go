package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/cli/verifier"
)

const verifyControlPlaneDescription = `
This command verifies the health of the OSM control plane.
`

const verifyControlPlaneExample = `
osm verify control-plane-health
`

type verifyControlPlaneCmd struct {
	stdout     io.Writer
	stderr     io.Writer
	restConfig *rest.Config
	kubeClient kubernetes.Interface
}

func newVerifyControlPlaneCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	verifyCmd := &verifyControlPlaneCmd{
		stdout: stdout,
		stderr: stderr,
	}

	cmd := &cobra.Command{
		Use:     "control-plane-health",
		Short:   "verify the health of the OSM control plane",
		Long:    verifyControlPlaneDescription,
		Args:    cobra.NoArgs,
		Example: verifyControlPlaneExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}
			verifyCmd.restConfig = config

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("error initializing client: %w", err)
			}
			verifyCmd.kubeClient = clientset

			return verifyCmd.run()
		},
	}

	return cmd
}

func (cmd *verifyControlPlaneCmd) run() error {
	v := verifier.NewControlPlaneHealthVerifier(cmd.stdout, cmd.stderr, cmd.kubeClient, cmd.restConfig, settings.Namespace())
	result := v.Run()

	fmt.Fprintln(cmd.stdout, "---------------------------------------------")
	verifier.Print(result, cmd.stdout, 1)
	fmt.Fprintln(cmd.stdout, "---------------------------------------------")

	return nil
}
