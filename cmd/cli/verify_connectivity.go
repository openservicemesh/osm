package main

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/cli/verifier"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
)

const verifyConnectivityDescription = `
This command consists of multiple subcommands related to verifying
connectivity related configurations.
`

var (
	fromPod     string
	toPod       string
	appProtocol string
	dstService  string
	egress      bool
)

type verifyConnectCmd struct {
	stdout      io.Writer
	stderr      io.Writer
	restConfig  *rest.Config
	kubeClient  kubernetes.Interface
	meshConfig  *configv1alpha2.MeshConfig
	trafficAttr verifier.TrafficAttribute
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
			verifyCmd.restConfig = config

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			verifyCmd.kubeClient = clientset

			configClient, err := configClientset.NewForConfig(config)
			if err != nil {
				return err
			}
			meshConfig, err := configClient.ConfigV1alpha2().MeshConfigs(settings.Namespace()).Get(context.Background(), constants.OSMMeshConfig, metav1.GetOptions{})
			if err != nil {
				return err
			}
			verifyCmd.meshConfig = meshConfig

			srcName, err := k8s.NamespacedNameFrom(fromPod)
			if err != nil {
				return errors.Errorf("--from-pod must be a namespaced name of the form <namespace>/<name>, got %s", fromPod)
			}
			verifyCmd.trafficAttr.SrcPod = &srcName

			if toPod == "" && !egress {
				return errors.New("one of --to-pod|--egress must be set for connectivity verification")
			}
			if toPod != "" && egress {
				return errors.New("--to-pod cannot be set with --egress")
			}
			dstName, err := k8s.NamespacedNameFrom(toPod)
			if err != nil {
				return errors.Errorf("--to-pod must be a namespaced name of the form <namespace>/<name>, got %s", toPod)
			}
			verifyCmd.trafficAttr.DstPod = &dstName

			verifyCmd.trafficAttr.AppProtocol = appProtocol
			verifyCmd.trafficAttr.DstService = &types.NamespacedName{Namespace: dstName.Namespace, Name: dstService}
			verifyCmd.trafficAttr.Egress = egress

			return verifyCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&fromPod, "from-pod", "", "Namespaced name of client pod: <namespace>/<name>")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("from-pod")
	f.StringVar(&toPod, "to-pod", "", "Namespaced name of destination pod: <namespace>/<name>")
	f.BoolVar(&egress, "egress", false, "For egress traffic")

	f.StringVar(&dstService, "service", "", "Name of the destination service")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("service")
	f.StringVar(&appProtocol, "app-protocol", constants.ProtocolHTTP, "Application protocol")
	f.StringVar(&verifyCmd.meshName, "mesh-name", defaultMeshName, "Mesh name")

	return cmd
}

func (cmd *verifyConnectCmd) run() error {
	var verify verifier.Verifier

	switch {
	// Pod-to-pod connectivity verifier
	case cmd.trafficAttr.DstPod != nil:
		verify = verifier.NewPodConnectivityVerifier(cmd.stdout, cmd.stderr, cmd.restConfig,
			cmd.kubeClient, cmd.meshConfig, cmd.trafficAttr, cmd.meshName)

	// Egress connectivity verifier
	case cmd.trafficAttr.Egress:
		verify = verifier.NewEgressConnectivityVerifier(cmd.stdout, cmd.stderr, cmd.restConfig,
			cmd.kubeClient, cmd.meshConfig, cmd.trafficAttr, cmd.meshName)
	}

	result := verify.Run()

	fmt.Fprintln(cmd.stdout, "---------------------------------------------")
	verifier.Print(result, cmd.stdout, 1)
	fmt.Fprintln(cmd.stdout, "---------------------------------------------")

	return nil
}
