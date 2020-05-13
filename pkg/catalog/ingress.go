package catalog

import (
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
)

// IsIngressService returns a boolean indicating if the service is a backend for an ingress resource
func (mc *MeshCatalog) IsIngressService(service service.NamespacedService) (bool, error) {
	policies, err := mc.GetIngressRoutePoliciesPerDomain(service)
	return len(policies) > 0, err
}

// GetIngressRoutePoliciesPerDomain returns the route policies per domain associated with an ingress service
func (mc *MeshCatalog) GetIngressRoutePoliciesPerDomain(service service.NamespacedService) (map[string][]trafficpolicy.Route, error) {
	domainRoutesMap := make(map[string][]trafficpolicy.Route)
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
			defaultRoutePolicy := trafficpolicy.Route{
				PathRegex: constants.RegexMatchAll,
				Methods:   []string{constants.RegexMatchAll},
			}
			domainRoutesMap[constants.WildcardHTTPMethod] = []trafficpolicy.Route{defaultRoutePolicy}
		}
		for _, rule := range ingress.Spec.Rules {
			domain := rule.Host
			if domain == "" {
				domain = constants.WildcardHTTPMethod
			}
			for _, ingressPath := range rule.HTTP.Paths {
				if ingressPath.Backend.ServiceName == service.Service {
					var pathRegex string
					if ingressPath.Path == "" {
						pathRegex = constants.RegexMatchAll
					} else {
						pathRegex = ingressPath.Path
					}
					routePolicy := trafficpolicy.Route{
						PathRegex: pathRegex,
						Methods:   []string{constants.RegexMatchAll},
					}
					domainRoutesMap[domain] = append(domainRoutesMap[domain], routePolicy)
				}
			}
		}
	}
	return domainRoutesMap, nil
}

// GetIngressWeightedCluster returns the weighted cluster for an ingress service
func (mc *MeshCatalog) GetIngressWeightedCluster(svc service.NamespacedService) (service.WeightedCluster, error) {
	return service.WeightedCluster{
		ClusterName: service.ClusterName(svc.String()),
		Weight:      100,
	}, nil
}
