package rds

import (
	"context"
	"reflect"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/route"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/smi"
	"github.com/deislabs/smc/pkg/utils"
)

type empty struct{}

var packageName = utils.GetLastChunkOfSlashed(reflect.TypeOf(empty{}).PkgPath())

// NewResponse creates a new Route Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy) (*v2.DiscoveryResponse, error) {
	allTrafficPolicies, err := catalog.ListTrafficRoutes("TBD")
	if err != nil {
		glog.Errorf("[%s] Failed listing routes: %+v", packageName, err)
		return nil, err
	}
	glog.V(level.Debug).Infof("[%s] trafficPolicies: %+v", packageName, allTrafficPolicies)

	resp := &v2.DiscoveryResponse{
		TypeUrl: string(envoy.TypeRDS),
	}

	for _, trafficPolicies := range allTrafficPolicies {
		routeConfiguration := route.NewRouteConfiguration(trafficPolicies)

		for _, config := range routeConfiguration {

			marshalledRouteConfig, err := ptypes.MarshalAny(&config)
			if err != nil {
				glog.Errorf("[%s] Failed to marshal route config for proxy %v", packageName, err)
				return nil, err
			}
			resp.Resources = append(resp.Resources, marshalledRouteConfig)
		}
	}
	return resp, nil
}
