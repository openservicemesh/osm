package endpoint

import (
	"github.com/deislabs/smc/pkg/mesh"
)

// EndpointProvider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers
type Provider interface {
	// Retrieve the IP addresses comprising the ServiceName.
	GetIPs(mesh.ServiceName) []mesh.IP
	GetID() string
	Run(<-chan struct{}) error
}
