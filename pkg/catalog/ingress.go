package catalog

import (
	"github.com/open-service-mesh/osm/pkg/endpoint"
)

const (
	// regexMatchAll is a match all regex pattern
	regexMatchAll = ".*"

	// wildcardDomain is a wildcard matching all domains
	wildcardDomain = "*"
)

// IsIngressService returns a boolean indicating if the service is a backend for an ingress resource
func (mc *MeshCatalog) IsIngressService(service endpoint.NamespacedService) (bool, error) {
	policies, err := mc.GetIngressRoutePoliciesPerDomain(service)
	return len(policies) > 0, err
}

// GetIngressRoutePoliciesPerDomain returns the route policies per domain associated with an ingress service
func (mc *MeshCatalog) GetIngressRoutePoliciesPerDomain(service endpoint.NamespacedService) (map[string][]endpoint.RoutePolicy, error) {
	domainRoutesMap := make(map[string][]endpoint.RoutePolicy)
	ingresses, err := mc.ingressMonitor.GetIngressResources(service)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources with backend %s", service)
		return domainRoutesMap, err
	}
	if len(ingresses) == 0 {
		return domainRoutesMap, err
	}

	for _, ingress := range ingresses {
		if ingress.Spec.Backend != nil && ingress.Spec.Backend.ServiceName == service.Service {
			// A default backend rule exists and will be used in
			// case more specific rules are not specified
			defaultRoutePolicy := endpoint.RoutePolicy{
				PathRegex: regexMatchAll,
				Methods:   []string{regexMatchAll},
			}
			domainRoutesMap[wildcardDomain] = append(domainRoutesMap[wildcardDomain], defaultRoutePolicy)
		}
		for _, rule := range ingress.Spec.Rules {
			domain := rule.Host
			if domain == "" {
				domain = wildcardDomain
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
					domainRoutesMap[domain] = append(domainRoutesMap[domain], routePolicy)
				}
			}
		}
	}
	return domainRoutesMap, nil
}

// GetIngressWeightedCluster returns the weighted cluster for an ingress service
func (mc *MeshCatalog) GetIngressWeightedCluster(service endpoint.NamespacedService) (endpoint.WeightedCluster, error) {
	return endpoint.WeightedCluster{
		ClusterName: endpoint.ClusterName(service.String()),
		Weight:      100,
	}, nil
}
