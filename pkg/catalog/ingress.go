package catalog

import (
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// GetIngressRoutesPerHost returns route matches per host as defined in observed ingress k8s resources.
func (mc *MeshCatalog) GetIngressRoutesPerHost(service service.MeshService) (map[string][]trafficpolicy.HTTPRouteMatch, error) {
	domainRoutesMap := make(map[string][]trafficpolicy.HTTPRouteMatch)
	ingresses, err := mc.ingressMonitor.GetIngressResources(service)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources with backend %s", service)
		return domainRoutesMap, err
	}
	if len(ingresses) == 0 {
		return domainRoutesMap, err
	}

	defaultRoute := trafficpolicy.HTTPRouteMatch{
		PathRegex: constants.RegexMatchAll,
		Methods:   []string{constants.RegexMatchAll},
	}

	for _, ingress := range ingresses {
		if ingress.Spec.Backend != nil && ingress.Spec.Backend.ServiceName == service.Name {
			domainRoutesMap[constants.WildcardHTTPMethod] = []trafficpolicy.HTTPRouteMatch{defaultRoute}
		}

		for _, rule := range ingress.Spec.Rules {
			domain := rule.Host
			if domain == "" {
				domain = constants.WildcardHTTPMethod
			}
			for _, ingressPath := range rule.HTTP.Paths {
				if ingressPath.Backend.ServiceName != service.Name {
					continue
				}
				routePolicy := defaultRoute
				if routePolicy.PathRegex != "" {
					routePolicy.PathRegex = ingressPath.Path
				}
				domainRoutesMap[domain] = append(domainRoutesMap[domain], routePolicy)
			}
		}
	}

	log.Trace().Msgf("Created routes per host for service %s: %+v", service, domainRoutesMap)

	return domainRoutesMap, nil
}
