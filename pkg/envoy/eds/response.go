package eds

import (
	"context"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/envoy/cla"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/smi"
)

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(ctx context.Context, catalog catalog.MeshCataloger, meshSpec smi.MeshSpec, proxy *envoy.Proxy, request *discovery.DiscoveryRequest, cfg configurator.Configurator) (*discovery.DiscoveryResponse, error) {
	svc, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up Service for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	proxyServiceName := *svc

	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to list traffic policies for proxy service %q", proxyServiceName)
		return nil, err
	}

	allServicesEndpoints := make(map[service.NamespacedService][]endpoint.Endpoint)
	for _, trafficPolicy := range allTrafficPolicies {
		isSourceService := trafficPolicy.Source.Service.Equals(proxyServiceName)
		if isSourceService {
			destService := trafficPolicy.Destination.Service
			serviceEndpoints, err := catalog.ListEndpointsForService(service.Name(destService.String()))
			if err != nil {
				log.Error().Err(err).Msgf("Failed listing endpoints")
				return nil, err
			}
			allServicesEndpoints[destService] = serviceEndpoints
		}
	}

	log.Debug().Msgf("allServicesEndpoints: %+v", allServicesEndpoints)
	var protos []*any.Any
	for serviceName, serviceEndpoints := range allServicesEndpoints {
		loadAssignment := cla.NewClusterLoadAssignment(serviceName, serviceEndpoints)

		proto, err := ptypes.MarshalAny(loadAssignment)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling EDS payload %+v", loadAssignment)
			continue
		}
		protos = append(protos, proto)
	}

	resp := &discovery.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   string(envoy.TypeEDS),
	}
	return resp, nil
}
