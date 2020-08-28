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

// GetServiceEndpoints returns the highest abstract set of endpoint destinations where the service is made available at.
// If no LB/virtual IPs are assigned to the service, GetServiceEndpoints will return ListEndpointsForService
func (mc *MeshCatalog) GetServiceEndpoints(svc service.MeshService) ([]endpoint.Endpoint, error) {
	var endpoints []endpoint.Endpoint
	var err error

	// Check if the service has been given Cluster IP
	// TODO: push this in providers. Providers currently do not have services cache.
	service := mc.GetSMISpec().GetService(svc)
	if service == nil {
		log.Error().Msgf("Could not find service %s", svc.String())
		return []endpoint.Endpoint{}, errServiceNotFound
	}

	if len(service.Spec.ClusterIP) > 0 {
		ip := net.ParseIP(service.Spec.ClusterIP)
		if ip == nil {
			log.Error().Msgf("Could not parse IP %s", service.Spec.ClusterIP)
			return []endpoint.Endpoint{}, errParseClusterIP
		}

		for _, svcPort := range service.Spec.Ports {
			endpoints = append(endpoints, endpoint.Endpoint{
				IP:   ip,
				Port: endpoint.Port(svcPort.Port),
			})
		}
	} else {
		endpoints, err = mc.ListEndpointsForService(svc)
	}

	return endpoints, err
}
