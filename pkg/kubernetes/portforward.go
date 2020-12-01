package kubernetes

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// PortForwarder is a type that implements port forwarding to a pod
type PortForwarder struct {
	forwarder *portforward.PortForwarder
	stopChan  chan struct{}
	readyChan <-chan struct{}
}

// NewPortForwarder creates a new port forwarder to a pod
func NewPortForwarder(conf *rest.Config, clientSet kubernetes.Interface, podName string, namespace string, localPort uint16, remotePort uint16) (*PortForwarder, error) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(conf)
	if err != nil {
		return nil, errors.Errorf("Error setting up round tripper for port forwarding: %s", err)
	}

	serverURL := clientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward").URL()

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, serverURL)

	stopChan := make(chan struct{})
	readyChan := make(chan struct{})
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopChan, readyChan, ioutil.Discard, os.Stderr)
	if err != nil {
		return nil, errors.Errorf("Error setting up port forwarding: %s", err)
	}

	// Check if the pod exists in the given namespace
	pod, err := clientSet.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Errorf("Error retrieving pod %s in namespace %s: %s", podName, namespace, err)
	}
	if pod.Status.Phase != corev1.PodRunning {
		return nil, errors.Errorf("Pod %s in namespace %s needs to be running before port forwarding, status is %s", podName, namespace, pod.Status.Phase)
	}

	return &PortForwarder{
		forwarder: forwarder,
		stopChan:  stopChan,
		readyChan: readyChan,
	}, nil
}

// Start starts the port forwarding and calls the readyFunc callback function when port forwarding is ready
func (pf *PortForwarder) Start(readyFunc func(pf *PortForwarder) error) error {
	// Set up a channel to process OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)

	// Set up a channel to process forwarding errors
	errChan := make(chan error, 1)

	// Set up forwarding, and relay errors to errChan so that caller is able to handle it
	go func() {
		err := pf.forwarder.ForwardPorts()
		errChan <- err
	}()

	// Stop forwarding in case OS signals are received
	go func() {
		<-sigChan
		if pf.stopChan != nil {
			close(pf.stopChan)
		}
	}()

	// Process signals and call readyFunc
	select {
	case <-pf.readyChan:
		return readyFunc(pf)

	case err := <-errChan:
		return errors.Errorf("Error during port forwarding: %s", err)

	case <-pf.stopChan:
		return nil
	}
}

// Stop stops the port forwarding if not stopped already
func (pf *PortForwarder) Stop() {
	if pf.stopChan != nil {
		close(pf.stopChan)
	}
}
