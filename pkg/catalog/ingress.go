package catalog

import (
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

const (
	// regexMatchAll is a match all regex pattern
	regexMatchAll = ".*"
)

// IsIngressService returns a boolean indicating if the service is a backend for an ingress resource
func (mc *MeshCatalog) IsIngressService(service endpoint.NamespacedService) (bool, error) {
	_, found, err := mc.GetIngressRoutePoliciesPerDomain(service)
	return found, err
}

// GetIngressRoutePoliciesPerDomain returns the route policies per domain associated with an ingress service
func (mc *MeshCatalog) GetIngressRoutePoliciesPerDomain(service endpoint.NamespacedService) (map[string][]endpoint.RoutePolicy, bool, error) {
	configMap := make(map[string][]endpoint.RoutePolicy)
	ingresses, found, err := mc.ingressMonitor.GetIngressResources(service)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources with backend %s", service)
		return configMap, false, err
	}
	if !found {
		return configMap, false, err
	}

	for _, ingress := range ingresses {
		for _, rule := range ingress.Spec.Rules {
			domain := rule.Host
			if domain == "" {
				domain = "*"
			}
			for _, ingressPath := range rule.HTTP.Paths {
				if ingressPath.Backend.ServiceName == service.Service {
					var pathRegex string
					if ingressPath.Path == "" {
						pathRegex = regexMatchAll
					} else {
						pathRegex = ingressPath.Path
					}
					routePolicy := endpoint.RoutePolicy{
						PathRegex: pathRegex,
						Methods:   []string{regexMatchAll},
					}
					configMap[domain] = append(configMap[domain], routePolicy)
				}
			}
		}
	}
	// TODO: if no path is specified, default
	return configMap, true, nil
}

// GetIngressWeightedCluster returns the weighted cluster for an ingress service
func (mc *MeshCatalog) GetIngressWeightedCluster(service endpoint.NamespacedService) (endpoint.WeightedCluster, error) {
	return endpoint.WeightedCluster{
		ClusterName: endpoint.ClusterName(service.String()),
		Weight:      100,
	}, nil
}
