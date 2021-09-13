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
	localZone             = "local"
	localClusterPriority  = uint32(0)
	remoteClusterPriority = uint32(1)
)

// newClusterLoadAssignment returns the cluster load assignments for the given service and its endpoints
func newClusterLoadAssignment(svc service.MeshService, serviceEndpoints []endpoint.Endpoint) *xds_endpoint.ClusterLoadAssignment {
	localLbEndpoints := &xds_endpoint.LocalityLbEndpoints{
		Locality: &xds_core.Locality{
			Zone: localZone,
		},
		LbEndpoints: []*xds_endpoint.LbEndpoint{},
		Priority:    localClusterPriority,
	}

	cla := &xds_endpoint.ClusterLoadAssignment{
		ClusterName: svc.EnvoyClusterName(),
		Endpoints:   []*xds_endpoint.LocalityLbEndpoints{localLbEndpoints},
	}

	// If there are no service endpoints corresponding to this service, we
	// return a ClusterLoadAssignment without any endpoints.
	// Envoy will correctly handle this response.
	// This can happen if we create a cluster via CDS corresponding to a traffic split
	// apex service that has no endpoints.
	if len(serviceEndpoints) == 0 {
		return cla
	}

	for _, meshEndpoint := range serviceEndpoints {
		lbEpt := &xds_endpoint.LbEndpoint{
			HostIdentifier: &xds_endpoint.LbEndpoint_Endpoint{
				Endpoint: &xds_endpoint.Endpoint{
					Address: envoy.GetAddress(meshEndpoint.IP.String(), uint32(meshEndpoint.Port)),
				},
			},
		}

		// Endpoint without a weight set implies it belongs to the local cluster
		if meshEndpoint.Weight == 0 {
			localLbEndpoints.LbEndpoints = append(localLbEndpoints.LbEndpoints, lbEpt)
			log.Trace().Msgf("Adding local endpoint: cluster=%s, endpoint=%s", svc, meshEndpoint)
			continue
		}

		// Endpoint belongs to a remote cluster, configure its locality
		remoteLbEndpoints := &xds_endpoint.LocalityLbEndpoints{
			Locality: &xds_core.Locality{
				Zone: meshEndpoint.Zone,
			},
			LbEndpoints: []*xds_endpoint.LbEndpoint{lbEpt},
			Priority:    remoteClusterPriority,
			LoadBalancingWeight: &wrappers.UInt32Value{
				Value: uint32(meshEndpoint.Weight),
			},
		}
		if meshEndpoint.Priority != 0 {
			remoteLbEndpoints.Priority = uint32(meshEndpoint.Priority)
		}
		cla.Endpoints = append(cla.Endpoints, remoteLbEndpoints)
		log.Trace().Msgf("Adding Endpoint: cluster=%s, endpoint=%s, weight=%d", svc, meshEndpoint, meshEndpoint.Weight)
	}

	return cla
}
