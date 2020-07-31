package service

import (
	"reflect"
	"strings"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	// namespaceNameSeparator used upon marshalling/unmarshalling NamespacedService to a string
	// or viceversa
	namespaceNameSeparator = "/"
)

// NamespacedService is the struct defining a service (Kubernetes or otherwise) within a service mesh.
type NamespacedService struct {
	// If the service resides on a Kubernetes service, this would be the Kubernetes namespace.
	Namespace string

	// The name of the service
	Name string
}

func (ms NamespacedService) String() string {
	return strings.Join([]string{ms.Namespace, namespaceNameSeparator, ms.Name}, "")
}

//Equals checks if two namespaced services are equal
func (ms NamespacedService) Equals(service NamespacedService) bool {
	return reflect.DeepEqual(ms, service)
}

// UnmarshalNamespacedService unmarshals a NamespaceService type from a string
func UnmarshalNamespacedService(str string) (*NamespacedService, error) {
	slices := strings.Split(str, namespaceNameSeparator)
	if len(slices) != 2 {
		return nil, errInvalidNamespacedServiceFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	for _, sep := range slices {
		if len(sep) == 0 {
			return nil, errInvalidNamespacedServiceFormat
		}
	}

	return &NamespacedService{
		Namespace: slices[0],
		Name:      slices[1],
	}, nil
}

// GetCommonName returns the Subject CN for the NamespacedService to be used for its certificate.
func (ms NamespacedService) GetCommonName() certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{ms.Name, ms.Namespace, "svc", "cluster", "local"}, "."))
}

// K8sServiceAccount is a type for a namespaced service account
type K8sServiceAccount struct {
	Namespace string
	Name      string
}

func (ns K8sServiceAccount) String() string {
	return strings.Join([]string{ns.Namespace, namespaceNameSeparator, ns.Name}, "")
}

// ClusterName is a type for a service name
type ClusterName string

//WeightedService is a struct of a service name, its weight and domain
type WeightedService struct {
	NamespacedService NamespacedService `json:"service_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
	Domain      string      `json:"domain:omitempty"`
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	ClusterName ClusterName `json:"cluster_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
}
