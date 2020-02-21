package eds

import (
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/cla"
)

// NewDiscoveryResponse creates a new Endpoint Discovery Response.
func (s *Server) NewDiscoveryResponse(proxy *envoy.Proxy) (*v2.DiscoveryResponse, error) {
	allServices, err := s.catalog.ListEndpoints("TBD")
	if err != nil {
		glog.Errorf("[%s] Failed listing endpoints: %+v", serverName, err)
		return nil, err
	}
	glog.Infof("[%s] WeightedServices: %+v", serverName, allServices)
	var protos []*any.Any
	for targetServiceName, weightedServices := range allServices {
		loadAssignment := cla.NewClusterLoadAssignment(targetServiceName, weightedServices)

		proto, err := ptypes.MarshalAny(&loadAssignment)
		if err != nil {
			glog.Errorf("[catalog] Error marshalling EDS payload %+v: %s", loadAssignment, err)
			continue
		}
		protos = append(protos, proto)
	}

	resp := &v2.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   string(envoy.TypeEDS),
	}
	return resp, nil
}
