package service

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
)

const (
	// namespaceNameSeparator used upon marshalling/unmarshalling MeshService to a string
	// or viceversa
	namespaceNameSeparator = "/"
)

// SyntheticServiceSuffix is a random string appended to the name of the synthetic service created for each K8s service account
var SyntheticServiceSuffix = uuid.New().String()

// MeshService is the struct defining a service (Kubernetes or otherwise) within a service mesh.
type MeshService struct {
	// If the service resides on a Kubernetes service, this would be the Kubernetes namespace.
	Namespace string

	// The name of the service
	Name string
}

func (ms MeshService) String() string {
	return strings.Join([]string{ms.Namespace, namespaceNameSeparator, ms.Name}, "")
}

// Equals checks if two namespaced services are equal
func (ms MeshService) Equals(service MeshService) bool {
	return reflect.DeepEqual(ms, service)
}

// UnmarshalMeshService unmarshals a NamespaceService type from a string
func UnmarshalMeshService(str string) (*MeshService, error) {
	slices := strings.Split(str, namespaceNameSeparator)
	if len(slices) != 2 {
		return nil, errInvalidMeshServiceFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	for _, sep := range slices {
		if len(sep) == 0 {
			return nil, errInvalidMeshServiceFormat
		}
	}

	return &MeshService{
		Namespace: slices[0],
		Name:      slices[1],
	}, nil
}

// ServerName returns the Server Name Identifier (SNI) for TLS connections
func (ms MeshService) ServerName() string {
	return strings.Join([]string{ms.Name, ms.Namespace, "svc", "cluster", "local"}, ".")
}

// K8sServiceAccount is a type for a namespaced service account
type K8sServiceAccount struct {
	Namespace string
	Name      string
}

func (sa K8sServiceAccount) String() string {
	return strings.Join([]string{sa.Namespace, namespaceNameSeparator, sa.Name}, "")
}

// GetSyntheticService creates a MeshService for the given K8s Service Account,
// which has a unique name and is in lieu of an existing Kubernetes service.
func (sa K8sServiceAccount) GetSyntheticService() MeshService {
	return MeshService{
		Namespace: sa.Namespace,
		Name:      fmt.Sprintf("%s.%s.osm.synthetic-%s", sa.Name, sa.Namespace, SyntheticServiceSuffix),
	}
}

// ClusterName is a type for a service name
type ClusterName string

// String returns the given ClusterName type as a string
func (c ClusterName) String() string {
	return string(c)
}

//WeightedService is a struct of a service name, its weight and its root service
type WeightedService struct {
	Service     MeshService `json:"service_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
	RootService string      `json:"root_service:omitempty"`
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	ClusterName ClusterName `json:"cluster_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
}
