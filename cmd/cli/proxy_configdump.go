package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

const dumpConfigDescription = `
This command will dump the sidecar proxy configuration for the given pod.
`

const dumpConfigExample = `
# Dump the proxy configuration for pod 'bookbuyer-5ccf77f46d-rc5mg' in the 'bookbuyer' namespace
osm proxy dump-config bookbuyer-5ccf77f46d-rc5mg -n bookbuyer
`

type proxyDumpConfigCmd struct {
	out        io.Writer
	config     *rest.Config
	clientSet  kubernetes.Interface
	namespace  string
	pod        string
	localPort  uint16
	sigintChan chan os.Signal
}

func newProxyDumpConfig(config *action.Configuration, out io.Writer) *cobra.Command {
	dumpConfigCmd := &proxyDumpConfigCmd{
		out:        out,
		sigintChan: make(chan os.Signal, 1),
	}

	cmd := &cobra.Command{
		Use:   "dump-config POD",
		Short: "dump proxy config",
		Long:  dumpConfigDescription,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dumpConfigCmd.pod = args[0]
			conf, err := config.RESTClientGetter.ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}
			dumpConfigCmd.config = conf

			clientset, err := kubernetes.NewForConfig(conf)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			dumpConfigCmd.clientSet = clientset
			return dumpConfigCmd.run()
		},
		Example: dumpConfigExample,
	}

	//add mesh name flag
	f := cmd.Flags()
	f.StringVarP(&dumpConfigCmd.namespace, "namespace", "n", metav1.NamespaceDefault, "Namespace of pod")
	f.Uint16VarP(&dumpConfigCmd.localPort, "local-port", "p", constants.EnvoyAdminPort, "Local port to use for port forwarding")

	return cmd
}

func (cmd *proxyDumpConfigCmd) run() error {
	// Check if the pod belongs to a mesh
	pod, err := cmd.clientSet.CoreV1().Pods(cmd.namespace).Get(context.TODO(), cmd.pod, metav1.GetOptions{})
	if err != nil {
		return errors.Errorf("Could not find pod %s in namespace %s", cmd.pod, cmd.namespace)
	}
	if !isMeshedPod(*pod) {
		return errors.Errorf("Pod %s in namespace %s is not a part of a mesh", cmd.pod, cmd.namespace)
	}

	portForwarder, err := k8s.NewPortForwarder(cmd.config, cmd.clientSet, cmd.pod, cmd.namespace, cmd.localPort, constants.EnvoyAdminPort)
	if err != nil {
		return errors.Errorf("Error setting up port forwarding: %s", err)
	}

	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		url := fmt.Sprintf("http://localhost:%d/config_dump", cmd.localPort)

		// #nosec G107: Potential HTTP request made with variable url
		resp, err := http.Get(url)
		if err != nil {
			return errors.Errorf("Error fetching url %s: %s", url, err)
		}
		if _, err := io.Copy(cmd.out, resp.Body); err != nil {
			return errors.Errorf("Error rendering HTTP response: %s", err)
		}
		pf.Stop()
		return nil
	})
	if err != nil {
		return errors.Errorf("Error retrieving proxy config for pod %s in namespace %s: %s", cmd.pod, cmd.namespace, err)
	}

	return nil
}

// isMeshedPod returns a boolean indicating if the pod is part of a mesh
func isMeshedPod(pod corev1.Pod) bool {
	// osm-controller adds a unique label to each pod that belongs to a mesh
	_, proxyLabelSet := pod.Labels[constants.EnvoyUniqueIDLabelName]
	return proxyLabelSet
}
