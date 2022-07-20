package main

import (
	"context"
	"fmt"
	"io"

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

const verifyConnectivityExample = `
# Verify connectivity configuration for traffic from pod 'curl/curl-7bb5845476-5b7w'
# to pod 'httpbin/httpbin-69dc7d545c-hbc6d' for the 'httpbin' service:
osm verify connectivity --from-pod curl/curl-7bb5845476-5b7wr --to-pod httpbin/httpbin-69dc7d545c-hbc6d --to-service httpbin

# Verify connectivity configuration for HTTPS traffic from pod 'curl/curl-7bb5845476-zwxbt'
# to external host 'httpbin.org' on port '443':
osm verify connectivity --from-pod curl/curl-7bb5845476-zwxbt --to-ext-port 443 --to-ext-host httpbin.org --app-protocol https

# Verify connectivity configuration for HTTP traffic from pod 'curl/curl-7bb5845476-zwxbt'
# to external host 'httpbin.org' on port '80':
osm verify connectivity --from-pod curl/curl-7bb5845476-zwxbt --to-ext-port 80 --to-ext-host httpbin.org --app-protocol http
`

var (
	fromPod     string
	toPod       string
	appProtocol string
	dstService  string
	toExtPort   uint16
	toExtHost   string
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
		Use:     "connectivity",
		Short:   "verify connectivity between a pod and a destination",
		Long:    verifyConnectivityDescription,
		Args:    cobra.NoArgs,
		Example: verifyConnectivityExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}
			verifyCmd.restConfig = config

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
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
				return fmt.Errorf("--from-pod must be a namespaced name of the form <namespace>/<name>, got %s", fromPod)
			}
			verifyCmd.trafficAttr.SrcPod = &srcName

			if toPod == "" && toExtPort == 0 {
				return fmt.Errorf("one of --to-pod|--to-ext-port must be set")
			}
			if toPod != "" && toExtPort != 0 {
				return fmt.Errorf("--to-pod cannot be set with --to-ext-port")
			}

			if toPod != "" {
				dstName, err := k8s.NamespacedNameFrom(toPod)
				if err != nil {
					return fmt.Errorf("--to-pod must be a namespaced name of the form <namespace>/<name>, got %s", toPod)
				}
				verifyCmd.trafficAttr.DstPod = &dstName

				if dstService == "" {
					return fmt.Errorf("--to-service must be set with --to-pod")
				}
				verifyCmd.trafficAttr.DstService = &types.NamespacedName{Namespace: dstName.Namespace, Name: dstService}
			}

			verifyCmd.trafficAttr.AppProtocol = appProtocol
			verifyCmd.trafficAttr.ExternalPort = toExtPort
			verifyCmd.trafficAttr.ExternalHost = toExtHost

			return verifyCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&fromPod, "from-pod", "", "Namespaced name of client pod: <namespace>/<name>")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("from-pod")
	f.StringVar(&toPod, "to-pod", "", "Namespaced name of destination pod: <namespace>/<name>")
	f.StringVar(&dstService, "to-service", "", "Name of the destination service")
	f.StringVar(&appProtocol, "app-protocol", constants.ProtocolHTTP, "Application protocol")
	f.Uint16Var(&toExtPort, "to-ext-port", 0, "External port")
	f.StringVar(&toExtHost, "to-ext-host", "", "External hostname")
	f.StringVar(&verifyCmd.meshName, "mesh-name", defaultMeshName, "Mesh name")

	return cmd
}

func (cmd *verifyConnectCmd) run() error {
	var verify verifier.Verifier

	switch {
	case cmd.trafficAttr.DstPod != nil:
		verify = verifier.NewPodConnectivityVerifier(cmd.stdout, cmd.stderr, cmd.restConfig,
			cmd.kubeClient, cmd.meshConfig, cmd.trafficAttr, cmd.meshName)

	case cmd.trafficAttr.ExternalPort != 0:
		verify = verifier.NewEgressConnectivityVerifier(cmd.stdout, cmd.stderr, cmd.restConfig,
			cmd.kubeClient, cmd.meshConfig, cmd.trafficAttr, cmd.meshName)

	default:
		return fmt.Errorf("one of --to-pod|to-ext-port must be set")
	}

	result := verify.Run()

	fmt.Fprintln(cmd.stdout, "---------------------------------------------")
	verifier.Print(result, cmd.stdout, 1)
	fmt.Fprintln(cmd.stdout, "---------------------------------------------")

	return nil
}
