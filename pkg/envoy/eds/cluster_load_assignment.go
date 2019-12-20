package eds

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/mesh"
)

const (
	zone = "zone"
)

func newClusterLoadAssignment(targetServiceName mesh.ServiceName, weightedServices []mesh.WeightedService) v2.ClusterLoadAssignment {
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
		lenIPs := len(delegateService.IPs)
		if lenIPs == 0 {
			lenIPs = 1
		}
		weight := uint32(delegateService.Weight / lenIPs)
		for _, ip := range delegateService.IPs {
			glog.Infof("[EDS][ClusterLoadAssignment] Adding Endpoint: Cluster=%s, Service=%s, IP=%s, Weight=%d\n", targetServiceName, delegateService.ServiceName, ip, weight)
			lbEpt := endpoint.LbEndpoint{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.TCP,
									Address:  string(ip),
									PortSpecifier: &core.SocketAddress_PortValue{
										// TODO(draychev): discover the port dynamically - service catalog
										PortValue: uint32(15003),
									},
								},
							},
						},
					},
				},
				LoadBalancingWeight: &types.UInt32Value{
					Value: weight,
				},
			}
			cla.Endpoints[0].LbEndpoints = append(cla.Endpoints[0].LbEndpoints, &lbEpt)
		}
	}
	glog.V(7).Infof("[EDS] Constructed ClusterLoadAssignment: %+v", cla)
	return cla
}
