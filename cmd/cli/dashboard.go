package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const openGrafanaDashboardDesc = `
This command will perform a port redirection towards a running
grafana instance running under the OSM namespace, and cast a
generic browser-open towards localhost on the redirected port.

By default redirects through port 3000 unless manually overriden.
This command blocks and redirection remains active until closed
from either side.
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
	log.Printf("[+] Starting Dashboard forwarding\n")

	conf, err := d.config.RESTClientGetter.ToRESTConfig()
	if err != nil {
		log.Fatalf("Failed to get REST config from Helm %s\n", err)
	}

	// Get v1 interface to our cluster. Do or die trying
	clientSet := kubernetes.NewForConfigOrDie(conf)
	v1ClientSet := clientSet.CoreV1()

	// Get Grafana service data
	svc, err := v1ClientSet.Services(settings.Namespace()).
		Get(context.TODO(), grafanaServiceName, metav1.GetOptions{})

	if err != nil {
		log.Fatalf("Failed to get OSM Grafana service data: %s", err)
	}

	// Select pod/s given the service data available
	set := labels.Set(svc.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := v1ClientSet.Pods(settings.Namespace()).
		List(context.TODO(), listOptions)

	// Will select first running Pod available
	it := 0
	for {
		if pods.Items[it].Status.Phase == "Running" {
			break
		}

		it++
		if it == len(pods.Items) {
			log.Fatalf("No running Grafana pod available.")
		}
	}

	// Build http spdy-upgraded handler
	dialer := createDialer(conf, v1ClientSet, pods.Items[it].GetName())

	// StopChan is used to understand the lifecycle of the forwarding blocking op
	// ReadyChan signals when the connection has been established and forwarding is in place
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	forwardStr := fmt.Sprintf("%d:%d", d.localPort, d.remotePort)
	log.Printf("[+] Using forwarding: %s\n", forwardStr)

	forwarder, err := portforward.New(dialer,
		[]string{forwardStr},
		stopChan,
		readyChan,
		out,
		errOut)
	if err != nil {
		log.Fatalf("Failed to create forwarder: %s\n", err)
	}

	// Binding SIGINT & SIGTERM to trigger the closing routine
	signal.Notify(d.sigintChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-d.sigintChan // Blocking
		log.Println("[+] SIGINT/TERM recieved, closing")
		close(stopChan)
	}()

	// This routine blocks on readyChan til forwarding is setup and ready.
	go func() {
		select {
		case <-readyChan:
			break
		}
		log.Printf("[+] Port forwarding successful (localhost:%d)\n", d.localPort)
		if d.openBrowser {
			url := fmt.Sprintf("http://localhost:%d", d.localPort)
			log.Printf("[+] Issuing open browser %s\n", url)
			browser.OpenURL(url)
		}
	}()

	if err = forwarder.ForwardPorts(); err != nil { // Locks until stopChan is closed.
		log.Fatalf("Failed to execute Forward: %s\n", err)
	}

	return nil
}
