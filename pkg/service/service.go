package service

import (
	"reflect"
	"strings"

	"github.com/openservicemesh/osm/pkg/constants"
)

// Equals checks if two namespaced services are equal
func (ms MeshService) Equals(service MeshService) bool {
	return reflect.DeepEqual(ms, service)
}

// ServerName returns the Server Name Identifier (SNI) for TLS connections
func (ms MeshService) ServerName() string {
	return strings.Join([]string{ms.Name, ms.Namespace, "svc", "cluster", constants.LocalDomain.String()}, ".")
}
