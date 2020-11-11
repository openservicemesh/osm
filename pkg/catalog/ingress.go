package catalog

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

func (mc *MeshCatalog) GetIngressTrafficPolicies(svc service.MeshService) ([]*trafficpolicy.TrafficPolicy, error) {
	ingresses, err := mc.ingressMonitor.GetIngressResources(svc)
	policies := []*trafficpolicy.TrafficPolicy{}
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources with backend %s", svc)
		return nil, err
	}
	if len(ingresses) == 0 {
		log.Debug().Msgf("No ingress resources found for %s", svc)
		return policies, nil
	}

	/* TODO WIP
	defaultRoute := trafficpolicy.HTTPRoute{
		PathRegex: constants.RegexMatchAll,
		Methods:   []string{constants.RegexMatchAll},
	}
	*/

	for _, ingress := range ingresses {
		id := rand.New(rand.NewSource(time.Now().UnixNano()))
		policy := &trafficpolicy.TrafficPolicy{
			Name:        fmt.Sprintf("%s-%s-ingress-%v", svc.Name, svc.Namespace, id),
			Destination: svc,
		}
		log.Debug().Msgf("Ingress %#v", ingress)
		/* TODO WIP
		if ingress.Spec.Backend != nil && ingress.Spec.Backend.ServiceName == service.Name {
			domainRoutesMap[constants.WildcardHTTPMethod] = []trafficpolicy.HTTPRoute{defaultRoute}
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
		*/
		policies = append(policies, policy)
	}
	return policies, nil
}

func (mc *MeshCatalog) buildIngressTrafficPolicy(service service.MeshService) (*trafficpolicy.TrafficPolicy, error) {
	return nil, nil
}

// GetIngressRoutesPerHost returns routes per host as defined in observed ingress k8s resources.
func (mc *MeshCatalog) GetIngressRoutesPerHost(service service.MeshService) (map[string][]trafficpolicy.HTTPRoute, error) {
	domainRoutesMap := make(map[string][]trafficpolicy.HTTPRoute)
	ingresses, err := mc.ingressMonitor.GetIngressResources(service)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources with backend %s", service)
		return domainRoutesMap, err
	}
	if len(ingresses) == 0 {
		return domainRoutesMap, err
	}

	defaultRoute := trafficpolicy.HTTPRoute{
		PathRegex: constants.RegexMatchAll,
		Methods:   []string{constants.RegexMatchAll},
	}

	for _, ingress := range ingresses {
		if ingress.Spec.Backend != nil && ingress.Spec.Backend.ServiceName == service.Name {
			domainRoutesMap[constants.WildcardHTTPMethod] = []trafficpolicy.HTTPRoute{defaultRoute}
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
