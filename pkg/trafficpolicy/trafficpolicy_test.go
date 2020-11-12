package trafficpolicy

import (
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/service"
)

func TestNewTrafficPolicy(t *testing.T) {
	assert := assert.New(t)

	source := service.MeshService{
		Name:      "source-service",
		Namespace: "source-ns",
	}
	dest := service.MeshService{
		Name:      "dest-service",
		Namespace: "dest-ns",
	}
	routesClusters := []RouteWeightedClusters{
		{
			HTTPRoute: HTTPRoute{
				PathRegex: "/hello",
			},
			WeightedClusters: mapset.NewSet(service.WeightedCluster{
				ClusterName: "dest-ns/dest-service",
				Weight:      100,
			}),
		},
	}
	hostnames := []string{"dest-service.dest-ns"}

	expected := &TrafficPolicy{
		Name:               "dest-service-dest-ns",
		Source:             source,
		Destination:        dest,
		HTTPRoutesClusters: routesClusters,
		Hostnames:          hostnames,
	}

	actual := NewTrafficPolicy(source, dest, routesClusters, hostnames)
	assert.Equal(expected, actual)
}
