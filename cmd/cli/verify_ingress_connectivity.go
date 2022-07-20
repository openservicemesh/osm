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
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/cli/verifier"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
)

const verifyIngressDescription = `
This command consists of multiple subcommands related to verifying
ingress connectivity related configurations.
`

const verifyIngressConnectivityExample = `
# Verify ingress connectivity configuration for traffic from service 'ingress-nginx/ingress-nginx-controller'
# to pod 'httpbin/httpbin-7c6464475-456pn' for the 'httpbin' service:
osm verify ingress --from-service ingress-nginx/ingress-nginx-controller --to-pod httpbin/httpbin-7c6464475-456pn --to-service httpbin --ingress-backend httpbin --to-port 14001
`

var (
	fromIngressService string
	backendPod         string
	backendProtocol    string
	backendPort        uint16
	backendService     string
	ingressBackend     string
)

type verifyIngressConnectCmd struct {
	stdout       io.Writer
	stderr       io.Writer
	restConfig   *rest.Config
	kubeClient   kubernetes.Interface
	policyClient policyClientset.Interface
	meshConfig   *configv1alpha2.MeshConfig
	trafficAttr  verifier.TrafficAttribute
	meshName     string
}

func newVerifyIngressConnectivityCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	verifyIngressCmd := &verifyIngressConnectCmd{
		stdout: stdout,
		stderr: stderr,
	}

	cmd := &cobra.Command{
		Use:     "ingress",
		Short:   "verify ingress connectivity between an ingress service and a destination",
		Long:    verifyIngressDescription,
		Example: verifyIngressConnectivityExample,
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}
			verifyIngressCmd.restConfig = config

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			verifyIngressCmd.kubeClient = clientset

			policyClient, err := policyClientset.NewForConfig(config)
			if err != nil {
				return err
			}
			verifyIngressCmd.policyClient = policyClient

			configClient, err := configClientset.NewForConfig(config)
			if err != nil {
				return err
			}
			meshConfig, err := configClient.ConfigV1alpha2().MeshConfigs(settings.Namespace()).Get(context.Background(), constants.OSMMeshConfig, metav1.GetOptions{})
			if err != nil {
				return err
			}
			verifyIngressCmd.meshConfig = meshConfig

			verifyIngressCmd.trafficAttr.IsIngress = true

			srcName, err := k8s.NamespacedNameFrom(fromIngressService)
			if err != nil {
				return fmt.Errorf("--from-service must be a namespaced name of the form <namespace>/<name>, got %s", fromPod)
			}
			verifyIngressCmd.trafficAttr.SrcService = &srcName
			dstName, err := k8s.NamespacedNameFrom(backendPod)
			if err != nil {
				return fmt.Errorf("--to-pod pod must be a namespaced name of the form <namespace>/<name>, got %s", toPod)
			}
			verifyIngressCmd.trafficAttr.DstPod = &dstName
			verifyIngressCmd.trafficAttr.DstPort = backendPort

			verifyIngressCmd.trafficAttr.AppProtocol = backendProtocol
			verifyIngressCmd.trafficAttr.DstService = &types.NamespacedName{Namespace: dstName.Namespace, Name: backendService}
			// IngressBackend must be in same namespace as backend service
			verifyIngressCmd.trafficAttr.IngressBackend = &types.NamespacedName{Namespace: dstName.Namespace, Name: ingressBackend}

			return verifyIngressCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&fromIngressService, "from-service", "", "Namespaced name of ingress service: <namespace>/<name>")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("from-service")
	f.StringVar(&backendPod, "to-pod", "", "Namespaced name of destination pod: <namespace>/<name>")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("to-pod")
	f.StringVar(&backendService, "to-service", "", "Name of the destination service")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("to-service")
	f.StringVar(&ingressBackend, "ingress-backend", "", "Name of ingress backend")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("ingress-backend")
	f.Uint16Var(&backendPort, "to-port", 0, "Target port the backend pod is listening on")
	//nolint: errcheck
	//#nosec G104: Errors unhandled
	cmd.MarkFlagRequired("to-port")
	f.StringVar(&backendProtocol, "app-protocol", constants.ProtocolHTTP, "Application protocol")
	f.StringVar(&verifyIngressCmd.meshName, "mesh-name", defaultMeshName, "Mesh name")

	return cmd
}

func (cmd *verifyIngressConnectCmd) run() error {
	v := verifier.NewIngressConnectivityVerifier(cmd.stdout, cmd.stderr, cmd.restConfig,
		cmd.kubeClient, cmd.policyClient, cmd.meshConfig, cmd.trafficAttr, cmd.meshName)
	result := v.Run()

	fmt.Fprintln(cmd.stdout, "---------------------------------------------")
	verifier.Print(result, cmd.stdout, 1)
	fmt.Fprintln(cmd.stdout, "---------------------------------------------")

	return nil
}
