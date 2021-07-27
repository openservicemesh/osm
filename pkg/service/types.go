// Package service models an instance of a service managed by OSM controller and utility routines associated with it.
package service

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/identity"
)

const (
	// namespaceNameSeparator used upon marshalling/unmarshalling MeshService to a string
	// or viceversa
	namespaceNameSeparator = "/"
)

// Locality is the relative locality of a service. ie: if a service is being accessed from the same namespace or a
// remote cluster.
type Locality int

const (
	// LocalNS refers to the local namespace within the local cluster.
	LocalNS Locality = iota

	// LocalCluster refers to access within the cluster, but not within the same namespace.
	LocalCluster

	// RemoteCluster refers to access from a different cluster.
	RemoteCluster
)

// MeshService is the struct defining a service (Kubernetes or otherwise) within a service mesh.
type MeshService struct {
	// If the service resides on a Kubernetes service, this would be the Kubernetes namespace.
	Namespace string

	// The name of the service
	Name string
}

func (ms MeshService) String() string {
	return fmt.Sprintf("%s%s%s", ms.Namespace, namespaceNameSeparator, ms.Name)
}

// NameWithoutCluster returns a string
func (ms MeshService) NameWithoutCluster() string {
	return fmt.Sprintf("%s%s%s", ms.Namespace, namespaceNameSeparator, ms.Name)
}

// FQDN is similar to String(), but uses a dot separator and is in a different order.
func (ms MeshService) FQDN() string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", ms.Name, ms.Namespace)
}

// ClusterName is a type for a service name
type ClusterName string

// String returns the given ClusterName type as a string
func (c ClusterName) String() string {
	return string(c)
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	ClusterName ClusterName `json:"cluster_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
}

// Provider is an interface to be implemented by components abstracting Kubernetes, and other compute/cluster providers
type Provider interface {
	// GetServicesForServiceIdentity retrieves the namespaced services for a given service identity
	GetServicesForServiceIdentity(identity.ServiceIdentity) ([]MeshService, error)

	// ListServices returns a list of services that are part of monitored namespaces
	ListServices() ([]MeshService, error)

	// ListServiceIdentitiesForService returns service identities for given service
	ListServiceIdentitiesForService(MeshService) ([]identity.ServiceIdentity, error)

	// GetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol,
	// where the ports returned are the ones used by downstream clients in their requests. This can be different from the ports
	// actually exposed by the application binary, ie. 'spec.ports[].port' instead of 'spec.ports[].targetPort' for a Kubernetes service.
	GetPortToProtocolMappingForService(MeshService) (map[uint32]string, error)

	// GetTargetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol.
	// The ports returned are the actual ports on which the application exposes the service derived from the service's endpoints,
	// ie. 'spec.ports[].targetPort' instead of 'spec.ports[].port' for a Kubernetes service.
	GetTargetPortToProtocolMappingForService(MeshService) (map[uint32]string, error)

	// GetHostnamesForService returns a list of hostnames over which the service can be accessed within the local cluster.
	GetHostnamesForService(MeshService, Locality) ([]string, error)

	// GetID returns the unique identifier of the ServiceProvider.
	GetID() string
}
