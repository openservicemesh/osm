// Package endpoint defines the interface for an endpoints provider. Endpoints providers communicate with the compute platforms
// and are primarily responsible for providing information regarding the endpoints for services, such as their IP
// addresses, port numbers and protocol information.
// Reference: https://github.com/openservicemesh/osm/blob/main/DESIGN.md#3-endpoints-providers
package endpoint

import (
	"fmt"
	"net"
)

// Endpoint is a tuple of IP and Port representing an instance of a service
type Endpoint struct {
	net.IP   `json:"ip"`
	Port     `json:"port"`
	Weight   `json:"weight"`
	Priority `json:"priority,omitempty"`

	// Zone is the zone the endpoint resides in.
	Zone string `json:"name"`
}

func (ep Endpoint) String() string {
	return fmt.Sprintf("(ip=%s, port=%d)", ep.IP, ep.Port)
}

// Port is a numerical type representing a port on which a service is exposed
type Port uint32

// Weight is the load assignment weight to the endpoint. The assignment works on the endpoints with the same priority.
type Weight uint32

// Priority is the priority of the remote cluster in locality based load balancing
type Priority uint32
