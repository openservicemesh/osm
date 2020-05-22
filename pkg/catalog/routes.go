package catalog

import (
	"fmt"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"

	//HostHeader specifies the host header key
	HostHeader = "host"
)

// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
func (mc *MeshCatalog) ListTrafficPolicies(service service.NamespacedService) ([]trafficpolicy.TrafficTarget, error) {
	log.Info().Msgf("Listing Routes for service: %s", service)
	allRoutes, err := mc.getHTTPPathsPerRoute()
	if err != nil {
		log.Error().Err(err).Msgf("Could not get all routes")
		return nil, err
	}

	allTrafficPolicies, err := getTrafficPolicyPerRoute(mc, allRoutes, service)
	if err != nil {
		log.Error().Err(err).Msgf("Could not get all traffic policies")
		return nil, err
	}
	return allTrafficPolicies, nil
}

//GetWeightedClusterForService returns the weighted cluster for a given service
func (mc *MeshCatalog) GetWeightedClusterForService(nsService service.NamespacedService) (service.WeightedCluster, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	log.Info().Msgf("Finding weighted cluster for service %s", nsService)

	//retrieve the weighted clusters from traffic split
	servicesList := mc.meshSpec.ListServices()
	for _, activeService := range servicesList {
		if activeService.NamespacedService == nsService {
			return service.WeightedCluster{
				ClusterName: service.ClusterName(activeService.NamespacedService.String()),
				Weight:      activeService.Weight,
			}, nil
		}
	}

	//service not referenced in traffic split, assign a default weight of 100 to the service/cluster
	return service.WeightedCluster{
		ClusterName: service.ClusterName(nsService.String()),
		Weight:      constants.WildcardClusterWeight,
	}, nil
}

//GetDomainForService returns the domain name of a service
func (mc *MeshCatalog) GetDomainForService(nsService service.NamespacedService, routeHeaders map[string]string) (string, error) {
	log.Info().Msgf("Finding domain for service %s", nsService)
	var domain string

	//retrieve the domain name from traffic split
	servicesList := mc.meshSpec.ListServices()
	for _, activeService := range servicesList {
		if activeService.NamespacedService == nsService {
			return activeService.Domain, nil
		}
	}
	//service not referenced in traffic split, check if the traffic policy has the host header as a part of the route spec
	hostName, hostExists := routeHeaders[HostHeader]
	if hostExists {
		return hostName, nil
	}
	return domain, errDomainNotFoundForService
}

func (mc *MeshCatalog) getHTTPPathsPerRoute() (map[string]map[string]trafficpolicy.Route, error) {
	routePolicies := make(map[string]map[string]trafficpolicy.Route)
	for _, trafficSpecs := range mc.meshSpec.ListHTTPTrafficSpecs() {
		log.Debug().Msgf("Discovered TrafficSpec resource: %s/%s \n", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Matches == nil {
			log.Error().Msgf("TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}
		// since this method gets only specs related to HTTPRouteGroups added HTTPTraffic to the specKey by default
		specKey := fmt.Sprintf("%s/%s/%s", HTTPTraffic, trafficSpecs.Namespace, trafficSpecs.Name)
		routePolicies[specKey] = make(map[string]trafficpolicy.Route)
		for _, trafficSpecsMatches := range trafficSpecs.Matches {
			serviceRoute := trafficpolicy.Route{}
			serviceRoute.PathRegex = trafficSpecsMatches.PathRegex
			serviceRoute.Methods = trafficSpecsMatches.Methods
			serviceRoute.Headers = trafficSpecsMatches.Headers
			routePolicies[specKey][trafficSpecsMatches.Name] = serviceRoute
		}
	}
	log.Debug().Msgf("Constructed HTTP path routes: %+v", routePolicies)
	return routePolicies, nil
}

func getTrafficPolicyPerRoute(mc *MeshCatalog, routePolicies map[string]map[string]trafficpolicy.Route, nsService service.NamespacedService) ([]trafficpolicy.TrafficTarget, error) {
	var trafficPolicies []trafficpolicy.TrafficTarget
	for _, trafficTargets := range mc.meshSpec.ListTrafficTargets() {
		log.Debug().Msgf("Discovered TrafficTarget resource: %s/%s \n", trafficTargets.Namespace, trafficTargets.Name)
		if trafficTargets.Specs == nil || len(trafficTargets.Specs) == 0 {
			log.Error().Msgf("TrafficTarget %s/%s has no spec routes; Skipping...", trafficTargets.Namespace, trafficTargets.Name)
			continue
		}

		dstNamespacedServiceAcc := service.NamespacedServiceAccount{
			Namespace:      trafficTargets.Destination.Namespace,
			ServiceAccount: trafficTargets.Destination.Name,
		}
		destService, destErr := mc.GetServiceForServiceAccount(dstNamespacedServiceAcc)
		if destErr != nil {
			log.Error().Msgf("TrafficTarget %s/%s could not get destination services for service account %s", trafficTargets.Namespace, trafficTargets.Name, dstNamespacedServiceAcc.String())
			return nil, destErr
		}

		for _, trafficSources := range trafficTargets.Sources {
			namespacedServiceAccount := service.NamespacedServiceAccount{
				Namespace:      trafficSources.Namespace,
				ServiceAccount: trafficSources.Name,
			}

			srcServices, srcErr := mc.GetServiceForServiceAccount(namespacedServiceAccount)
			if srcErr != nil {
				log.Error().Msgf("TrafficTarget %s/%s could not get source services for service account %s", trafficTargets.Namespace, trafficTargets.Name, fmt.Sprintf("%s/%s", trafficSources.Namespace, trafficSources.Name))
				return nil, srcErr
			}
			policy := trafficpolicy.TrafficTarget{}
			policy.Name = trafficTargets.Name
			policy.Destination = trafficpolicy.TrafficResource{
				ServiceAccount: service.Account(trafficTargets.Destination.Name),
				Namespace:      trafficTargets.Destination.Namespace,
				Service:        destService}
			policy.Source = trafficpolicy.TrafficResource{
				ServiceAccount: service.Account(trafficSources.Name),
				Namespace:      trafficSources.Namespace,
				Service:        srcServices}

			for _, trafficTargetSpecs := range trafficTargets.Specs {
				if trafficTargetSpecs.Kind != HTTPTraffic {
					log.Error().Msgf("TrafficTarget %s/%s has Spec Kind %s which isn't supported for now; Skipping...", trafficTargets.Namespace, trafficTargets.Name, trafficTargetSpecs.Kind)
					continue
				}

				specKey := fmt.Sprintf("%s/%s/%s", trafficTargetSpecs.Kind, trafficTargets.Namespace, trafficTargetSpecs.Name)
				routePoliciesMatched, matchFound := routePolicies[specKey]
				if !matchFound {
					log.Error().Msgf("TrafficTarget %s/%s could not find a TrafficSpec %s", trafficTargets.Namespace, trafficTargets.Name, specKey)
					return nil, errNoTrafficSpecFoundForTrafficPolicy

				}
				if len(trafficTargetSpecs.Matches) == 0 {
					// no match name provided, so routes are build for all matches in traffic spec
					for _, routePolicy := range routePoliciesMatched {
						policy.Route = routePolicy
						// append a traffic policy only if it corresponds to the service
						if policy.Source.Service.Equals(nsService) || policy.Destination.Service.Equals(nsService) {
							trafficPolicies = append(trafficPolicies, policy)
						}
					}
				} else {
					// route is built only for the matche name specified in the policy
					for _, specMatchesName := range trafficTargetSpecs.Matches {
						routePolicy, matchFound := routePoliciesMatched[specMatchesName]
						if !matchFound {
							log.Error().Msgf("TrafficTarget %s/%s could not find a TrafficSpec %s with match name %s", trafficTargets.Namespace, trafficTargets.Name, specKey, specMatchesName)
							return nil, errNoTrafficSpecFoundForTrafficPolicy
						}
						policy.Route = routePolicy
						// append a traffic policy only if it corresponds to the service
						if policy.Source.Service.Equals(nsService) || policy.Destination.Service.Equals(nsService) {
							trafficPolicies = append(trafficPolicies, policy)
						}
					}
				}
			}
		}
	}

	log.Debug().Msgf("Constructed traffic policies: %+v", trafficPolicies)
	return trafficPolicies, nil
}
