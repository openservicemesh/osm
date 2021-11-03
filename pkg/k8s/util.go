package k8s

import (
	"fmt"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/service"
)

// GetHostnamesForService returns the hostnames over which the service is accessible
func GetHostnamesForService(svc service.MeshService, localNamespace bool) []string {
	var hostnames []string

	if localNamespace {
		hostnames = append(hostnames, []string{
			svc.Name,                                 // service
			fmt.Sprintf("%s:%d", svc.Name, svc.Port), // service:port
		}...)
	}

	hostnames = append(hostnames, []string{
		fmt.Sprintf("%s.%s", svc.Name, svc.Namespace),                                // service.namespace
		fmt.Sprintf("%s.%s:%d", svc.Name, svc.Namespace, svc.Port),                   // service.namespace:port
		fmt.Sprintf("%s.%s.svc", svc.Name, svc.Namespace),                            // service.namespace.svc
		fmt.Sprintf("%s.%s.svc:%d", svc.Name, svc.Namespace, svc.Port),               // service.namespace.svc:port
		fmt.Sprintf("%s.%s.svc.cluster", svc.Name, svc.Namespace),                    // service.namespace.svc.cluster
		fmt.Sprintf("%s.%s.svc.cluster:%d", svc.Name, svc.Namespace, svc.Port),       // service.namespace.svc.cluster:port
		fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace),              // service.namespace.svc.cluster.local
		fmt.Sprintf("%s.%s.svc.cluster.local:%d", svc.Name, svc.Namespace, svc.Port), // service.namespace.svc.cluster.local:port
	}...)

	return hostnames
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

// NamespacedNameFrom returns the namespaced name for the given name if possible, otherwise an error
func NamespacedNameFrom(name string) (types.NamespacedName, error) {
	var nsName types.NamespacedName

	chunks := strings.Split(name, "/")
	if len(chunks) != 2 {
		return nsName, errors.Errorf("%s is not a namespaced name", name)
	}

	nsName.Namespace = chunks[0]
	nsName.Name = chunks[1]

	return nsName, nil
}
