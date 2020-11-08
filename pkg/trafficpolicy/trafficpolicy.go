package trafficpolicy

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/service"
)

func NewTrafficPolicy(source, dest service.MeshService, routesClusters []RouteWeightedClusters, hostnames []string) *TrafficPolicy {
	return &TrafficPolicy{
		Name:               fmt.Sprintf("%s-%s-%s-%s", source.Name, source.Namespace, dest.Name, dest.Namespace),
		Source:             source,
		Destination:        dest,
		HTTPRoutesClusters: routesClusters,
		Hostnames:          hostnames,
	}
}

// TotalClustersWeight returns total weight of the WeightedClusters in trafficpolicy.RouteWeightedClusters
func (rwc *RouteWeightedClusters) TotalClustersWeight() int {
	var totalWeight int
	for clusterInterface := range rwc.WeightedClusters.Iter() { // iterate
		cluster := clusterInterface.(service.WeightedCluster)
		totalWeight += cluster.Weight
	}
	return totalWeight
}
