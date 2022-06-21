package k8s

import (
	"bytes"
	"fmt"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
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
func splitHostName(c Controller, host string) (svc string, subdomain string) {
	host = strings.Split(host, ":")[0] // chop port off the end

	serviceComponents := strings.Split(host, ".")

	// The service name is usually the first string in the host name for a service.
	// Ex. service.namespace, service.namespace.svc.cluster.local
	// However, if there's a subdomain, we the service name is the second string.
	// Ex. mysql-0.service.namespace, mysql-0.service.namespace.svc.cluster.local, mysql-0.service.namespace.svc.cluster.local
	switch l := len(serviceComponents); {
	case l == 1:
		// e.g. service
		svc = serviceComponents[0]
		subdomain = ""
	case l == 2:
		// e.g. service.namespace, mysql-0.service
		p1 := serviceComponents[0] // service name or pod name
		p2 := serviceComponents[1] // namespace name or service name

		// by default, assume service.namespace
		svc = p1
		subdomain = ""

		if c == nil {
			// no controller was passed in; default to non-heuristic behavior
			return
		}

		ns := c.GetNamespace(p2)
		if ns == nil {
			// namespace not present in cache/doesn't exist; this is probably subdomain.service
			subdomain = p1
			svc = p2
			return
		}

		// namespace does exist in the cache, so this is service.namespace
	case l == 3:
		tld := serviceComponents[2]

		if c == nil {
			// use a more basic heuristic since we don't have a kubecontroller
			if tld == "svc" {
				// e.g. service.namespace.svc
				svc = serviceComponents[0]
				subdomain = ""
				return
			}

			// e.g. mysql-0.service.namespace
			svc = serviceComponents[1]
			subdomain = serviceComponents[0]
			return
		}

		ns := c.GetNamespace(tld)
		if ns == nil {
			// tld isn't a namespace; so this is service.namespace.svc
			svc = serviceComponents[0]
			subdomain = ""
			return
		}

		// tld is a namespace, so this is mysql-0.service.namespace
		svc = serviceComponents[1]
		subdomain = serviceComponents[0]
	case l == 4:
		// e.g mysql-0.service.namespace.svc
		svc = serviceComponents[1]
		subdomain = serviceComponents[0]
	case l == 5:
		// e.g. service.namespace.svc.cluster.local
		svc = serviceComponents[0]
		subdomain = ""
	case l == 6:
		// e.g. mysql-0.service.namespace.svc.cluster.local
		svc = serviceComponents[1]
		subdomain = serviceComponents[0]
	default:
		svc = serviceComponents[0]
		subdomain = ""
	}

	return
}

// GetServiceFromHostname returns the service name from its hostname
// This assumes the default k8s trustDomain: cluster.local
func GetServiceFromHostname(c Controller, host string) string {
	svc, _ := splitHostName(c, host)
	return svc
}

// GetSubdomainFromHostname returns the service subdomain from its hostname
// This assumes the default k8s trustDomain: cluster.local
func GetSubdomainFromHostname(c Controller, host string) string {
	_, subdomain := splitHostName(c, host)
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

func StopJobSidecars(stop <-chan struct{}, broker *messaging.Broker, config *rest.Config, clientset kubernetes.Interface) {
	pods := broker.GetKubeEventPubSub().Sub(string(announcements.PodUpdated))
	for {
		select {
		case <-stop:
			return
		case e := <-pods:
			pod := e.(events.PubSubMessage).NewObj.(*corev1.Pod)
			isJob := false
			for _, o := range pod.OwnerReferences {
				if o.Kind == "Job" && o.APIVersion == "batch/v1" {
					isJob = true
					break
				}
			}
			if !isJob {
				continue
			}

			var sidecars []string
			otherContainersRunning := false
			for _, s := range pod.Status.ContainerStatuses {
				if s.Name == constants.EnvoyContainerName || s.Name == "osm-healthcheck" {
					if s.State.Running != nil {
						sidecars = append(sidecars, s.Name)
					}
				} else if s.State.Running != nil {
					otherContainersRunning = true
					break
				}
			}
			if otherContainersRunning {
				continue
			}

			for _, sidecar := range sidecars {
				log.Debug().Msgf("stopping container %s in pod %s/%s", sidecar, pod.Namespace, pod.Name)

				req := clientset.CoreV1().RESTClient().
					Post().
					Resource("pods").
					Name(pod.Name).
					Namespace(pod.Namespace).
					SubResource("exec")

				req.VersionedParams(
					&corev1.PodExecOptions{
						Command:   []string{"kill", "1"},
						Container: sidecar,
						Stdout:    true,
						Stderr:    true,
					},
					scheme.ParameterCodec,
				)
				exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
				if err != nil {
					log.Error().Err(err).Msg("failed to initialize executor")
					continue
				}
				stdout := bytes.NewBuffer(nil)
				stderr := bytes.NewBuffer(nil)
				err = exec.Stream(remotecommand.StreamOptions{
					Stdout: stdout,
					Stderr: stderr,
				})
				if err != nil {
					log.Error().Err(err).Msg("command failed")
					continue
				}

				log.Debug().Str("stdout", stdout.String()).Str("stderr", stderr.String()).Msgf("stopped container %s in pod %s/%s", sidecar, pod.Namespace, pod.Name)
			}
		}
	}
}
