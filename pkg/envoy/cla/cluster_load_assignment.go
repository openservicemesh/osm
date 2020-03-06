package cla

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/wrappers"

	smcEndpoint "github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
)

const (
	zone = "zone"
)

// NewClusterLoadAssignment constructs the Envoy struct necessary for TrafficSplit implementation.
func NewClusterLoadAssignment(serviceEndpoints smcEndpoint.ServiceEndpoints) v2.ClusterLoadAssignment {
	cla := v2.ClusterLoadAssignment{
		ClusterName: string(serviceEndpoints.Service.ServiceName.String()),
		Endpoints: []*endpoint.LocalityLbEndpoints{
			{
				Locality: &core.Locality{
					Zone: zone,
				},
				LbEndpoints: []*endpoint.LbEndpoint{},
			},
		},
	}

	lenIPs := len(serviceEndpoints.Endpoints)
	if lenIPs == 0 {
		lenIPs = 1
	}
	weight := uint32(100 / lenIPs)

	for _, meshEndpoint := range serviceEndpoints.Endpoints {
		glog.V(level.Trace).Infof("[EDS][ClusterLoadAssignment] Adding Endpoint: Cluster=%s, Services=%s, Endpoint=%+v, Weight=%d\n", serviceEndpoints.Service.ServiceName, serviceEndpoints.Service.ServiceName, meshEndpoint, weight)
		lbEpt := endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: envoy.GetAddress(string(meshEndpoint.IP), uint32(meshEndpoint.Port)),
				},
			},
			LoadBalancingWeight: &wrappers.UInt32Value{
				Value: weight,
			},
		}
		cla.Endpoints[0].LbEndpoints = append(cla.Endpoints[0].LbEndpoints, &lbEpt)
	}
	glog.V(level.Debug).Infof("[EDS] Constructed ClusterLoadAssignment: %+v", cla)
	return cla
}
