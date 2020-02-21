package rds

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/route"
)

const (
	serverName = "RDS"
)

// NewDiscoveryResponse creates a new Route Discovery Response.
func (s *Server) NewDiscoveryResponse(proxy *envoy.Proxy) (*v2.DiscoveryResponse, error) {
	allTrafficPolicies, err := s.catalog.ListTrafficRoutes("TBD")
	if err != nil {
		glog.Errorf("[%s] Failed listing routes: %+v", serverName, err)
		return nil, err
	}
	glog.Infof("[%s] trafficPolicies: %+v", serverName, allTrafficPolicies)

	resp := &v2.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	for _, trafficPolicies := range allTrafficPolicies {
		routeConfiguration := route.NewRouteConfiguration(trafficPolicies)

		for _, config := range routeConfiguration {

			marshalledRouteConfig, err := ptypes.MarshalAny(&config)
			if err != nil {
				glog.Errorf("[%s] Failed to marshal route config for proxy %v", serverName, err)
				return nil, err
			}
			resp.Resources = append(resp.Resources, marshalledRouteConfig)
		}
	}
	return resp, nil
}
