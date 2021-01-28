package kubernetes

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	clusterDomain      = "cluster.local"
	defaultAppProtocol = "http"
)

// GetHostnamesForService returns a list of hostnames over which the service can be accessed within the local cluster.
// If 'sameNamespace' is set to true, then the shorthand hostnames service and service:port are also returned.
func GetHostnamesForService(service *corev1.Service, sameNamespace bool) []string {
	var domains []string
	if service == nil {
		return domains
	}

	serviceName := service.Name
	namespace := service.Namespace

	if sameNamespace {
		// Within the same namespace, service name is resolvable to its address
		domains = append(domains, serviceName) // service
	}

	domains = append(domains, fmt.Sprintf("%s.%s", serviceName, namespace))                       // service.namespace
	domains = append(domains, fmt.Sprintf("%s.%s.svc", serviceName, namespace))                   // service.namespace.svc
	domains = append(domains, fmt.Sprintf("%s.%s.svc.cluster", serviceName, namespace))           // service.namespace.svc.cluster
	domains = append(domains, fmt.Sprintf("%s.%s.svc.%s", serviceName, namespace, clusterDomain)) // service.namespace.svc.cluster.local
	for _, portSpec := range service.Spec.Ports {
		port := portSpec.Port

		if sameNamespace {
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

// GetAppProtocolFromPortName returns the port's application protocol from its name, 'defaultAppProtocol' if not specified.
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
		return defaultAppProtocol
	}
}
