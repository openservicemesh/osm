package main

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/cli/verifier"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
)

const verifyConnectivityDescription = `
This command consists of multiple subcommands related to verifying
connectivity related configurations.
`

var (
	fromPod string
	toPod   string
)

type verifyConnectCmd struct {
	stdout      io.Writer
	stderr      io.Writer
	kubeClient  kubernetes.Interface
	srcPod      types.NamespacedName
	dstPod      types.NamespacedName
	appProtocol string
	meshName    string
}

func newVerifyConnectivityCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	verifyCmd := &verifyConnectCmd{
		stdout: stdout,
		stderr: stderr,
	}

	cmd := &cobra.Command{
		Use:   "connectivity",
		Short: "verify connectivity between a pod and a destination",
		Long:  verifyConnectivityDescription,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			verifyCmd.kubeClient = clientset

			namespacedName, err := k8s.NamespacedNameFrom(fromPod)
			if err != nil {
				return errors.Errorf("Source must be a namespaced name of the form <namespace>/<name>, got %s", fromPod)
			}
			verifyCmd.srcPod = namespacedName
			namespacedName, err = k8s.NamespacedNameFrom(toPod)
			if err != nil {
				return errors.Errorf("Destination must be a namespaced name of the form <namespace>/<name>, got %s", toPod)
			}
			verifyCmd.dstPod = namespacedName

			return verifyCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&fromPod, "from-pod", "", "Namespaced name of client pod: <namespace>/<name>")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("from-pod")
	f.StringVar(&toPod, "to-pod", "", "Namespaced name of destination pod: <namespace>/<name>")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("to-pod")
	f.StringVar(&verifyCmd.appProtocol, "app-protocol", constants.ProtocolHTTP, "Application protocol")
	f.StringVar(&verifyCmd.meshName, "mesh-name", defaultMeshName, "Mesh name")

	return cmd
}

func (cmd *verifyConnectCmd) run() error {
	podConnectivityVerifier := verifier.NewPodConnectivityVerifier(cmd.stdout, cmd.stderr, cmd.kubeClient,
		cmd.srcPod, cmd.dstPod, cmd.appProtocol, cmd.meshName)
	result := podConnectivityVerifier.Run()

	fmt.Fprintln(cmd.stdout, "---------------------------------------------")
	verifier.Print(result, cmd.stdout)
	fmt.Fprintln(cmd.stdout, "---------------------------------------------")

	return nil
}
