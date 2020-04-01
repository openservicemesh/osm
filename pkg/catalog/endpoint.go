package catalog

import (
	"strings"

	"github.com/open-service-mesh/osm/pkg/endpoint"
)

type empty struct{}

// ListEndpoints constructs a map from service to weighted sub-services with all endpoints the given Envoy proxy should be aware of.
func (sc *MeshCatalog) ListEndpoints(clientID endpoint.NamespacedService) ([]endpoint.ServiceEndpoints, error) {
	log.Info().Msgf("[%s] Listing Endpoints for client: %s", packageName, clientID.String())
	// todo (sneha) : TBD if clientID is needed for filtering endpoints
	return sc.getWeightedEndpointsPerService(clientID)
}

func (sc *MeshCatalog) listEndpointsForService(service endpoint.WeightedService) ([]endpoint.Endpoint, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	// todo (sneha) : TBD if clientID is needed for filtering endpoints
	log.Info().Msgf("[%s] listEndpointsForService %s", packageName, service.ServiceName)
	if _, found := sc.servicesCache[service]; !found {
		sc.refreshCache()
	}
	var endpoints []endpoint.Endpoint
	var found bool
	if endpoints, found = sc.servicesCache[service]; !found {
		log.Error().Msgf("[%s] Did not find any Endpoints for service %s", packageName, service.ServiceName)
		return nil, errNotFound
	}
	log.Info().Msgf("[%s] Found Endpoints=%v for service %s", packageName, endpointsToString(endpoints), service.ServiceName)
	return endpoints, nil
}

func (sc *MeshCatalog) getWeightedEndpointsPerService(clientID endpoint.NamespacedService) ([]endpoint.ServiceEndpoints, error) {
	// todo (sneha) : TBD if clientID is needed for filtering endpoints
	var serviceEndpoints []endpoint.ServiceEndpoints

	for _, trafficSplit := range sc.meshSpec.ListTrafficSplits() {
		log.Debug().Msgf("[%s] Discovered TrafficSplit resource: %s/%s", packageName, trafficSplit.Namespace, trafficSplit.Name)
		if trafficSplit.Spec.Backends == nil {
			log.Error().Msgf("[%s] TrafficSplit %s/%s has no Backends in Spec; Skipping...", packageName, trafficSplit.Namespace, trafficSplit.Name)
			continue
		}
		for _, trafficSplitBackend := range trafficSplit.Spec.Backends {
			namespacedServiceName := endpoint.NamespacedService{
				Namespace: trafficSplit.Namespace,
				Service:   trafficSplitBackend.Service,
			}
			svcEp := endpoint.ServiceEndpoints{}
			svcEp.WeightedService = endpoint.WeightedService{
				ServiceName: namespacedServiceName,
				Weight:      trafficSplitBackend.Weight,
			}
			var err error
			if svcEp.Endpoints, err = sc.listEndpointsForService(svcEp.WeightedService); err != nil {
				log.Error().Err(err).Msgf("[%s] Error getting Endpoints for service %s", packageName, namespacedServiceName)
				svcEp.Endpoints = []endpoint.Endpoint{}
			}
			serviceEndpoints = append(serviceEndpoints, svcEp)
		}
	}
	log.Trace().Msgf("[%s] Constructed service endpoints: %+v", packageName, serviceEndpoints)
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
