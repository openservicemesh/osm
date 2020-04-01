package eds

import (
	"context"

	xds "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/cla"

	"github.com/open-service-mesh/osm/pkg/smi"
)

type empty struct{}

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *xds.DiscoveryRequest) (*xds.DiscoveryResponse, error) {
	proxyServiceName := proxy.GetService()
	allServicesEndpoints, err := catalog.ListEndpoints(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed listing endpoints")
		return nil, err
	}
	log.Debug().Msgf("allServicesEndpoints: %+v", allServicesEndpoints)
	var protos []*any.Any
	for _, serviceEndpoints := range allServicesEndpoints {
		loadAssignment := cla.NewClusterLoadAssignment(serviceEndpoints)

		proto, err := ptypes.MarshalAny(&loadAssignment)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling EDS payload %+v", loadAssignment)
			continue
		}
		protos = append(protos, proto)
	}

	resp := &xds.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   string(envoy.TypeEDS),
	}
	return resp, nil
}
