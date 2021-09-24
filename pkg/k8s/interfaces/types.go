package interfaces

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// Controller is the controller interface for K8s services
type Controller interface {
	// ListServices returns a list of all (monitored-namespace filtered) services in the mesh
	ListServices() []*corev1.Service

	// ListServiceAccounts returns a list of all (monitored-namespace filtered) service accounts in the mesh
	ListServiceAccounts() []*corev1.ServiceAccount

	// GetService returns a corev1 Service representation if the MeshService exists in cache, otherwise nil
	GetService(svc service.MeshService) *corev1.Service

	// IsMonitoredNamespace returns whether a namespace with the given name is being monitored
	// by the mesh
	IsMonitoredNamespace(string) bool

	// ListMonitoredNamespaces returns the namespaces monitored by the mesh
	ListMonitoredNamespaces() ([]string, error)

	// GetNamespace returns k8s namespace present in cache
	GetNamespace(ns string) *corev1.Namespace

	// ListPods returns a list of pods part of the mesh
	ListPods() []*corev1.Pod

	// ListServiceIdentitiesForService lists ServiceAccounts associated with the given service
	ListServiceIdentitiesForService(service.MeshService) ([]identity.K8sServiceAccount, error)

	// GetEndpoints returns the endpoints for a given service, if found
	GetEndpoints(service.MeshService) (*corev1.Endpoints, error)

	// IsMetricsEnabled returns true if the pod in the mesh is correctly annotated for prometheus scrapping
	IsMetricsEnabled(*corev1.Pod) bool

	// UpdateStatus updates the status subresource for the given resource and GroupVersionKind
	// The object within the 'interface{}' must be a pointer to the underlying resource
	UpdateStatus(interface{}) (metav1.Object, error)

	// K8sServiceToMeshServices translates a k8s service with one or more ports to one or more
	// MeshService objects per port.
	K8sServiceToMeshServices(corev1.Service) []service.MeshService
}
