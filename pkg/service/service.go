package service

import (
	"reflect"
)

// Equals checks if two namespaced services are equal
func (ms MeshService) Equals(service MeshService) bool {
	return reflect.DeepEqual(ms, service)
}

// ServerName returns the Server Name Identifier (SNI) for TLS connections
func (ms MeshService) ServerName() string {
	return ms.FQDN()
}
