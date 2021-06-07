// Package provider defines the interface for all providers
// that are primarily responsible for providing information regarding the endpoints for services, such as their IP
// addresses, port numbers and protocol information.
// Reference: https://github.com/openservicemesh/osm/blob/main/DESIGN.md#3-providers

package provider

import (
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// Provider is an interface to be implemented by components abstracting Kubernetes, and other compute/cluster providers
type Provider interface {
	// ListEndpointsForService retrieves the IP addresses comprising the given service.
	ListEndpointsForService(service.MeshService) []endpoint.Endpoint

	// ListEndpointsForIdentity retrieves the list of IP addresses for the given service account
	ListEndpointsForIdentity(identity.ServiceIdentity) []endpoint.Endpoint

	// GetServicesForServiceAccount retrieves the namespaced services for a given service account
	GetServicesForServiceAccount(identity.K8sServiceAccount) ([]service.MeshService, error)

	// GetTargetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol
	GetTargetPortToProtocolMappingForService(service.MeshService) (map[uint32]string, error)

	// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service FQDN is resolved under
	// the scope of the provider
	GetResolvableEndpointsForService(service.MeshService) ([]endpoint.Endpoint, error)

	// GetID returns the unique identifier of the Provider.
	GetID() string
}
