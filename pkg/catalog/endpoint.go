package catalog

import (
	"net"

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

// GetResolvableServiceEndpoints returns the resolvable set of endpoint over which a service is accessible using its FQDN
func (mc *MeshCatalog) GetResolvableServiceEndpoints(svc service.MeshService) ([]endpoint.Endpoint, error) {
	// TODO: Move the implmentation of this function to be provider-specific. Currently, the providers might
	// not have access to some common structures in order to perform these operations in an optimal way
	var endpoints []endpoint.Endpoint
	var err error

	// Check if the service has been given Cluster IP
	service := mc.kubeController.GetService(svc)
	if service == nil {
		log.Error().Msgf("Could not find service %s", svc.String())
		return nil, errServiceNotFound
	}

	if len(service.Spec.ClusterIP) == 0 {
		// If no cluster IP, use final endpoint as resolvable destinations
		return mc.ListEndpointsForService(svc)
	}

	// Cluster IP is present
	ip := net.ParseIP(service.Spec.ClusterIP)
	if ip == nil {
		log.Error().Msgf("Could not parse Cluster IP %s", service.Spec.ClusterIP)
		return nil, errParseClusterIP
	}

	for _, svcPort := range service.Spec.Ports {
		endpoints = append(endpoints, endpoint.Endpoint{
			IP:   ip,
			Port: endpoint.Port(svcPort.Port),
		})
	}

	return endpoints, err
}
