package service

import (
	"reflect"
	"strings"
	"strconv"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	// namespaceNameSeparator used upon marshalling/unmarshalling MeshService to a string
	// or viceversa
	namespaceNameSeparator = "/"
)

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

// GetCommonName returns the Subject CN for the MeshService to be used for its certificate.
func (ms MeshService) GetCommonName() certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{ms.Name, ms.Namespace, "svc", "cluster", "local"}, "."))
}

func (ms MeshService) GetMeshServicePort() MeshServicePort {
	return MeshServicePort{
		Namespace: ms.Namespace,
		Name: ms.Name,
		Port: 0,
	}
}

type MeshServicePort struct {
	// If the service resides on a Kubernetes service, this would be the Kubernetes namespace.
	Namespace string

	// The name of the service
	Name string

	// Service port
	Port int
}

func (ms MeshServicePort) GetMeshService() MeshService {
	return MeshService{
		Namespace: ms.Namespace,
		Name: ms.Name,
	}
}

func (ms MeshServicePort) String() string {
	return strings.Join([]string{ms.Namespace, namespaceNameSeparator, ms.Name, namespaceNameSeparator, strconv.Itoa(ms.Port)}, "")
}

// Equals checks if two namespaced services are equal
func (ms MeshServicePort) Equals(service MeshServicePort) bool {
	return reflect.DeepEqual(ms, service)
}

// UnmarshalMeshServicePort unmarshals a NamespaceService type from a string
func UnmarshalMeshServicePort(str string) (*MeshServicePort, error) {
	slices := strings.Split(str, namespaceNameSeparator)
	if len(slices) != 3 {
		return nil, errInvalidMeshServiceFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	for i, sep := range slices {
		if i == 2 {
			// Port can be empty
			continue
		}
		if len(sep) == 0 {
			return nil, errInvalidMeshServiceFormat
		}
	}

	port := 0
	if slices[2] != "" {
		port, _ = strconv.Atoi(slices[2])
	}

	return &MeshServicePort{
		Namespace: slices[0],
		Name:      slices[1],
		Port:      port,
	}, nil
}

// GetCommonName returns the Subject CN for the MeshServicePort to be used for its certificate.
func (ms MeshServicePort) GetCommonName() certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{ms.Name, ms.Namespace, "svc", "cluster", "local", strconv.Itoa(ms.Port)}, "."))
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
