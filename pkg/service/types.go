package service

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	// separator used upon marshalling/unmarshalling Namespaced MeshService to a string
	// or viceversa
	separator = "/"
)

// MeshService is the struct defining a service (Kubernetes or otherwise) within a service mesh.
type MeshService struct {
	// If the service resides on a Kubernetes service, this would be the Kubernetes namespace.
	Namespace string

	// The name of the service
	Name string
}

func (ms MeshService) String() string {
	return fmt.Sprintf("%s/%s", ms.Namespace, ms.Name)
}

//Equals checks if two namespaced services are equal
func (ms MeshService) Equals(service MeshService) bool {
	return reflect.DeepEqual(ms, service)
}

// UnmarshalNamespacedService unmarshals a NamespaceService type from a string
func UnmarshalNamespacedService(str string) (*MeshService, error) {
	slices := strings.Split(str, separator)
	if len(slices) != 2 {
		return nil, errInvalidNamespacedServiceFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	for _, sep := range slices {
		if len(sep) == 0 {
			return nil, errInvalidNamespacedServiceFormat
		}
	}

	return &MeshService{
		Namespace: slices[0],
		Name:      slices[1],
	}, nil
}

// GetCommonName returns the Subject CN for the MeshService to be used for its certificate.
func (ms MeshService) GetCommonName() certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{ms.Name, ms.Namespace, "svc", "cluster", "local"}, "."))
}

// K8sServiceAccount is a type for a namespaced service account
type K8sServiceAccount struct {
	Namespace string
	Name      string
}

func (ns K8sServiceAccount) String() string {
	return fmt.Sprintf("%s/%s", ns.Namespace, ns.Name)
}

// ClusterName is a type for a service name
type ClusterName string

//WeightedService is a struct of a service name, its weight and domain
type WeightedService struct {
	MeshService MeshService `json:"service_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
	Domain      string      `json:"domain:omitempty"`
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	ClusterName ClusterName `json:"cluster_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
}
