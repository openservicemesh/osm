package trafficpolicy

import (
	set "github.com/deckarep/golang-set"
	"github.com/open-service-mesh/osm/pkg/service"
)

// Route is a struct of a path regex and the methods on a given route
type Route struct {
	PathRegex string            `json:"path_regex:omitempty"`
	Methods   []string          `json:"methods:omitempty"`
	Headers   map[string]string `json:"headers:omitempty"`
}

// TrafficTarget is a struct of the allowed RoutePaths from sources to a destination
type TrafficTarget struct {
	Name        string          `json:"name:omitempty"`
	Destination TrafficResource `json:"destination:omitempty"`
	Source      TrafficResource `json:"source:omitempty"`
	Route       Route           `json:"route:omitempty"`
}

//TrafficResource is a struct of the various resources of a source/destination in the TrafficPolicy
type TrafficResource struct {
	ServiceAccount service.Account           `json:"service_account:omitempty"`
	Namespace      string                    `json:"namespace:omitempty"`
	Service        service.NamespacedService `json:"services:omitempty"`
}

//RouteWeightedClusters is a struct of a route and the weighted clusters on that route
type RouteWeightedClusters struct {
	Route            Route   `json:"route:omitempty"`
	WeightedClusters set.Set `json:"weighted_clusters:omitempty"`
}
