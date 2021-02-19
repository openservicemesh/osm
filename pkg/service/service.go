package service

import (
	"reflect"
	"strings"
)

// Equals checks if two namespaced services are equal
func (ms MeshService) Equals(service MeshService) bool {
	return reflect.DeepEqual(ms, service)
}

// IsSyntheticService evaluates the given service name and returns a boolean to denote whether the name
// looks like the name of a synthetic service or not
func (ms MeshService) IsSyntheticService() bool {
	return strings.Contains(ms.Name, ".osm.synthetic-")
}

// ServerName returns the Server Name Identifier (SNI) for TLS connections
func (ms MeshService) ServerName() string {
	return strings.Join([]string{ms.Name, ms.Namespace, "svc", "cluster", "local"}, ".")
}

func (ms MeshService) String() string {
	return strings.Join([]string{ms.Namespace, namespaceNameSeparator, ms.Name}, "")
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
