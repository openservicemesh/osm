// Package service models an instance of a service managed by OSM controller and utility routines associated with it.
package service

import (
	"fmt"
	"strings"

	"github.com/openservicemesh/osm/pkg/identity"
)

const (
	// namespaceNameSeparator used upon marshalling/unmarshalling MeshService to a string
	// or viceversa
	namespaceNameSeparator = "/"
	localCluster           = "local"
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

	ClusterDomain string
}

func (ms MeshService) String() string {
	return fmt.Sprintf("%s%s%s", ms.Namespace, namespaceNameSeparator, ms.Name)
}

// FQDN is similar to String(), but uses a dot separator and is in a different order.
func (ms MeshService) FQDN() string {
	if ms.ClusterDomain == "" {
		ms.ClusterDomain = localCluster
	}
	return strings.Join([]string{ms.Name, ms.Namespace, ms.ClusterDomain}, ".")
}

// Local returns whether or not this is service is in the local cluster.
func (ms MeshService) Local() bool {
	// TODO(steeling): if it's unset consider it local for now.
	return ms.ClusterDomain == localCluster || ms.ClusterDomain == ""
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
	// TODO(whitneygriffith): implement from pkg/kubernetes/client.go:202
	ListServices() ([]MeshService, error)

	// listMeshServices returns all services in the mesh
	// TODO(whitneygriffith): implement from pkg/catalog/service.go:169
	ListMeshServices() ([]MeshService, error)

	// GetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol,
	// where the ports returned are the ones used by downstream clients in their requests. This can be different from the ports
	// actually exposed by the application binary, ie. 'spec.ports[].port' instead of 'spec.ports[].targetPort' for a Kubernetes service.
	// TODO(whitneygriffith): implement from pkg/catalog/service.go:147
	GetPortToProtocolMappingForService(svc MeshService) (map[uint32]string, error)

	// GetTargetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol.
	// The ports returned are the actual ports on which the application exposes the service derived from the service's endpoints,
	// ie. 'spec.ports[].targetPort' instead of 'spec.ports[].port' for a Kubernetes service.
	// The function ensures the port:protocol mapping is the same across different endpoint providers for the service, and returns
	// an error otherwise.
	// TODO(whitneygriffith): implement from pkg/catalog/service.go:116 and pkg/endoint/types.go: 24 and pkg/providers/kube/client.go
	GetTargetPortToProtocolMappingForService(svc MeshService) (map[uint32]string, error)

	// GetHostnamesForService returns a list of hostnames over which the service can be accessed within the local cluster.
	// If 'sameNamespace' is set to true, then the shorthand hostnames service and service:port are also returned.
	// TODO(whitneygriffith): implement from pkg/kubernetes/util.go:21
	GetHostnamesForService(svc MeshService, sameNamespace bool) ([]string, error)

	// GetID returns the unique identifier of the EndpointsProvider.
	GetID() string
}
