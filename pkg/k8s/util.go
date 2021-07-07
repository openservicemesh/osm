package k8s

import (
	"strings"

	goversion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

// GetServiceFromHostname returns the service name from its hostname
func GetServiceFromHostname(host string) string {
	// The service name is the first string in the host name for a service.
	// Ex. service.namespace, service.namespace.cluster.local
	service := strings.Split(host, ".")[0]

	// For services that are not namespaced the service name contains the port as well
	// Ex. service:port
	return strings.Split(service, ":")[0]
}

// GetAppProtocolFromPortName returns the port's application protocol from its name, defaults to 'http' if not specified.
func GetAppProtocolFromPortName(portName string) string {
	portName = strings.ToLower(portName)

	switch {
	case strings.HasPrefix(portName, "http-"):
		return "http"

	case strings.HasPrefix(portName, "tcp-"):
		return "tcp"

	case strings.HasPrefix(portName, "grpc-"):
		return "grpc"

	default:
		return constants.ProtocolHTTP
	}
}

// GetKubernetesServerVersionNumber returns the Kubernetes server version number in chunks, ex. v1.19.3 => [1, 19, 3]
func GetKubernetesServerVersionNumber(kubeClient kubernetes.Interface) ([]int, error) {
	if kubeClient == nil {
		return nil, errors.Errorf("Kubernetes client is not initialized")
	}

	version, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, errors.Errorf("Error getting K8s server version: %s", err)
	}

	ver, err := goversion.NewVersion(version.String())
	if err != nil {
		return nil, errors.Errorf("Error parsing k8s server version %s: %s", version, err)
	}

	return ver.Segments(), nil
}
