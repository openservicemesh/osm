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

// FilterMeshServicesBySubdomain takes a slice of MeshServices and filters them down
// to elements with matching subdomains (optionally including elements with no subdomain)
func FilterMeshServicesBySubdomain(svcs []MeshService, subdomain string, includeEmptySubdomains bool) []MeshService {
	var filteredSvcs []MeshService
	for _, svc := range svcs {
		if svc.Subdomain() == subdomain {
			filteredSvcs = append(filteredSvcs, svc)
			continue
		}

		if includeEmptySubdomains && svc.Subdomain() == "" {
			filteredSvcs = append(filteredSvcs, svc)
			continue
		}
	}

	return filteredSvcs
}
