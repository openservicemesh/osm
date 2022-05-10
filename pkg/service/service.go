package service

import (
	"reflect"
)

// Equals checks if two namespaced services are equal
func (ms MeshService) Equals(service MeshService) bool {
	// Must eagerly initialize unexported fields to do
	// an accurate comparison
	if !ms.subdomainPopulated {
		ms.Subdomain()
	}

	if !service.subdomainPopulated {
		service.Subdomain()
	}

	ms.ProviderKey()
	service.ProviderKey()
	return reflect.DeepEqual(ms, service)
}

// ServerName returns the Server Name Identifier (SNI) for TLS connections
func (ms MeshService) ServerName() string {
	return ms.FQDN()
}
