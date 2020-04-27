package catalog

import (
	"strings"

	"github.com/open-service-mesh/osm/pkg/endpoint"
)

// ListTrafficSplitEndpoints constructs a map from service to weighted sub-services with all endpoints the given Envoy proxy should be aware of.
func (sc *MeshCatalog) ListTrafficSplitEndpoints(clientID endpoint.NamespacedService) ([]endpoint.WeightedServiceEndpoints, error) {
	log.Info().Msgf("Listing Endpoints for client: %s", clientID.String())
	// todo (sneha) : TBD if clientID is needed for filtering endpoints
	return sc.getWeightedEndpointsPerService(clientID)
}

func (sc *MeshCatalog) listEndpointsForService(service endpoint.WeightedService) ([]endpoint.Endpoint, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	// todo (sneha) : TBD if clientID is needed for filtering endpoints
	log.Info().Msgf("listEndpointsForService %s", service.ServiceName)
	if _, found := sc.servicesCache[service]; !found {
		sc.refreshCache()
	}
	var endpoints []endpoint.Endpoint
	var found bool
	if endpoints, found = sc.servicesCache[service]; !found {
		log.Error().Msgf("Did not find any Endpoints for service %s", service.ServiceName)
		return nil, errServiceNotFound
	}
	log.Info().Msgf("Found Endpoints=%v for service %s", endpointsToString(endpoints), service.ServiceName)
	return endpoints, nil
}

func (sc *MeshCatalog) getWeightedEndpointsPerService(clientID endpoint.NamespacedService) ([]endpoint.WeightedServiceEndpoints, error) {
	// todo (sneha) : TBD if clientID is needed for filtering endpoints
	var serviceEndpoints []endpoint.WeightedServiceEndpoints

	for _, trafficSplit := range sc.meshSpec.ListTrafficSplits() {
		log.Debug().Msgf("Discovered TrafficSplit resource: %s/%s", trafficSplit.Namespace, trafficSplit.Name)
		if trafficSplit.Spec.Backends == nil {
			log.Error().Msgf("TrafficSplit %s/%s has no Backends in Spec; Skipping...", trafficSplit.Namespace, trafficSplit.Name)
			continue
		}
		domain := trafficSplit.Spec.Service
		for _, trafficSplitBackend := range trafficSplit.Spec.Backends {
			namespacedServiceName := endpoint.NamespacedService{
				Namespace: trafficSplit.Namespace,
				Service:   trafficSplitBackend.Service,
			}
			svcEp := endpoint.WeightedServiceEndpoints{}
			svcEp.WeightedService = endpoint.WeightedService{
				ServiceName: namespacedServiceName,
				Weight:      trafficSplitBackend.Weight,
				Domain:      domain,
			}
			var err error
			if svcEp.Endpoints, err = sc.listEndpointsForService(svcEp.WeightedService); err != nil {
				log.Error().Err(err).Msgf("Error getting Endpoints for service %s", namespacedServiceName)
				svcEp.Endpoints = []endpoint.Endpoint{}
			}
			serviceEndpoints = append(serviceEndpoints, svcEp)
		}
	}
	log.Trace().Msgf("Constructed service endpoints: %+v", serviceEndpoints)
	return serviceEndpoints, nil
}

// endpointsToString stringifies a list of endpoints to a readable form
func endpointsToString(endpoints []endpoint.Endpoint) string {
	var epts []string
	for _, ep := range endpoints {
		epts = append(epts, ep.String())
	}
	return strings.Join(epts, ",")
}

// ListEndpointsForService returns the list of provider endpoints corresponding to a service
func (sc *MeshCatalog) ListEndpointsForService(service endpoint.ServiceName) ([]endpoint.Endpoint, error) {
	var endpoints []endpoint.Endpoint
	for _, provider := range sc.endpointsProviders {
		ep := provider.ListEndpointsForService(service)
		if len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service=%s", provider.GetID(), service)
			continue
		}
		endpoints = append(endpoints, ep...)
	}
	return endpoints, nil
}
