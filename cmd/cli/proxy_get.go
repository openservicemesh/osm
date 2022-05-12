package main

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/cli"
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

type proxyGetCmd struct {
	out        io.Writer
	config     *rest.Config
	clientSet  kubernetes.Interface
	query      string
	namespace  string
	pod        string
	localPort  uint16
	outFile    string
	sigintChan chan os.Signal
}

func newProxyGetCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	getCmd := &proxyGetCmd{
		out:        out,
		sigintChan: make(chan os.Signal, 1),
	}

	cmd := &cobra.Command{
		Use:   "get QUERY POD",
		Short: "get query for proxy",
		Long:  getCmdDescription,
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			getCmd.query = args[0]
			getCmd.pod = args[1]
			conf, err := config.RESTClientGetter.ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}
			getCmd.config = conf

			clientset, err := kubernetes.NewForConfig(conf)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			getCmd.clientSet = clientset
			return getCmd.run()
		},
		Example: getCmdExample,
	}

	//add mesh name flag
	f := cmd.Flags()
	f.StringVarP(&getCmd.namespace, "namespace", "n", metav1.NamespaceDefault, "Namespace of pod")
	f.StringVarP(&getCmd.outFile, "file", "f", "", "File to write output to")
	f.Uint16VarP(&getCmd.localPort, "local-port", "p", constants.EnvoyAdminPort, "Local port to use for port forwarding")

	return cmd
}

func (cmd *proxyGetCmd) run() error {
	envoyProxyConfig, err := cli.GetEnvoyProxyConfig(cmd.clientSet, cmd.config, cmd.namespace, cmd.pod, cmd.localPort, cmd.query)
	if err != nil {
		return err
	}

	out := cmd.out // By default, output is written to stdout
	if cmd.outFile != "" {
		fd, err := os.Create(cmd.outFile)
		if err != nil {
			return errors.Errorf("Error opening file %s: %s", cmd.outFile, err)
		}
		//nolint: errcheck
		//#nosec G307
		defer fd.Close()
		out = fd // write output to file
	}

	_, err = out.Write(envoyProxyConfig)
	return err
}

// isMeshedPod returns a boolean indicating if the pod is part of a mesh
func isMeshedPod(pod corev1.Pod) bool {
	// osm-controller adds a unique label to each pod that belongs to a mesh
	_, proxyLabelSet := pod.Labels[constants.EnvoyUniqueIDLabelName]
	return proxyLabelSet
}
