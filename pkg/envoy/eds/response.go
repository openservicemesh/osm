package eds

import (
	"fmt"
	"time"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/envoy/cla"
)

const (
	serverName = "EDS"
)

func (s *Server) NewEndpointDiscoveryResponse(proxy envoy.Proxyer) (*v2.DiscoveryResponse, error) {
	allServices, err := s.catalog.ListEndpoints("TBD")
	if err != nil {
		glog.Errorf("[%s][stream] Failed listing endpoints: %+v", serverName, err)
		return nil, err
	}
	glog.Infof("[%s][stream] WeightedServices: %+v", serverName, allServices)
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

	s.lastVersion = s.lastVersion + 1
	s.lastNonce = string(time.Now().Nanosecond())
	resp.Nonce = s.lastNonce
	resp.VersionInfo = fmt.Sprintf("v%d", s.lastVersion)

	return resp, nil
}
