package catalog

import (
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

// ListEndpointsForService returns the list of provider endpoints corresponding to a service
func (mc *MeshCatalog) ListEndpointsForService(service endpoint.ServiceName) ([]endpoint.Endpoint, error) {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		ep := provider.ListEndpointsForService(endpoint.ServiceName(service.String()))
		if len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service=%s", provider.GetID(), service)
			continue
		}
		endpoints = append(endpoints, ep...)
	}
	return endpoints, nil
}
