package catalog

import (
	"fmt"

	set "github.com/deckarep/golang-set"

	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/trafficpolicy"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"
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

func (mc *MeshCatalog) listServicesForServiceAccount(namespacedServiceAccount service.NamespacedServiceAccount) (set.Set, error) {
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
func (mc *MeshCatalog) GetWeightedClusterForService(nsService service.NamespacedService) (service.WeightedCluster, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	log.Info().Msgf("Finding weighted cluster for service %s", nsService)
	servicesList := mc.meshSpec.ListServices()
	for _, activeService := range servicesList {
		if activeService.ServiceName == nsService {
			return service.WeightedCluster{
				ClusterName: service.ClusterName(activeService.ServiceName.String()),
				Weight:      activeService.Weight,
			}, nil
		}
	}
	log.Error().Msgf("Did not find WeightedCluster for service %q", nsService)
	return service.WeightedCluster{}, errServiceNotFound
}

//GetDomainForService returns the domain name of a service
func (mc *MeshCatalog) GetDomainForService(nsService service.NamespacedService) (string, error) {
	log.Info().Msgf("Finding domain for service %s", nsService)
	var domain string
	servicesList := mc.meshSpec.ListServices()
	for _, activeService := range servicesList {
		if activeService.ServiceName == nsService {
			return activeService.Domain, nil
		}
	}
	return domain, errServiceNotFound
}

func (mc *MeshCatalog) getActiveServices(services set.Set) set.Set {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	log.Info().Msgf("Finding active services only %v", services)
	activeServices := set.NewSet()
	servicesList := mc.meshSpec.ListServices()
	for _, service := range servicesList {
		activeServices.Add(service.ServiceName)
	}
	return activeServices.Intersect(services)
}

func (mc *MeshCatalog) getHTTPPathsPerRoute() (map[string]trafficpolicy.Route, error) {
	routePolicies := make(map[string]trafficpolicy.Route)
	for _, trafficSpecs := range mc.meshSpec.ListHTTPTrafficSpecs() {
		log.Debug().Msgf("Discovered TrafficSpec resource: %s/%s \n", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Matches == nil {
			log.Error().Msgf("TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}
		// since this method gets only specs related to HTTPRouteGroups added HTTPTraffic to the specKey by default
		specKey := fmt.Sprintf("%s/%s/%s", HTTPTraffic, trafficSpecs.Namespace, trafficSpecs.Name)
		for _, trafficSpecsMatches := range trafficSpecs.Matches {
			serviceRoute := trafficpolicy.Route{}
			serviceRoute.PathRegex = trafficSpecsMatches.PathRegex
			serviceRoute.Methods = trafficSpecsMatches.Methods
			routePolicies[fmt.Sprintf("%s/%s", specKey, trafficSpecsMatches.Name)] = serviceRoute
		}
	}
	log.Debug().Msgf("Constructed HTTP path routes: %+v", routePolicies)
	return routePolicies, nil
}

func getTrafficPolicyPerRoute(mc *MeshCatalog, routePolicies map[string]trafficpolicy.Route, nsService service.NamespacedService) ([]trafficpolicy.TrafficTarget, error) {
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
			namespacedServiceAccount := service.NamespacedServiceAccount{
				Namespace:      trafficSources.Namespace,
				ServiceAccount: trafficSources.Name,
			}

			srcServices, srcErr := mc.listServicesForServiceAccount(namespacedServiceAccount)
			if srcErr != nil {
				log.Error().Msgf("TrafficSpec %s/%s could not get source services for service account %s", trafficTargets.Namespace, trafficTargets.Name, fmt.Sprintf("%s/%s", trafficSources.Namespace, trafficSources.Name))
				return nil, srcErr
			}
			trafficPolicy := trafficpolicy.TrafficTarget{}
			trafficPolicy.Name = trafficTargets.Name
			trafficPolicy.Destination = trafficpolicy.TrafficResource{
				ServiceAccount: service.Account(trafficTargets.Destination.Name),
				Namespace:      trafficTargets.Destination.Namespace,
				Services:       activeDestServices}
			trafficPolicy.Source = trafficpolicy.TrafficResource{
				ServiceAccount: service.Account(trafficSources.Name),
				Namespace:      trafficSources.Namespace,
				Services:       srcServices}

			for _, trafficTargetSpecs := range trafficTargets.Specs {
				if trafficTargetSpecs.Kind != HTTPTraffic {
					log.Error().Msgf("TrafficTarget %s/%s has Spec Kind %s which isn't supported for now; Skipping...", trafficTargets.Namespace, trafficTargets.Name, trafficTargetSpecs.Kind)
					continue
				}
				trafficPolicy.Routes = []trafficpolicy.Route{}

				for _, specMatches := range trafficTargetSpecs.Matches {
					routeKey := fmt.Sprintf("%s/%s/%s/%s", trafficTargetSpecs.Kind, trafficTargets.Namespace, trafficTargetSpecs.Name, specMatches)
					routePolicy := routePolicies[routeKey]
					trafficPolicy.Routes = append(trafficPolicy.Routes, routePolicy)
				}
			}
			// append a traffic policy only if it corresponds to the service
			if trafficPolicy.Source.Services.Contains(nsService) || trafficPolicy.Destination.Services.Contains(nsService) {
				trafficPolicies = append(trafficPolicies, trafficPolicy)
			}
		}
	}

	log.Debug().Msgf("Constructed traffic policies: %+v", trafficPolicies)
	return trafficPolicies, nil
}
