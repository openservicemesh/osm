package service

import (
	"fmt"
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
