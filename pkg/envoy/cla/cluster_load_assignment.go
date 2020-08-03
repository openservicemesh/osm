package cla

import (
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"

	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	zone = "zone"
)

// NewClusterLoadAssignment constructs the Envoy struct necessary for TrafficSplit implementation.
func NewClusterLoadAssignment(serviceName service.MeshService, serviceEndpoints []endpoint.Endpoint) *xds_endpoint.ClusterLoadAssignment {
	cla := &xds_endpoint.ClusterLoadAssignment{
		ClusterName: serviceName.String(),
		Endpoints: []*xds_endpoint.LocalityLbEndpoints{
			{
				Locality: &xds_core.Locality{
					Zone: zone,
				},
				LbEndpoints: []*xds_endpoint.LbEndpoint{},
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
		lbEpt := xds_endpoint.LbEndpoint{
			HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
				Endpoint: &xds_endpoint.Endpoint{
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
