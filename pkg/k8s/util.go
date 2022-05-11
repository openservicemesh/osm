package k8s

import (
	"fmt"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

// splitHostName takes a k8s FQDN (i.e. host) and retrieves the service name
// as well as the subdomain (may be empty)
func splitHostName(host string) (service string, subdomain string) {
	serviceComponents := strings.Split(host, ".")

	// The service name is usually the first string in the host name for a service.
	// Ex. service.namespace, service.namespace.svc.cluster.local
	// However, if there's a subdomain, we the service name is the second string.
	// Ex. mysql-0.service.namespace, mysql-0.service.namespace.svc.cluster.local, mysql-0.service.namespace.svc.cluster.local
	switch l := len(serviceComponents); {
	case l == 1:
		// e.g. service
		service = serviceComponents[0]
		subdomain = ""
	case l == 2:
		// e.g. service.namespace, mysql-0.service
		// NOTE: There's no way to tell which component is the service or not, so we're going to keep the default behavior
		// Thus, we have a requirement that users who want to reference statefulsets must include the namespace in the host name
		service = serviceComponents[0]
		subdomain = ""
	case l == 3:
		tld := serviceComponents[l-1]
		// tld may contain a port
		tldComponents := strings.Split(tld, ":")
		if len(tldComponents) > 1 {
			// port detected
			tld = tldComponents[0]
		}

		if tld == "svc" {
			// e.g. service.namespace.svc
			service = serviceComponents[0]
			subdomain = ""
		} else {
			// e.g. mysql-0.service.namespace
			service = serviceComponents[1]
			subdomain = serviceComponents[0]
		}
	case l == 4:
		// e.g mysql-0.service.namespace.svc
		service = serviceComponents[1]
		subdomain = serviceComponents[0]
	case l == 5:
		// e.g. service.namespace.svc.cluster.local
		service = serviceComponents[0]
		subdomain = ""
	case l == 6:
		// e.g. mysql-0.service.namespace.svc.cluster.local
		service = serviceComponents[1]
		subdomain = serviceComponents[0]
	default:
		service = serviceComponents[0]
		subdomain = ""
	}

	// For services that are not namespaced the service name contains the port as well
	// Ex. service:port
	service = strings.Split(service, ":")[0]

	return
}

// GetServiceFromHostname returns the service name from its hostname
// This assumes the default k8s trustDomain: cluster.local
func GetServiceFromHostname(host string) string {
	svc, _ := splitHostName(host)
	return svc
}

// GetSubdomainFromHostname returns the service subdomain from its hostname
// This assumes the default k8s trustDomain: cluster.local
func GetSubdomainFromHostname(host string) string {
	_, subdomain := splitHostName(host)
	return subdomain
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

// IsHeadlessService determines whether or not a corev1.Service is a headless service
func IsHeadlessService(svc corev1.Service) bool {
	return len(svc.Spec.ClusterIP) == 0 || svc.Spec.ClusterIP == corev1.ClusterIPNone
}
