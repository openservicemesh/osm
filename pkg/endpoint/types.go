package endpoint

import (
	"fmt"
	"net"

	"github.com/open-service-mesh/osm/pkg/service"
)

// Provider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers
type Provider interface {
	// Retrieve the IP addresses comprising the given service.
	ListEndpointsForService(service.Name) []Endpoint

	// Retrieve the namespaced service for a given service account
	GetServiceForServiceAccount(service.ServiceAccount) (service.NamespacedService, error)

	// GetID returns the unique identifier of the EndpointsProvider.
	GetID() string

	// GetAnnouncementsChannel obtains the channel on which providers will announce changes to the infrastructure.
	GetAnnouncementsChannel() <-chan interface{}
}

// Endpoint is a tuple of IP and Port, representing an Envoy proxy, fronting an instance of a service
type Endpoint struct {
	net.IP `json:"ip"`
	Port   `json:"port"`
}

func (ep Endpoint) String() string {
	return fmt.Sprintf("(ip=%s, port=%d)", ep.IP, ep.Port)
}

// Port is a numerical port of an Envoy proxy
type Port uint32
