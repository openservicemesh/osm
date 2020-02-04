package rds

import (
	"fmt"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy/route"
)

func (e *Server) newDiscoveryResponse(allTrafficPolicies []endpoint.TrafficTargetPolicies) (*v2.DiscoveryResponse, error) {
	var protos []*types.Any
	for _, trafficPolicies := range allTrafficPolicies {
		routeConfiguration := route.NewRouteConfiguration(trafficPolicies)

		proto, err := types.MarshalAny(&routeConfiguration)
		if err != nil {
			glog.Errorf("[catalog] Error marshalling RouteConfigurationURI %+v: %s", routeConfiguration, err)
			continue
		}
		protos = append(protos, proto)
	}

	resp := &v2.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   route.RouteConfigurationURI,
	}

	e.lastVersion = e.lastVersion + 1
	e.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = e.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", e.lastVersion)

	return resp, nil
}
