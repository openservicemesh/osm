package endpoint

import (
	"net"
)

// Provider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers
type Provider interface {
	// Retrieve the IP addresses comprising the given service.
	ListEndpointsForService(ServiceName) []Endpoint

	// GetID returns the unique identifier of the EndpointsProvider.
	GetID() string
}

// Endpoint is a tuple of IP and Port, representing an Envoy proxy, fronting an instance of a service
type Endpoint struct {
	net.IP `json:"ip"`
	Port   `json:"port"`
}

// Port is a numerical port of an Envoy proxy
type Port uint32

// ServiceName is a type for a service name
type ServiceName string

// WeightedService is a struct of a delegated service backing a target service
type WeightedService struct {
	ServiceName ServiceName `json:"service_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
	Endpoints   []Endpoint  `json:"endpoints:omitempty"`
}

// RoutePaths is a struct of a path and the allowed methods on a given route
type RoutePaths struct {
	RoutePathRegex string   `json:"route_path_regex:omitempty"`
	RouteMethods   []string `json:"route_methods:omitempty"`
}

// TrafficTargetPolicies is a struct of the allowed RoutePaths from sources to a destination
type TrafficTargetPolicies struct {
	PolicyName       string     `json:"policy_name:omitempty"`
	Destination      string     `json:"destination:omitempty"`
	Source          string   `json:"sources:omitempty"`
	PolicyRoutePaths []RoutePaths `json:"policy_route_paths:omitempty"`
}
