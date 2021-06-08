package kubernetes

import (
	"fmt"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	clusterDomain = "cluster.local"
)

// GetHostnamesForService returns a list of hostnames over which the service can be accessed within the local cluster.
// If 'sameNamespace' is set to true, then the shorthand hostnames service and service:port are also returned.
func GetHostnamesForService(svc *corev1.Service, locality service.Locality) []string {
	var domains []string
	if svc == nil {
		return domains
	}

	serviceName := svc.Name
	namespace := svc.Namespace

	if locality == service.LocalNS {
		// Within the same namespace, service name is resolvable to its address
		domains = append(domains, serviceName) // service
	}

	domains = append(domains, fmt.Sprintf("%s.%s", serviceName, namespace))                       // service.namespace
	domains = append(domains, fmt.Sprintf("%s.%s.svc", serviceName, namespace))                   // service.namespace.svc
	domains = append(domains, fmt.Sprintf("%s.%s.svc.cluster", serviceName, namespace))           // service.namespace.svc.cluster
	domains = append(domains, fmt.Sprintf("%s.%s.svc.%s", serviceName, namespace, clusterDomain)) // service.namespace.svc.cluster.local
	for _, portSpec := range svc.Spec.Ports {
		port := portSpec.Port

		if locality == service.LocalNS {
			// Within the same namespace, service name is resolvable to its address
			domains = append(domains, fmt.Sprintf("%s:%d", serviceName, port)) // service:port
		}

		domains = append(domains, fmt.Sprintf("%s.%s:%d", serviceName, namespace, port))                       // service.namespace:port
		domains = append(domains, fmt.Sprintf("%s.%s.svc:%d", serviceName, namespace, port))                   // service.namespace.svc:port
		domains = append(domains, fmt.Sprintf("%s.%s.svc.cluster:%d", serviceName, namespace, port))           // service.namespace.svc.cluster:port
		domains = append(domains, fmt.Sprintf("%s.%s.svc.%s:%d", serviceName, namespace, clusterDomain, port)) // service.namespace.svc.cluster.local:port
	}
	return domains
}

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
