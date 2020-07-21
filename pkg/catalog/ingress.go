package catalog

import (
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// GetIngressRoutesPerHost returns routes per host as defined in observed ingress k8s resources.
func (mc *MeshCatalog) GetIngressRoutesPerHost(service service.NamespacedService) (map[string][]trafficpolicy.Route, error) {
	hostRoutes := make(map[string][]trafficpolicy.Route)
	ingresses, err := mc.ingressMonitor.GetIngressResources(service)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources with backend %s", service)
		return hostRoutes, err
	}
	if len(ingresses) == 0 {
		return hostRoutes, err
	}

	// A default backend rule exists and will be used in case more specific rules are not specified
	defaultRoute := trafficpolicy.Route{
		PathRegex: constants.RegexMatchAll,
		Methods:   []string{constants.RegexMatchAll},
	}

	for _, ingress := range ingresses {
		if ingress.Spec.Backend != nil && ingress.Spec.Backend.ServiceName == service.Service {
			hostRoutes[constants.WildcardHTTPMethod] = []trafficpolicy.Route{defaultRoute}
			hostRoutes[`*`] = []trafficpolicy.Route{defaultRoute}
		}

		for _, rule := range ingress.Spec.Rules {
			host := rule.Host
			if host == "" {
				host = `*`
			}
			for _, ingressPath := range rule.HTTP.Paths {
				if ingressPath.Backend.ServiceName != service.Service {
					continue
				}

				routePolicy := defaultRoute
				if routePolicy.PathRegex != "" {
					routePolicy.PathRegex = ingressPath.Path
				}
				hostRoutes[host] = append(hostRoutes[host], routePolicy)
			}
		}
	}

	log.Trace().Msgf("Created routes per host for service %s: %+v", service, hostRoutes)

	return hostRoutes, nil
}
