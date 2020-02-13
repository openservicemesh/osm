package eds

import (
	"fmt"
	"github.com/deislabs/smc/pkg/envoy"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy/cla"
)

func (e *Server) newEndpointDiscoveryResponse(allServices map[endpoint.ServiceName][]endpoint.WeightedService) (*v2.DiscoveryResponse, error) {
	var protos []*any.Any
	for targetServiceName, weightedServices := range allServices {
		loadAssignment := cla.NewClusterLoadAssignment(targetServiceName, weightedServices)

		proto, err := ptypes.MarshalAny(&loadAssignment)
		if err != nil {
			glog.Errorf("[catalog] Error marshalling TypeCLA %+v: %s", loadAssignment, err)
			continue
		}
		protos = append(protos, proto)
	}

	resp := &v2.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   envoy.TypeCLA,
	}

	e.lastVersion = e.lastVersion + 1
	e.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = e.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", e.lastVersion)

	return resp, nil
}
