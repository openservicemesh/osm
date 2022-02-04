package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/pkg/browser"

	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/k8s"
)

const openOsmControllerDebugServerDesc = `
TODO
`

type osmControllerDebugServerCmd struct {
	out         io.Writer
	config      *rest.Config
	clientSet   kubernetes.Interface
	namespace   string
	localPort   uint16
	remotePort  uint16
	openBrowser bool
	sigintChan  chan os.Signal // Allows interacting with the command from outside
}

func newOsmControllerDebugServerCmd(out io.Writer) *cobra.Command {
	debugServerCmd := &osmControllerDebugServerCmd{
		out:        out,
		sigintChan: make(chan os.Signal, 1),
	}

	cmd := &cobra.Command{
		Use:   "controller-debug-server",
		Short: "open osm-controller debug server through redirection",
		Long:  openOsmControllerDebugServerDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}
			debugServerCmd.config = config

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			debugServerCmd.clientSet = clientset

			debugServerCmd.namespace = settings.Namespace()
			return debugServerCmd.run()
		},
	}

	cmd.Flags().Uint16VarP(&debugServerCmd.localPort, "local-port", "p", constants.DebugPort, "Local port to use")
	cmd.Flags().Uint16VarP(&debugServerCmd.remotePort, "remote-port", "r", constants.DebugPort, "Remote port on osm-controller's debug server")
	cmd.Flags().BoolVarP(&debugServerCmd.openBrowser, "open-browser", "b", true, "Triggers browser open, true by default")

	return cmd
}

func (d *osmControllerDebugServerCmd) run() error {
	fmt.Fprintf(d.out, "[+] Starting osm-controller debug server forwarding\n")

	meshControllerPods := k8s.GetOSMControllerPods(d.clientSet, d.namespace)
	if len(meshControllerPods.Items) <= 0 {
		return errors.Errorf("Could not find any osm-controller pods in namespace %s", d.namespace)
	}
	// Will select first running Pod available
	var osmControllerPod *corev1.Pod
	for _, p := range meshControllerPods.Items {
		pod := p // prevents aliasing address of loop variable which is the same in each iteration
		if pod.Status.Phase == corev1.PodRunning {
			osmControllerPod = &pod
			break
		}
	}
	if osmControllerPod == nil {
		return errors.Errorf("Could not find any running osm-controller pods in namespace %s", d.namespace)
	}

	fmt.Fprintf(d.out, "[+] Performing port redirection for pod [%s] in namespace [%s] for remote port [%d] to local port [%d]\n",
		osmControllerPod.Name, osmControllerPod.Namespace, d.remotePort, d.localPort)

	dialer, err := k8s.DialerToPod(d.config, d.clientSet, osmControllerPod.Name, osmControllerPod.Namespace)
	if err != nil {
		return err
	}
	portForwarder, err := k8s.NewPortForwarder(dialer, fmt.Sprintf("%d:%d", d.localPort, d.remotePort))
	if err != nil {
		return fmt.Errorf("error setting up port forwarding: %s", err)
	}

	err = portForwarder.Start(func(*k8s.PortForwarder) error {
		if d.openBrowser {
			url := fmt.Sprintf("http://localhost:%d%s", d.localPort, constants.DebugServerIndexPath)
			fmt.Fprintf(d.out, "[+] Issuing open browser %s\n", url)
			_ = browser.OpenURL(url)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("port forwarding failed: %s", err)
	}

	// The command should only exit when a signal is received from the OS.
	// Exiting before will result in port forwarding to stop causing the browser
	// if open to not render the dashboard.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	// portforwarder.Stop() triggered implicitly by SIGINT. Ensure it completes
	// before exiting.
	<-portForwarder.Done()

	return nil
}
