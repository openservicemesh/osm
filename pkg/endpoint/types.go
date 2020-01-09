package endpoint

import (
	"github.com/deislabs/smc/pkg/mesh"
)

// Provider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers
type Provider interface {
	// Retrieve the IP addresses comprising the given service.
	ListEndpointsForService(mesh.ServiceName) []mesh.Endpoint

	// GetID returns the unique identifier of the EndpointsProvider.
	GetID() string
}
