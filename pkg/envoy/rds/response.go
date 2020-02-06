package rds

import (
	"fmt"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy/route"
)

func (e *Server) newRouteDiscoveryResponse(allTrafficPolicies []endpoint.TrafficTargetPolicies) (*v2.DiscoveryResponse, error) {
	/*var protos []*any.Any
	for _, trafficPolicies := range allTrafficPolicies {
		//routeConfiguration := route.NewRouteConfiguration(trafficPolicies)


		proto, err := ptypes.MarshalAny(&routeConfiguration)
		if err != nil {
			glog.Errorf("[catalog] Error marshalling RouteConfigurationURI %+v: %s", routeConfiguration, err)
			continue
		}
		protos = append(protos, proto)
	}*/

	resp := &v2.DiscoveryResponse{
		TypeUrl: route.RouteConfigurationURI,
	}

	serverRouteConfig := route.GetServerRouteConfiguration()
	clientRouteConfig := route.GetClientRouteConfiguration()

	glog.Infof("[RDS] Constructed Server RouteConfig: %+v", serverRouteConfig)
	glog.Infof("[RDS] Constructed Client RouteConfig: %+v", clientRouteConfig)

	marshalledSerever, err := ptypes.MarshalAny(&serverRouteConfig)
	if err != nil {
		glog.Errorf("[%s] Failed to marshal server route config for proxy %v", serverName, err)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledSerever)

	marshalledClient, err := ptypes.MarshalAny(&clientRouteConfig)
	if err != nil {
		glog.Errorf("[%s] Failed to marshal client route config for proxy %v", serverName, err)
		return nil, err
	}
	resp.Resources = append(resp.Resources, marshalledClient)

	e.lastVersion = e.lastVersion + 1
	e.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = e.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", e.lastVersion)
	glog.Infof("[%s] Constructed response: %+v", serverName, resp)
	return resp, nil
}
