package catalog

import (
	"fmt"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/envoy"
	"github.com/deislabs/smc/pkg/log/level"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"
)

// ListTrafficRoutes constructs a DiscoveryResponse with all routes the given Envoy proxy should be aware of.
func (sc *MeshCatalog) ListTrafficRoutes(clientID endpoint.NamespacedService) ([]endpoint.TrafficTargetPolicies, error) {
	glog.Info("[catalog] Listing Routes for client: ", clientID)
	allRoutes, err := sc.getHTTPPathsPerRoute()
	if err != nil {
		glog.Error("[catalog] Could not get all routes: ", err)
		return nil, err
	}

	allTrafficPolicies, err := getTrafficPolicyPerRoute(sc, allRoutes, clientID)
	if err != nil {
		glog.Error("[catalog] Could not get all traffic policies: ", err)
		return nil, err
	}
	return allTrafficPolicies, nil
}

func (sc *MeshCatalog) listServicesForServiceAccount(namespacedServiceAccount endpoint.NamespacedServiceAccount) ([]endpoint.NamespacedService, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	glog.Infof("[catalog] Listing services for service account: %s", namespacedServiceAccount)
	if _, found := sc.serviceAccountsCache[namespacedServiceAccount]; !found {
		sc.refreshCache()
	}
	var services []endpoint.NamespacedService
	var found bool
	if services, found = sc.serviceAccountsCache[namespacedServiceAccount]; !found {
		glog.Errorf("[catalog] Did not find any services for service account %s", namespacedServiceAccount)
		return nil, errNotFound
	}
	glog.Infof("[catalog] Found service account %s for service %s", servicesToString(services), namespacedServiceAccount)
	return services, nil
}

func (sc *MeshCatalog) listClustersForServices(services []endpoint.NamespacedService) []endpoint.WeightedCluster {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	glog.Infof("[catalog] Finding active clusters for services %v", services)
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
	glog.Infof("[catalog] Finding active services only %v", services)
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

func (sc *MeshCatalog) getHTTPPathsPerRoute() (map[string]endpoint.RoutePaths, error) {
	routes := make(map[string]endpoint.RoutePaths)
	for _, trafficSpecs := range sc.meshSpec.ListHTTPTrafficSpecs() {
		glog.V(level.Debug).Infof("[RDS][catalog] Discovered TrafficSpec resource: %s/%s \n", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Matches == nil {
			glog.Errorf("[catalog] TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}
		// since this method gets only spces related to HTTPRouteGroups added HTTPTRaffic to the specKey by default
		specKey := fmt.Sprintf("%s/%s/%s", HTTPTraffic, trafficSpecs.Namespace, trafficSpecs.Name)
		for _, trafficSpecsMatches := range trafficSpecs.Matches {
			serviceRoute := endpoint.RoutePaths{}
			serviceRoute.RoutePathRegex = trafficSpecsMatches.PathRegex
			serviceRoute.RouteMethods = trafficSpecsMatches.Methods
			routes[fmt.Sprintf("%s/%s", specKey, trafficSpecsMatches.Name)] = serviceRoute
		}
	}
	glog.V(level.Debug).Infof("[catalog] Constructed HTTP path routes: %+v", routes)
	return routes, nil
}

func getTrafficPolicyPerRoute(sc *MeshCatalog, routes map[string]endpoint.RoutePaths, clientID endpoint.NamespacedService) ([]endpoint.TrafficTargetPolicies, error) {
	var trafficPolicies []endpoint.TrafficTargetPolicies
	for _, trafficTargets := range sc.meshSpec.ListTrafficTargets() {
		glog.V(level.Debug).Infof("[catalog] Discovered TrafficTarget resource: %s/%s \n", trafficTargets.Namespace, trafficTargets.Name)
		if trafficTargets.Specs == nil || len(trafficTargets.Specs) == 0 {
			glog.Errorf("[catalog] TrafficTarget %s/%s has no spec routes; Skipping...", trafficTargets.Namespace, trafficTargets.Name)
			continue
		}

		dstNamespacedServiceAcc := endpoint.NamespacedServiceAccount{
			Namespace:      trafficTargets.Destination.Namespace,
			ServiceAccount: trafficTargets.Destination.Name,
		}
		destServices, destErr := sc.listServicesForServiceAccount(dstNamespacedServiceAcc)
		if destErr != nil {
			glog.Errorf("[catalog] TrafficSpec %s/%s could not get services for service account %s", trafficTargets.Namespace, trafficTargets.Name, dstNamespacedServiceAcc.String())
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
					glog.Errorf("[catalog] TrafficSpec %s/%s could not get services for service account %s", trafficTargets.Namespace, trafficTargets.Name, fmt.Sprintf("%s/%s", trafficSources.Namespace, trafficSources.Name))
					return nil, srcErr
				}
				trafficTargetPolicy := endpoint.TrafficTargetPolicies{}
				trafficTargetPolicy.PolicyName = trafficTargets.Name
				trafficTargetPolicy.Destination = endpoint.TrafficResource{
					ServiceAccount: endpoint.ServiceAccount(trafficTargets.Destination.Name),
					Namespace:      trafficTargets.Destination.Namespace,
					Services:       activeDestServices,
					Clusters:       destClusters}
				trafficTargetPolicy.Source = endpoint.TrafficResource{
					ServiceAccount: endpoint.ServiceAccount(trafficSources.Name),
					Namespace:      trafficSources.Namespace,
					Services:       srcServices,
					Clusters:       destClusters}

				for _, trafficTargetSpecs := range trafficTargets.Specs {
					if trafficTargetSpecs.Kind != HTTPTraffic {
						glog.Errorf("[catalog] TrafficTarget %s/%s has Spec Kind %s which isn't supported for now; Skipping...", trafficTargets.Namespace, trafficTargets.Name, trafficTargetSpecs.Kind)
						continue
					}
					trafficTargetPolicy.PolicyRoutePaths = []endpoint.RoutePaths{}

					for _, specMatches := range trafficTargetSpecs.Matches {
						routeKey := fmt.Sprintf("%s/%s/%s/%s", trafficTargetSpecs.Kind, trafficTargets.Namespace, trafficTargetSpecs.Name, specMatches)
						routePath := routes[routeKey]
						trafficTargetPolicy.PolicyRoutePaths = append(trafficTargetPolicy.PolicyRoutePaths, routePath)
					}
				}
				// append a traffic policy only if it corresponds to the clientID
				if envoy.Contains(clientID, trafficTargetPolicy.Source.Services) || envoy.Contains(clientID, trafficTargetPolicy.Destination.Services) {
					trafficPolicies = append(trafficPolicies, trafficTargetPolicy)
				}
			}
		}
	}
	glog.V(level.Debug).Infof("[catalog] Constructed traffic policies: %+v", trafficPolicies)
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
