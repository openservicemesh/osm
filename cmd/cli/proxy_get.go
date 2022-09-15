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

const getCmdDescription = `
This command will get the Envoy proxy configuration for the given query and pod.
The query is forwarded as is to the Envoy proxy sidecar.
Refer to https://www.envoyproxy.io/docs/envoy/latest/operations/admin for the
list of supported GET queries.
`

const getCmdExample = `
# Get the proxy config dump for the given pod 'bookbuyer-5ccf77f46d-rc5mg' in the 'bookbuyer' namespace
osm proxy get config_dump bookbuyer-5ccf77f46d-rc5mg -n bookbuyer

# Get the cluster config for the given pod 'bookbuyer-5ccf77f46d-rc5mg' in the 'bookbuyer' namespace and output to file 'clusters.txt'
osm proxy get clusters bookbuyer-5ccf77f46d-rc5mg -n bookbuyer -f clusters.txt
`

func newProxyGetCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	adminCmd := &proxyAdminCmd{
		out:        out,
		sigintChan: make(chan os.Signal, 1),
	}

	cmd := &cobra.Command{
		Use:   "get QUERY POD",
		Short: "get query for proxy",
		Long:  getCmdDescription,
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
			return adminCmd.run("GET")
		},
		Example: getCmdExample,
	}

	//add mesh name flag
	f := cmd.Flags()
	f.StringVarP(&adminCmd.namespace, "namespace", "n", metav1.NamespaceDefault, "Namespace of pod")
	f.StringVarP(&adminCmd.outFile, "file", "f", "", "File to write output to")
	f.Uint16VarP(&adminCmd.localPort, "local-port", "p", constants.EnvoyAdminPort, "Local port to use for port forwarding")

	return cmd
}
