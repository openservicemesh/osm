package trafficpolicy

import (
	set "github.com/deckarep/golang-set"

	"github.com/openservicemesh/osm/pkg/service"
)

// TrafficSpecName is the namespaced name of the SMI TrafficSpec
type TrafficSpecName string

// TrafficSpecMatchName is the  name of a match in SMI TrafficSpec
type TrafficSpecMatchName string

// Route is a struct of a path regex and the methods on a given route
type Route struct {
	PathRegex string            `json:"path_regex:omitempty"`
	Methods   []string          `json:"methods:omitempty"`
	Headers   map[string]string `json:"headers:omitempty"`
}

// TrafficTarget is a struct of the allowed RoutePaths from sources to a destination
type TrafficTarget struct {
	Name        string              `json:"name:omitempty"`
	Destination service.MeshService `json:"destination:omitempty"`
	Source      service.MeshService `json:"source:omitempty"`
	Route       Route               `json:"route:omitempty"`
}

//RouteWeightedClusters is a struct of a route and the weighted clusters on that route
type RouteWeightedClusters struct {
	Route            Route   `json:"route:omitempty"`
	WeightedClusters set.Set `json:"weighted_clusters:omitempty"`
}
