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
func NewClusterLoadAssignment(targetServiceName smcEndpoint.ServiceName, weightedServices []smcEndpoint.WeightedService) v2.ClusterLoadAssignment {
	cla := v2.ClusterLoadAssignment{
		// NOTE: results.ServiceName is the top level service that is cURLed.
		ClusterName: string(targetServiceName),
		Endpoints: []*endpoint.LocalityLbEndpoints{
			{
				Locality: &core.Locality{
					Zone: zone,
				},
				LbEndpoints: []*endpoint.LbEndpoint{},
			},
		},
	}

	for _, delegateService := range weightedServices {
		glog.Infof("Adding delegate service %+v to target service %s", delegateService, targetServiceName)
		lenIPs := len(delegateService.Endpoints)
		if lenIPs == 0 {
			lenIPs = 1
		}
		weight := uint32(delegateService.Weight / lenIPs)
		for _, meshEndpoint := range delegateService.Endpoints {
			glog.Infof("[EDS][ClusterLoadAssignment] Adding Endpoint: Cluster=%s, Services=%s, Endpoint=%+v, Weight=%d\n", targetServiceName, delegateService.ServiceName, meshEndpoint, weight)
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
	}
	glog.V(level.Trace).Infof("[EDS] Constructed ClusterLoadAssignment: %+v", cla)
	return cla
}
