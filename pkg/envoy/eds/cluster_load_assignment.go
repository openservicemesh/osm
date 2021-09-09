package eds

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

// newClusterLoadAssignment returns the cluster load assignments for the given service and its endpoints
func newClusterLoadAssignment(svc service.MeshService, serviceEndpoints []endpoint.Endpoint) *xds_endpoint.ClusterLoadAssignment {
	cla := &xds_endpoint.ClusterLoadAssignment{
		ClusterName: svc.EnvoyClusterName(),
		Endpoints: []*xds_endpoint.LocalityLbEndpoints{
			{
				Locality: &xds_core.Locality{
					Zone: zone,
				},
				LbEndpoints: []*xds_endpoint.LbEndpoint{},
			},
		},
	}

	// If there are no service endpoints corresponding to this service, we
	// return a ClusterLoadAssignment without any endpoints.
	// Envoy will correctly handle this response.
	// This can happen if we create a cluster via CDS corresponding to a traffic split
	// apex service that has no endpoints.
	if len(serviceEndpoints) == 0 {
		return cla
	}

	// Equal weight is assigned to a cluster with multiple endpoints in the same locality
	lbWeightPerEndpoint := 100 / len(serviceEndpoints)

	for _, meshEndpoint := range serviceEndpoints {
		log.Trace().Msgf("Adding Endpoint: cluster=%s, endpoint=%s, weight=%d", svc, meshEndpoint, lbWeightPerEndpoint)
		lbEpt := &xds_endpoint.LbEndpoint{
			HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
				Endpoint: &xds_endpoint.Endpoint{
					Address: envoy.GetAddress(meshEndpoint.IP.String(), uint32(meshEndpoint.Port)),
				},
			},
			LoadBalancingWeight: &wrappers.UInt32Value{
				Value: uint32(lbWeightPerEndpoint),
			},
		}
		cla.Endpoints[0].LbEndpoints = append(cla.Endpoints[0].LbEndpoints, lbEpt)
	}

	return cla
}
