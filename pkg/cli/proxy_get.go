package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
)

// ExecuteEnvoyAdminReq makes an HTTP request to the Envoy admin server for the given
// request type and query by port-forwarding to the given pod containing the Envoy instance
func ExecuteEnvoyAdminReq(clientSet kubernetes.Interface, config *rest.Config, namespace string, podName string,
	localPort uint16, reqType string, query string) ([]byte, error) {
	portForwarder, err := getPortForwarder(clientSet, config, namespace, podName, localPort)
	if err != nil {
		return nil, fmt.Errorf("error setting up port forwarding: %w", err)
	}

	var envoyProxyConfig []byte
	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		defer pf.Stop()
		url := fmt.Sprintf("http://localhost:%d/%s", localPort, query)

		var resp *http.Response
		var err error

		switch reqType {
		case "GET":
			//#nosec G107: Potential HTTP request made with variable url
			resp, err = http.Get(url)

		case "POST":
			//#nosec G107: Potential HTTP request made with variable url
			resp, err = http.PostForm(url, nil)

		default:
			return fmt.Errorf("expected request type to be one of 'GET|POST', got: %s", reqType)
		}

		if err != nil {
			return fmt.Errorf("error making %s request to url %s: %w", reqType, url, err)
		}

		//nolint: errcheck
		//#nosec G307
		defer resp.Body.Close()

		envoyProxyConfig, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error rendering HTTP response: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving proxy config for pod %s in namespace %s: %w", podName, namespace, err)
	}

	return envoyProxyConfig, nil
}

func getPortForwarder(clientSet kubernetes.Interface, config *rest.Config, namespace string, podName string, localPort uint16) (*k8s.PortForwarder, error) {
	pod, err := clientSet.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting pod %s/%s", namespace, podName)
	}
	if uuid, labelFound := pod.Labels[constants.EnvoyUniqueIDLabelName]; !labelFound || !isValidUUID(uuid) {
		return nil, fmt.Errorf("pod %s/%s is not a part of a mesh", namespace, podName)
	}
	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("pod %s/%s is not running", namespace, podName)
	}

	dialer, err := k8s.DialerToPod(config, clientSet, podName, namespace)
	if err != nil {
		return nil, err
	}

	portForwarder, err := k8s.NewPortForwarder(dialer, fmt.Sprintf("%d:%d", localPort, constants.EnvoyAdminPort))
	if err != nil {
		return nil, err
	}

	return portForwarder, nil
}

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
