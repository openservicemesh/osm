package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

const setCmdDescription = `
This command will set the Envoy proxy configuration for the given query and pod.
The query is forwarded as is to the Envoy proxy sidecar as a POST request.
Refer to https://www.envoyproxy.io/docs/envoy/latest/operations/admin for the
list of supported POST queries.
`

const setCmdExample = `
# Reset the proxy stats counters for the pod 'bookbuyer-5ccf77f46d-rc5mg' in the 'bookbuyer' namespace
osm proxy set reset_counters bookbuyer-5ccf77f46d-rc5mg -n bookbuyer
`

func newProxySetCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	adminCmd := &proxyAdminCmd{
		out:        out,
		sigintChan: make(chan os.Signal, 1),
	}

	cmd := &cobra.Command{
		Use:   "set QUERY POD",
		Short: "set query for proxy",
		Long:  setCmdDescription,
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			adminCmd.query = args[0]
			adminCmd.pod = args[1]
			conf, err := config.RESTClientGetter.ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}
			adminCmd.config = conf

			clientset, err := kubernetes.NewForConfig(conf)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			adminCmd.clientSet = clientset
			return adminCmd.run("POST")
		},
		Example: setCmdExample,
	}

	//add mesh name flag
	f := cmd.Flags()
	f.StringVarP(&adminCmd.namespace, "namespace", "n", metav1.NamespaceDefault, "Namespace of pod")
	f.StringVarP(&adminCmd.outFile, "file", "f", "", "File to write output to")
	f.Uint16VarP(&adminCmd.localPort, "local-port", "p", constants.EnvoyAdminPort, "Local port to use for port forwarding")

	return cmd
}
