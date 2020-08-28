package eds

import (
	"context"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/cla"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(_ context.Context, catalog catalog.MeshCataloger, proxy *envoy.Proxy, request *xds_discovery.DiscoveryRequest, cfg configurator.Configurator) (*xds_discovery.DiscoveryResponse, error) {
	svc, err := catalog.GetServiceFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	proxyServiceName := *svc

	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	log.Debug().Msgf("EDS svc %s allTrafficPolicies %+v", proxyServiceName, allTrafficPolicies)

	if err != nil {
		log.Error().Err(err).Msgf("Failed to list traffic policies for proxy service %q", proxyServiceName)
		return nil, err
	}

	allServicesEndpoints := make(map[service.MeshServicePort][]endpoint.Endpoint)
	for _, trafficPolicy := range allTrafficPolicies {
		isSourceService := trafficPolicy.Source.Equals(proxyServiceName)
		if isSourceService {
			destService := trafficPolicy.Destination.GetMeshService()
			serviceEndpoints, err := catalog.ListEndpointsForService(destService)
			if err != nil {
				log.Error().Err(err).Msgf("Failed listing endpoints for proxy %s", proxyServiceName)
				return nil, err
			}
			destServicePort := trafficPolicy.Destination
			if destServicePort.Port == 0  {
				allServicesEndpoints[destServicePort] = serviceEndpoints
				continue
			}
			// if port specified, filter based on port
			filteredEndpoints := make([]endpoint.Endpoint, 0)
			for _, endpoint := range serviceEndpoints {
				if int(endpoint.Port) != destServicePort.Port {
					continue
				}
				filteredEndpoints = append(filteredEndpoints, endpoint)
			}
			allServicesEndpoints[destServicePort] = filteredEndpoints
		}
	}

	log.Debug().Msgf("Computed endpoints for proxy %s: %+v", proxyServiceName, allServicesEndpoints)
	var protos []*any.Any
	for serviceName, serviceEndpoints := range allServicesEndpoints {
		loadAssignment := cla.NewClusterLoadAssignment(serviceName, serviceEndpoints)

		proto, err := ptypes.MarshalAny(loadAssignment)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling EDS payload for proxy %s: %+v", proxyServiceName, loadAssignment)
			continue
		}
		protos = append(protos, proto)
	}

	log.Debug().Msgf("EDS url:%s protos: %+v", string(envoy.TypeEDS), protos)
	resp := &xds_discovery.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   string(envoy.TypeEDS),
	}
	return resp, nil
}
