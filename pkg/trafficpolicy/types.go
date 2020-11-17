package trafficpolicy

import (
	set "github.com/deckarep/golang-set"

	"github.com/openservicemesh/osm/pkg/service"
)

// TrafficSpecName is the namespaced name of the SMI TrafficSpec
type TrafficSpecName string

// TrafficSpecMatchName is the  name of a match in SMI TrafficSpec
type TrafficSpecMatchName string

// HTTPRoute is a struct to represent an HTTP route comprised of a path regex, methods, and headers
type HTTPRoute struct {
	PathRegex string            `json:"path_regex:omitempty"`
	Methods   []string          `json:"methods:omitempty"`
	Headers   map[string]string `json:"headers:omitempty"`
}

// TrafficTarget is a struct to represent a traffic policy between a source and destination along with its routes
type TrafficTarget struct {
	Name        string              `json:"name:omitempty"`
	Destination service.MeshService `json:"destination:omitempty"`
	Source      service.MeshService `json:"source:omitempty"`
	HTTPRoutes  []HTTPRoute         `json:"http_route:omitempty"`
}

// TrafficPolicy represents route configuration from a source to a destination with an associated set of hostnames
type TrafficPolicy struct {
	Name               string                  `json:"name:omitempty"`
	Destination        service.MeshService     `json:"destination:omitempty"`
	Source             service.MeshService     `json:"source:omitempty"`
	HTTPRoutesClusters []RouteWeightedClusters `json:"http_routes:omitempty"`
	Hostnames          []string                `json:"hostnames:omitempty"`
}

// RouteWeightedClusters is a struct of an HTTPRoute, associated weighted clusters and the domains
type RouteWeightedClusters struct {
	HTTPRoute        HTTPRoute `json:"http_route:omitempty"`
	WeightedClusters set.Set   `json:"weighted_clusters:omitempty"`
	Hostnames        set.Set   `json:"hostnames:omitempty"`
}
