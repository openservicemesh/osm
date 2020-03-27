package endpoint

import (
	"fmt"
	"net"
)

// Provider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers
type Provider interface {
	// Retrieve the IP addresses comprising the given service.
	ListEndpointsForService(ServiceName) []Endpoint

	// Retrieve the servics for a given service account
	ListServicesForServiceAccount(NamespacedServiceAccount) []NamespacedService

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

// ServiceName is a type for a service name
type ServiceName string

func (s ServiceName) String() string {
	return string(s)
}

// NamespacedService is a type for a namespaced service
type NamespacedService struct {
	Namespace string
	Service   string
}

func (ns NamespacedService) String() string {
	return fmt.Sprintf("%s/%s", ns.Namespace, ns.Service)
}

// ServiceAccount is a type for a service account
type ServiceAccount string

func (s ServiceAccount) String() string {
	return string(s)
}

// NamespacedServiceAccount is a type for a namespaced service account
type NamespacedServiceAccount struct {
	Namespace      string
	ServiceAccount string
}

func (ns NamespacedServiceAccount) String() string {
	return fmt.Sprintf("%s/%s", ns.Namespace, ns.ServiceAccount)
}

// ClusterName is a type for a service name
type ClusterName string

//WeightedService is a struct of a service name and its weight
type WeightedService struct {
	ServiceName NamespacedService `json:"service_name:omitempty"`
	Weight      int               `json:"weight:omitempty"`
}

// ServiceEndpoints is a struct of a weighted service and its endpoints
type ServiceEndpoints struct {
	WeightedService WeightedService `json:"service:omitempty"`
	Endpoints       []Endpoint      `json:"endpoints:omitempty"`
}

// RoutePaths is a struct of a path and the allowed methods on a given route
type RoutePaths struct {
	RoutePathRegex string   `json:"route_path_regex:omitempty"`
	RouteMethods   []string `json:"route_methods:omitempty"`
}

// TrafficTargetPolicies is a struct of the allowed RoutePaths from sources to a destination
type TrafficTargetPolicies struct {
	PolicyName       string          `json:"policy_name:omitempty"`
	Destination      TrafficResource `json:"destination:omitempty"`
	Source           TrafficResource `json:"source:omitempty"`
	PolicyRoutePaths []RoutePaths    `json:"policy_route_paths:omitempty"`
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	ClusterName ClusterName `json:"cluster_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
}

//TrafficResource is a struct of the various resources of a source/destination in the TrafficTargetPolicies
type TrafficResource struct {
	ServiceAccount ServiceAccount      `json:"service_account:omitempty"`
	Namespace      string              `json:"namespace:omitempty"`
	Services       []NamespacedService `json:"services:omitempty"`
	Clusters       []WeightedCluster   `json:"clusters:omitempty"`
}
