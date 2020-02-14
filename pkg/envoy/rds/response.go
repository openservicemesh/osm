package rds

import (
	"fmt"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/route"
	"github.com/deislabs/smc/pkg/log/level"
)

func (s *Server) NewRouteDiscoveryResponse(allTrafficPolicies []endpoint.TrafficTargetPolicies) (*v2.DiscoveryResponse, error) {
	resp := &v2.DiscoveryResponse{
		TypeUrl: envoy.TypeRDS,
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

	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = s.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)
	glog.V(level.Trace).Infof("[%s] Constructed response: %+v", serverName, resp)

	return resp, nil
}
