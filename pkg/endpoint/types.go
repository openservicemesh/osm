// Package endpoint defines the interface for an endpoint
package endpoint

import (
	"fmt"
	"net"
)

// Endpoint is a tuple of IP and Port representing an instance of a service
type Endpoint struct {
	net.IP `json:"ip"`
	Port   `json:"port"`
}

func (ep Endpoint) String() string {
	return fmt.Sprintf("(ip=%s, port=%d)", ep.IP, ep.Port)
}

// Port is a numerical type representing a port on which a service is exposed
type Port uint32
