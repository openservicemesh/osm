package catalog

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
)

// ListEndpointsForService returns the list of provider endpoints corresponding to a service
func (mc *MeshCatalog) ListEndpointsForService(svc service.MeshService) ([]endpoint.Endpoint, error) {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		ep := provider.ListEndpointsForService(svc)
		if len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service=%s", provider.GetID(), svc)
			continue
		}
		endpoints = append(endpoints, ep...)
	}
	return endpoints, nil
}

// ListLocalEndpoints returns the list of endpoints for this kubernetes cluster
func (mc *MeshCatalog) ListLocalClusterEndpoints() (map[string][]EndpointJSON, error) {
	endpointMap := make(map[string][]EndpointJSON)
	services := mc.meshSpec.ListServices()
	for _, provider := range mc.endpointsProviders {
		if provider.GetID() != constants.KubeProviderName {
			continue
		}
		for _, svc := range services {
			log.Trace().Msgf("[ListLocalClusterEndpoints] service=%+v", svc.Name)
			meshSvc := service.MeshService {
				Namespace: "default",
				Name: svc.Name,
			}
			eps := provider.ListEndpointsForService(meshSvc)
			if len(eps) == 0 {
				continue
			}
			// convert to use JSON format as endpoint.Endpoint is not suitable
			var epsJSON = []EndpointJSON{}
			for _, ep := range eps {
				epJSON := EndpointJSON{
					IP:   ep.IP,
					Port: ep.Port,
				}
				epsJSON = append(epsJSON, epJSON)
			}
			log.Trace().Msgf("[ListLocalClusterEndpoints] endpoints for service=%+v", epsJSON)
			meshSvcStr := fmt.Sprintf("%s/%s", meshSvc.Namespace, meshSvc.Name)
			endpointMap[meshSvcStr] = epsJSON
		}
	}
	return endpointMap, nil
}
