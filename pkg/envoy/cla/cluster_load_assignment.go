package cla

import (
	"encoding/json"

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
		if delegateServiceJSON, err := json.Marshal(delegateService); err == nil {
			glog.Infof("[CLA] Service %s delegates to %s", targetServiceName, string(delegateServiceJSON))
		} else {
			glog.Error("[CLA] Error marshaling delegate service: ", err)
			glog.Infof("[CLA] Service %s delegates to %+v", targetServiceName, delegateService)
		}
		lenIPs := len(delegateService.Endpoints)
		if lenIPs == 0 {
			lenIPs = 1
		}
		weight := uint32(delegateService.Weight / lenIPs)
		for _, meshEndpoint := range delegateService.Endpoints {
			if ept, err := json.Marshal(meshEndpoint); err == nil {
				glog.Infof("[CLA] Adding Endpoint: Cluster=%s, Services=%s, Endpoint=%s, Weight=%d", targetServiceName, delegateService.ServiceName, string(ept), weight)
			} else {
				glog.Error("[CLA] Error marshalling meshEndpoint: ", meshEndpoint)
			}
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
	glog.V(level.Trace).Infof("[CLA] Constructed ClusterLoadAssignment: %+v", cla)
	return cla
}
