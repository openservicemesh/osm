package rds

import (
	"fmt"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/route"
	"github.com/deislabs/smc/pkg/log/level"
)

const (
	serverName = "RDS"
)

func (s *Server) NewRouteDiscoveryResponse(proxy envoy.Proxyer) (*v2.DiscoveryResponse, error) {
	allTrafficPolicies, err := s.catalog.ListTrafficRoutes("TBD")
	if err != nil {
		glog.Errorf("[%s][stream] Failed listing routes: %+v", serverName, err)
		return nil, err
	}
	glog.Infof("[%s][stream] trafficPolicies: %+v", serverName, allTrafficPolicies)

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

	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = s.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)
	glog.V(level.Trace).Infof("[%s] Constructed response: %+v", serverName, resp)

	return resp, nil
}
