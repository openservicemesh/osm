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

const verifyConnectivityPodToPodDescription = `
This command verifies pod-to-pod connectivity configurations.
`

type verifyConnectPodToPodCmd struct {
	stdout      io.Writer
	stderr      io.Writer
	kubeClient  kubernetes.Interface
	srcPod      types.NamespacedName
	dstPod      types.NamespacedName
	appProtocol string
	meshName    string
}

func newVerifyConnectivityPodToPodCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	verifyConnectPodToPodCmd := &verifyConnectPodToPodCmd{
		stdout: stdout,
		stderr: stderr,
	}

	cmd := &cobra.Command{
		Use:   "pod-to-pod <source-namespace>/<source-pod-name> <destination-namespace>/<destination-pod-name>",
		Short: "verify pod-to-pod connectivity configuration",
		Long:  verifyConnectivityPodToPodDescription,
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			var namespacedName types.NamespacedName
			namespacedName, err := k8s.NamespacedNameFrom(args[0])
			if err != nil {
				return errors.Errorf("Source must be a namespaced name of the form <namespace>/<name>, got %s", args[0])
			}
			verifyConnectPodToPodCmd.srcPod = namespacedName
			namespacedName, err = k8s.NamespacedNameFrom(args[1])
			if err != nil {
				return errors.Errorf("Destination must be a namespaced name of the form <namespace>/<name>, got %s", args[1])
			}
			verifyConnectPodToPodCmd.dstPod = namespacedName

			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			verifyConnectPodToPodCmd.kubeClient = clientset
			return verifyConnectPodToPodCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&verifyConnectPodToPodCmd.appProtocol, "app-protocol", constants.ProtocolHTTP, "Application protocol")
	f.StringVar(&verifyConnectPodToPodCmd.meshName, "mesh-name", defaultMeshName, "Mesh name")

	return cmd
}

func (cmd *verifyConnectPodToPodCmd) run() error {
	podConnectivityVerifier := verifier.NewPodConnectivityVerifier(cmd.stdout, cmd.stderr, cmd.kubeClient,
		cmd.srcPod, cmd.dstPod, cmd.appProtocol, cmd.meshName)
	result := podConnectivityVerifier.Run()

	fmt.Fprintln(cmd.stdout, "---------------------------------------------")
	verifier.Print(result, cmd.stdout)
	fmt.Fprintln(cmd.stdout, "---------------------------------------------")

	return nil
}
