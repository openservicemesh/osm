package trafficpolicy

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/service"
)

// NewTrafficPolicy returns a new Traffic Policy given a source and destination service, a list of RouteWeightedClusters and hostnames
func NewTrafficPolicy(source, dest service.MeshService, routesClusters []RouteWeightedClusters, hostnames []string) *TrafficPolicy {
	return &TrafficPolicy{
		Name:               fmt.Sprintf("%s-%s", dest.Name, dest.Namespace),
		Source:             source,
		Destination:        dest,
		HTTPRoutesClusters: routesClusters,
		Hostnames:          hostnames,
	}
}
