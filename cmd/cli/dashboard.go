package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport/spdy"
)

const openGrafanaDashboardDesc = `
This command will perform a port redirection towards a running
grafana instance running under the OSM namespace, and cast a
generic browser-open towards localhost on the redirected port.

By default redirects through port 3000 unless manually overridden.
This command blocks if port forwarding is successful until the
process is interrupted with a signal from the OS.
`
const (
	grafanaServiceName = "osm-grafana"
	grafanaWebPort     = 3000
)

type dashboardCmd struct {
	out         io.Writer
	config      *action.Configuration
	localPort   uint16
	remotePort  uint16
	openBrowser bool
	sigintChan  chan os.Signal // Allows interacting with the command from outside
}

func newDashboardCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	dash := &dashboardCmd{
		out:        out,
		config:     config,
		sigintChan: make(chan os.Signal, 1),
	}
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "open grafana dashboard through ssh redirection",
		Long:  openGrafanaDashboardDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			return dash.run()
		},
	}
	cmd.Flags().Uint16VarP(&dash.localPort, "local-port", "p", grafanaWebPort, "Local port to use")
	cmd.Flags().Uint16VarP(&dash.remotePort, "remote-port", "r", grafanaWebPort, "Remote port on Grafana")
	cmd.Flags().BoolVarP(&dash.openBrowser, "open-browser", "b", true, "Triggers browser open, true by default")

	return cmd
}

// Creates an spdy-upgraded http stream handler
func createDialer(conf *rest.Config, v1ClientSet v1.CoreV1Interface, podName string) httpstream.Dialer {
	roundTripper, upgrader, err := spdy.RoundTripperFor(conf)
	if err != nil {
		panic(err)
	}

	serverURL := v1ClientSet.RESTClient().Post().
		Resource("pods").
		Namespace(settings.Namespace()).
		Name(podName).
		SubResource("portforward").URL()

	return spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, serverURL)
}

func (d *dashboardCmd) run() error {
	var err error
	fmt.Fprintf(d.out, "[+] Starting Dashboard forwarding\n")

	conf, err := d.config.RESTClientGetter.ToRESTConfig()
	if err != nil {
		return errors.Errorf("Failed to get REST config from Helm %s\n", err)
	}

	// Get v1 interface to our cluster. Do or die trying
	clientSet := kubernetes.NewForConfigOrDie(conf)
	v1ClientSet := clientSet.CoreV1()

	// Get Grafana service data
	svc, err := v1ClientSet.Services(settings.Namespace()).
		Get(context.TODO(), grafanaServiceName, metav1.GetOptions{})

	if err != nil {
		return errors.Errorf("Failed to get OSM Grafana service data: %s", err)
	}

	// Select pod/s given the service data available
	set := labels.Set(svc.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := v1ClientSet.Pods(settings.Namespace()).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Errorf("Error listing pods: %s", err)
	}

	// Will select first running Pod available
	var grafanaPod *corev1.Pod
	for _, pod := range pods.Items {
		pod := pod // prevents aliasing address of loop variable which is the same in each iteration
		if pod.Status.Phase == "Running" {
			grafanaPod = &pod
			break
		}
	}
	if grafanaPod == nil {
		return errors.Errorf("No running Grafana pod available")
	}

	portForwarder, err := NewPortForwarder(conf, clientSet, grafanaPod.Name, grafanaPod.Namespace, d.localPort, d.remotePort)
	if err != nil {
		return errors.Errorf("Error setting up port forwarding: %s", err)
	}

	err = portForwarder.Start(func(pf *PortForwarder) error {
		if d.openBrowser {
			url := fmt.Sprintf("http://localhost:%d", d.localPort)
			fmt.Fprintf(d.out, "[+] Issuing open browser %s\n", url)
			browser.OpenURL(url)
		}
		return nil
	})
	if err != nil {
		return errors.Errorf("Port forwarding failed: %s", err)
	}

	// The command should only exit when a signal is received from the OS.
	// Exiting before will result in port forwarding to stop causing the browser
	// if open to not render the dashboard.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	return nil
}
