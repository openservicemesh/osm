// Package trafficpolicy defines the types to represent traffic policies internally in the OSM control plane, and
// utility routines to process them.
package trafficpolicy

import (
	set "github.com/deckarep/golang-set"

	"github.com/openservicemesh/osm/pkg/identity"
)

// TrafficSpecName is the namespaced name of the SMI TrafficSpec
type TrafficSpecName string

// TrafficSpecMatchName is the  name of a match in SMI TrafficSpec
type TrafficSpecMatchName string

// HTTPRouteMatch is a struct to represent an HTTP route match comprised of a path regex, methods, and headers
type HTTPRouteMatch struct {
	PathRegex string            `json:"path_regex:omitempty"`
	Methods   []string          `json:"methods:omitempty"`
	Headers   map[string]string `json:"headers:omitempty"`
}

// TCPRouteMatch is a struct to represent a TCP route matching based on ports
type TCPRouteMatch struct {
	Ports []int `json:"ports:omitempty"`
}

// RouteWeightedClusters is a struct of an HTTPRoute, associated weighted clusters and the domains
type RouteWeightedClusters struct {
	HTTPRouteMatch   HTTPRouteMatch `json:"http_route_match:omitempty"`
	WeightedClusters set.Set        `json:"weighted_clusters:omitempty"`
}

// InboundTrafficPolicy is a struct that associates incoming traffic on a set of Hostnames with a list of Rules
type InboundTrafficPolicy struct {
	Name      string   `json:"name:omitempty"`
	Hostnames []string `json:"hostnames"`
	Rules     []*Rule  `json:"rules:omitempty"`
}

// Rule is a struct that represents which Service Accounts can access a Route
type Rule struct {
	Route                  RouteWeightedClusters `json:"route:omitempty"`
	AllowedServiceAccounts set.Set               `json:"allowed_service_accounts:omitempty"`
}

// OutboundTrafficPolicy is a struct that associates a list of Routes with outbound traffic on a set of Hostnames
type OutboundTrafficPolicy struct {
	Name      string                   `json:"name:omitempty"`
	Hostnames []string                 `json:"hostnames"`
	Routes    []*RouteWeightedClusters `json:"routes:omitempty"`
}

// TrafficTargetWithRoutes is a struct to represent an SMI TrafficTarget resource composed of its associated routes
type TrafficTargetWithRoutes struct {
	Name            string                     `json:"name:omitempty"`
	Destination     identity.ServiceIdentity   `json:"destination:omitempty"`
	Sources         []identity.ServiceIdentity `json:"sources:omitempty"`
	TCPRouteMatches []TCPRouteMatch            `json:"tcp_route_matches:omitempty"`
}
