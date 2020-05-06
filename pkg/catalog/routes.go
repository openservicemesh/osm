package catalog

import (
	"fmt"

	set "github.com/deckarep/golang-set"

	"github.com/open-service-mesh/osm/pkg/endpoint"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"
)

// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
func (mc *MeshCatalog) ListTrafficPolicies(service endpoint.NamespacedService) ([]endpoint.TrafficPolicy, error) {
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

func (mc *MeshCatalog) listServicesForServiceAccount(namespacedServiceAccount endpoint.NamespacedServiceAccount) (set.Set, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	log.Info().Msgf("Listing services for ServiceAccount: %s", namespacedServiceAccount)
	if _, found := mc.serviceAccountToServicesCache[namespacedServiceAccount]; !found {
		mc.refreshCache()
	}
	services, found := mc.serviceAccountToServicesCache[namespacedServiceAccount]
	if !found {
		log.Error().Msgf("Did not find any services for ServiceAccount: %s", namespacedServiceAccount)
		return nil, errServiceNotFound
	}
	log.Info().Msgf("Found services %v for ServiceAccount: %s", services, namespacedServiceAccount)
	servicesMap := set.NewSet()
	for _, service := range services {
		servicesMap.Add(service)
	}
	return servicesMap, nil
}

//GetWeightedClusterForService returns the weighted cluster for a given service
func (mc *MeshCatalog) GetWeightedClusterForService(service endpoint.NamespacedService) (endpoint.WeightedCluster, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	log.Info().Msgf("Finding weighted cluster for service %s", service)
	for activeService := range mc.servicesCache {
		if activeService.ServiceName == service {
			return endpoint.WeightedCluster{
				ClusterName: endpoint.ClusterName(activeService.ServiceName.String()),
				Weight:      activeService.Weight,
			}, nil
		}
	}
	log.Error().Msgf("Did not find WeightedCluster for service %q", service)
	return endpoint.WeightedCluster{}, errServiceNotFound
}

//GetDomainForService returns the domain name of a service
func (mc *MeshCatalog) GetDomainForService(service endpoint.NamespacedService) (string, error) {
	log.Info().Msgf("Finding domain for service %s", service)
	var domain string
	for activeService := range mc.servicesCache {
		if activeService.ServiceName == service {
			return activeService.Domain, nil
		}
	}
	return domain, errServiceNotFound
}

func (mc *MeshCatalog) getActiveServices(services set.Set) set.Set {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	log.Info().Msgf("Finding active services only %v", services)
	activeServices := set.NewSet()
	for service := range mc.servicesCache {
		activeServices.Add(service.ServiceName)
	}
	return activeServices.Intersect(services)
}

func (mc *MeshCatalog) getHTTPPathsPerRoute() (map[string]endpoint.RoutePolicy, error) {
	routePolicies := make(map[string]endpoint.RoutePolicy)
	for _, trafficSpecs := range mc.meshSpec.ListHTTPTrafficSpecs() {
		log.Debug().Msgf("Discovered TrafficSpec resource: %s/%s \n", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Matches == nil {
			log.Error().Msgf("TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}
		// since this method gets only specs related to HTTPRouteGroups added HTTPTraffic to the specKey by default
		specKey := fmt.Sprintf("%s/%s/%s", HTTPTraffic, trafficSpecs.Namespace, trafficSpecs.Name)
		for _, trafficSpecsMatches := range trafficSpecs.Matches {
			serviceRoute := endpoint.RoutePolicy{}
			serviceRoute.PathRegex = trafficSpecsMatches.PathRegex
			serviceRoute.Methods = trafficSpecsMatches.Methods
			routePolicies[fmt.Sprintf("%s/%s", specKey, trafficSpecsMatches.Name)] = serviceRoute
		}
	}
	log.Debug().Msgf("Constructed HTTP path routes: %+v", routePolicies)
	return routePolicies, nil
}

func getTrafficPolicyPerRoute(mc *MeshCatalog, routePolicies map[string]endpoint.RoutePolicy, service endpoint.NamespacedService) ([]endpoint.TrafficPolicy, error) {
	var trafficPolicies []endpoint.TrafficPolicy
	for _, trafficTargets := range mc.meshSpec.ListTrafficTargets() {
		log.Debug().Msgf("Discovered TrafficTarget resource: %s/%s \n", trafficTargets.Namespace, trafficTargets.Name)
		if trafficTargets.Specs == nil || len(trafficTargets.Specs) == 0 {
			log.Error().Msgf("TrafficTarget %s/%s has no spec routes; Skipping...", trafficTargets.Namespace, trafficTargets.Name)
			continue
		}

		dstNamespacedServiceAcc := endpoint.NamespacedServiceAccount{
			Namespace:      trafficTargets.Destination.Namespace,
			ServiceAccount: trafficTargets.Destination.Name,
		}
		destServices, destErr := mc.listServicesForServiceAccount(dstNamespacedServiceAcc)
		if destErr != nil {
			log.Error().Msgf("TrafficSpec %s/%s could not get destination services for service account %s", trafficTargets.Namespace, trafficTargets.Name, dstNamespacedServiceAcc.String())
			return nil, destErr
		}

		activeDestServices := mc.getActiveServices(destServices)
		//Routes are configured only if destination services exist i.e traffic split (endpoints) is setup
		if activeDestServices.Cardinality() == 0 || activeDestServices == nil {
			continue
		}
		for _, trafficSources := range trafficTargets.Sources {
			namespacedServiceAccount := endpoint.NamespacedServiceAccount{
				Namespace:      trafficSources.Namespace,
				ServiceAccount: trafficSources.Name,
			}

			srcServices, srcErr := mc.listServicesForServiceAccount(namespacedServiceAccount)
			if srcErr != nil {
				log.Error().Msgf("TrafficSpec %s/%s could not get source services for service account %s", trafficTargets.Namespace, trafficTargets.Name, fmt.Sprintf("%s/%s", trafficSources.Namespace, trafficSources.Name))
				return nil, srcErr
			}
			trafficPolicy := endpoint.TrafficPolicy{}
			trafficPolicy.PolicyName = trafficTargets.Name
			trafficPolicy.Destination = endpoint.TrafficPolicyResource{
				ServiceAccount: endpoint.ServiceAccount(trafficTargets.Destination.Name),
				Namespace:      trafficTargets.Destination.Namespace,
				Services:       activeDestServices}
			trafficPolicy.Source = endpoint.TrafficPolicyResource{
				ServiceAccount: endpoint.ServiceAccount(trafficSources.Name),
				Namespace:      trafficSources.Namespace,
				Services:       srcServices}

			for _, trafficTargetSpecs := range trafficTargets.Specs {
				if trafficTargetSpecs.Kind != HTTPTraffic {
					log.Error().Msgf("TrafficTarget %s/%s has Spec Kind %s which isn't supported for now; Skipping...", trafficTargets.Namespace, trafficTargets.Name, trafficTargetSpecs.Kind)
					continue
				}
				trafficPolicy.RoutePolicies = []endpoint.RoutePolicy{}

				for _, specMatches := range trafficTargetSpecs.Matches {
					routeKey := fmt.Sprintf("%s/%s/%s/%s", trafficTargetSpecs.Kind, trafficTargets.Namespace, trafficTargetSpecs.Name, specMatches)
					routePolicy := routePolicies[routeKey]
					trafficPolicy.RoutePolicies = append(trafficPolicy.RoutePolicies, routePolicy)
				}
			}
			// append a traffic policy only if it corresponds to the service
			if trafficPolicy.Source.Services.Contains(service) || trafficPolicy.Destination.Services.Contains(service) {
				trafficPolicies = append(trafficPolicies, trafficPolicy)
			}
		}
	}

	log.Debug().Msgf("Constructed traffic policies: %+v", trafficPolicies)
	return trafficPolicies, nil
}
