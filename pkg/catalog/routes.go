package catalog

import (
	"fmt"

	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"
)

// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
func (sc *MeshCatalog) ListTrafficPolicies(clientID endpoint.NamespacedService) ([]endpoint.TrafficPolicy, error) {
	log.Info().Msgf("Listing Routes for client: %s", clientID)
	allRoutes, err := sc.getHTTPPathsPerRoute()
	if err != nil {
		log.Error().Err(err).Msgf("Could not get all routes")
		return nil, err
	}

	allTrafficPolicies, err := getTrafficPolicyPerRoute(sc, allRoutes, clientID)
	if err != nil {
		log.Error().Err(err).Msgf("Could not get all traffic policies")
		return nil, err
	}
	return allTrafficPolicies, nil
}

func (sc *MeshCatalog) listServicesForServiceAccount(namespacedServiceAccount endpoint.NamespacedServiceAccount) ([]endpoint.NamespacedService, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	log.Info().Msgf("Listing services for service account: %s", namespacedServiceAccount)
	if _, found := sc.serviceAccountsCache[namespacedServiceAccount]; !found {
		sc.refreshCache()
	}
	var services []endpoint.NamespacedService
	var found bool
	if services, found = sc.serviceAccountsCache[namespacedServiceAccount]; !found {
		log.Error().Msgf("Did not find any services for service account %s", namespacedServiceAccount)
		return nil, errServiceNotFound
	}
	log.Info().Msgf("Found service account %s for service %s", servicesToString(services), namespacedServiceAccount)
	return services, nil
}

func (sc *MeshCatalog) listClustersForServices(services []endpoint.NamespacedService) []endpoint.WeightedCluster {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	log.Info().Msgf("Finding active clusters for services %v", services)
	var clusters []endpoint.WeightedCluster
	for _, service := range services {
		for activeService := range sc.servicesCache {
			if activeService.ServiceName == service {
				weightedCluster := endpoint.WeightedCluster{
					ClusterName: endpoint.ClusterName(activeService.ServiceName.String()),
					Weight:      activeService.Weight,
				}
				clusters = append(clusters, weightedCluster)
			}
		}
	}
	return uniqueClusters(clusters)
}

func (sc *MeshCatalog) getActiveServices(services []endpoint.NamespacedService) []endpoint.NamespacedService {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	log.Info().Msgf("Finding active services only %v", services)
	var activeServices []endpoint.NamespacedService
	for _, service := range services {
		for activeService := range sc.servicesCache {
			if activeService.ServiceName == service {
				activeServices = append(activeServices, service)
			}
		}
	}
	return uniqueServices(activeServices)
}

func (sc *MeshCatalog) getHTTPPathsPerRoute() (map[string]endpoint.RoutePolicy, error) {
	routePolicies := make(map[string]endpoint.RoutePolicy)
	for _, trafficSpecs := range sc.meshSpec.ListHTTPTrafficSpecs() {
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

func getTrafficPolicyPerRoute(sc *MeshCatalog, routePolicies map[string]endpoint.RoutePolicy, clientID endpoint.NamespacedService) ([]endpoint.TrafficPolicy, error) {
	var trafficPolicies []endpoint.TrafficPolicy
	for _, trafficTargets := range sc.meshSpec.ListTrafficTargets() {
		log.Debug().Msgf("Discovered TrafficTarget resource: %s/%s \n", trafficTargets.Namespace, trafficTargets.Name)
		if trafficTargets.Specs == nil || len(trafficTargets.Specs) == 0 {
			log.Error().Msgf("TrafficTarget %s/%s has no spec routes; Skipping...", trafficTargets.Namespace, trafficTargets.Name)
			continue
		}

		dstNamespacedServiceAcc := endpoint.NamespacedServiceAccount{
			Namespace:      trafficTargets.Destination.Namespace,
			ServiceAccount: trafficTargets.Destination.Name,
		}
		destServices, destErr := sc.listServicesForServiceAccount(dstNamespacedServiceAcc)
		if destErr != nil {
			log.Error().Msgf("TrafficSpec %s/%s could not get services for service account %s", trafficTargets.Namespace, trafficTargets.Name, dstNamespacedServiceAcc.String())
			return nil, destErr
		}

		activeDestServices := sc.getActiveServices(destServices)
		destClusters := sc.listClustersForServices(activeDestServices)
		//Routes are configured only if destination cluster/s exist i.e traffic split (endpoints) is setup
		if len(destClusters) > 0 {

			for _, trafficSources := range trafficTargets.Sources {
				namespacedServiceAccount := endpoint.NamespacedServiceAccount{
					Namespace:      trafficSources.Namespace,
					ServiceAccount: trafficSources.Name,
				}
				srcServices, srcErr := sc.listServicesForServiceAccount(namespacedServiceAccount)
				if srcErr != nil {
					log.Error().Msgf("TrafficSpec %s/%s could not get services for service account %s", trafficTargets.Namespace, trafficTargets.Name, fmt.Sprintf("%s/%s", trafficSources.Namespace, trafficSources.Name))
					return nil, srcErr
				}
				trafficPolicy := endpoint.TrafficPolicy{}
				trafficPolicy.PolicyName = trafficTargets.Name
				trafficPolicy.Destination = endpoint.TrafficResource{
					ServiceAccount: endpoint.ServiceAccount(trafficTargets.Destination.Name),
					Namespace:      trafficTargets.Destination.Namespace,
					Services:       activeDestServices,
					Clusters:       destClusters}
				trafficPolicy.Source = endpoint.TrafficResource{
					ServiceAccount: endpoint.ServiceAccount(trafficSources.Name),
					Namespace:      trafficSources.Namespace,
					Services:       srcServices,
					Clusters:       destClusters}

				for _, trafficTargetSpecs := range trafficTargets.Specs {
					if trafficTargetSpecs.Kind != HTTPTraffic {
						log.Error().Msgf("TrafficTarget %s/%s has Spec Kind %s which isn't supported for now; Skipping...", trafficTargets.Namespace, trafficTargets.Name, trafficTargetSpecs.Kind)
						continue
					}
					trafficPolicy.PolicyRoutePaths = []endpoint.RoutePolicy{}

					for _, specMatches := range trafficTargetSpecs.Matches {
						routeKey := fmt.Sprintf("%s/%s/%s/%s", trafficTargetSpecs.Kind, trafficTargets.Namespace, trafficTargetSpecs.Name, specMatches)
						routePath := routePolicies[routeKey]
						trafficPolicy.PolicyRoutePaths = append(trafficPolicy.PolicyRoutePaths, routePath)
					}
				}
				// append a traffic policy only if it corresponds to the clientID
				if envoy.Contains(clientID, trafficPolicy.Source.Services) || envoy.Contains(clientID, trafficPolicy.Destination.Services) {
					trafficPolicies = append(trafficPolicies, trafficPolicy)
				}
			}
		}
	}
	log.Debug().Msgf("Constructed traffic policies: %+v", trafficPolicies)
	return trafficPolicies, nil
}

func servicesToString(services []endpoint.NamespacedService) []string {
	var svcs []string
	for _, svc := range services {
		svcs = append(svcs, fmt.Sprintf("%v", svc.String()))
	}
	return svcs
}

func uniqueServices(slice []endpoint.NamespacedService) []endpoint.NamespacedService {
	keys := make(map[endpoint.NamespacedService]interface{})
	uniqueSlice := []endpoint.NamespacedService{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = nil
			uniqueSlice = append(uniqueSlice, entry)
		}
	}
	return uniqueSlice
}

func uniqueClusters(slice []endpoint.WeightedCluster) []endpoint.WeightedCluster {
	keys := make(map[endpoint.WeightedCluster]interface{})
	uniqueSlice := []endpoint.WeightedCluster{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = nil
			uniqueSlice = append(uniqueSlice, entry)
		}
	}
	return uniqueSlice
}
