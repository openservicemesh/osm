package cla

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/golang/protobuf/ptypes/wrappers"

	osmEndpoint "github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/service"
)

const (
	zone = "zone"
)

// NewClusterLoadAssignment constructs the Envoy struct necessary for TrafficSplit implementation.
func NewClusterLoadAssignment(serviceName service.NamespacedService, serviceEndpoints []osmEndpoint.Endpoint) *v2.ClusterLoadAssignment {
	cla := &v2.ClusterLoadAssignment{
		ClusterName: serviceName.String(),
		Endpoints: []*endpoint.LocalityLbEndpoints{
			{
				Locality: &core.Locality{
					Zone: zone,
				},
				LbEndpoints: []*endpoint.LbEndpoint{},
			},
		},
	}

	lenIPs := len(serviceEndpoints)
	if lenIPs == 0 {
		lenIPs = 1
	}
	weight := uint32(100 / lenIPs)

	for _, meshEndpoint := range serviceEndpoints {
		log.Trace().Msgf("[EDS][ClusterLoadAssignment] Adding Endpoint: Cluster=%s, Services=%s, Endpoint=%+v, Weight=%d", serviceName.String(), serviceName.String(), meshEndpoint, weight)
		lbEpt := endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: envoy.GetAddress(meshEndpoint.IP.String(), uint32(meshEndpoint.Port)),
				},
			},
			LoadBalancingWeight: &wrappers.UInt32Value{
				Value: weight,
			},
		}
		cla.Endpoints[0].LbEndpoints = append(cla.Endpoints[0].LbEndpoints, &lbEpt)
	}
	log.Debug().Msgf("[EDS] Constructed ClusterLoadAssignment: %+v", cla)
	return cla
}
