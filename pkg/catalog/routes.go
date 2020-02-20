package catalog

import (
	"fmt"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log/level"
	"github.com/deislabs/smc/pkg/smi"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"
)

// ListTrafficRoutes constructs a DiscoveryResponse with all routes the given Envoy proxy should be aware of.
func (sc *MeshCatalog) ListTrafficRoutes(clientID smi.ClientIdentity) ([]endpoint.TrafficTargetPolicies, error) {
	glog.Info("[catalog] Listing Routes for client: ", clientID)
	allRoutes, err := sc.getHTTPPathsPerRoute()
	if err != nil {
		glog.Error("[catalog] Could not get all routes: ", err)
		return nil, err
	}

	allTrafficPolicies, err := getTrafficPolicyPerRoute(sc, allRoutes)
	if err != nil {
		glog.Error("[catalog] Could not get all traffic policies: ", err)
		return nil, err
	}
	return allTrafficPolicies, nil
}

func (sc *MeshCatalog) listServicesForServiceAccount(namespacedServiceAccount endpoint.ServiceAccount) ([]endpoint.ServiceName, error) {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	glog.Infof("[catalog] Listing services for service account: %s", namespacedServiceAccount)
	if _, found := sc.serviceAccountsCache[namespacedServiceAccount]; !found {
		sc.refreshCache()
	}
	var services []endpoint.ServiceName
	var found bool
	if services, found = sc.serviceAccountsCache[namespacedServiceAccount]; !found {
		glog.Errorf("[catalog] Did not find any services for service account %s", namespacedServiceAccount)
		return nil, errNotFound
	}
	glog.Infof("[catalog] Found service account %s for service %s", servicesToString(services), namespacedServiceAccount)
	return services, nil
}

func (sc *MeshCatalog) listAggregatedClusters(services []endpoint.ServiceName) []endpoint.ServiceName {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	glog.Infof("[catalog] Listing aggregated clusters for services : %v", services)
	var clusters []endpoint.ServiceName
	for _, service := range services {
		clusterFound := false
		for key, values := range sc.targetServicesCache {
			for _, value := range values {
				if value == service {
					glog.V(level.Trace).Infof("[catalog] Found aggregated cluster %s for service %s", key, value)
					clusterFound = true
					clusters = append(clusters, key)
				}
			}
		}
		if !clusterFound {
			glog.V(level.Trace).Infof("[catalog] No aggregated cluster for service %s ", service)
			clusters = append(clusters, service)
		}
	}
	return uniques(clusters)
}

func (sc *MeshCatalog) listClustersForServices(services []endpoint.ServiceName) []endpoint.ServiceName {
	// TODO(draychev): split namespace from the service name -- for non-K8s services
	glog.Infof("[catalog] Finding active clusters for services %v", services)
	var clusters []endpoint.ServiceName
	for _, service := range services {
		for _, values := range sc.targetServicesCache {
			for _, value := range values {
				if value == service {
					clusters = append(clusters, service)
				}
			}
		}
	}
	return uniques(clusters)
}

func (sc *MeshCatalog) getHTTPPathsPerRoute() (map[string]endpoint.RoutePaths, error) {
	routes := make(map[string]endpoint.RoutePaths)
	for _, trafficSpecs := range sc.meshSpec.ListHTTPTrafficSpecs() {
		glog.V(level.Debug).Infof("[RDS][catalog] Discovered TrafficSpec resource: %s/%s \n", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Matches == nil {
			glog.Errorf("[RDS][catalog] TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}
		spec := fmt.Sprintf("%s/%s/%s", trafficSpecs.Name, trafficSpecs.Kind, trafficSpecs.Namespace)
		//todo (snchh) : no mapping yet for route methods (GET,POST) in the envoy configuration
		for _, trafficSpecsMatches := range trafficSpecs.Matches {
			serviceRoute := endpoint.RoutePaths{}
			serviceRoute.RoutePathRegex = trafficSpecsMatches.PathRegex
			serviceRoute.RouteMethods = trafficSpecsMatches.Methods
			routes[fmt.Sprintf("%s/%s", spec, trafficSpecsMatches.Name)] = serviceRoute
		}
	}
	glog.V(level.Debug).Infof("[catalog] Constructed HTTP path routes: %+v", routes)
	return routes, nil
}

func getTrafficPolicyPerRoute(sc *MeshCatalog, routes map[string]endpoint.RoutePaths) ([]endpoint.TrafficTargetPolicies, error) {
	var trafficPolicies []endpoint.TrafficTargetPolicies
	for _, trafficTargets := range sc.meshSpec.ListTrafficTargets() {
		glog.V(level.Debug).Infof("[RDS][catalog] Discovered TrafficTarget resource: %s/%s \n", trafficTargets.Namespace, trafficTargets.Name)
		if trafficTargets.Specs == nil || len(trafficTargets.Specs) == 0 {
			glog.Errorf("[RDS][catalog] TrafficTarget %s/%s has no spec routes; Skipping...", trafficTargets.Namespace, trafficTargets.Name)
			continue
		}

		destServices, destErr := sc.listServicesForServiceAccount(endpoint.ServiceAccount(fmt.Sprintf("%s/%s", trafficTargets.Destination.Namespace, trafficTargets.Destination.Name)))
		if destErr != nil {
			glog.Errorf("[RDS][catalog] TrafficSpec %s/%s could not get services for service account %s", trafficTargets.Namespace, trafficTargets.Name, fmt.Sprintf("%s/%s", trafficTargets.Destination.Namespace, trafficTargets.Destination.Name))
			return nil, destErr
		}

		for _, trafficSources := range trafficTargets.Sources {
			srcServices, srcErr := sc.listServicesForServiceAccount(endpoint.ServiceAccount(fmt.Sprintf("%s/%s", trafficSources.Namespace, trafficSources.Name)))
			if srcErr != nil {
				glog.Errorf("[RDS][catalog] TrafficSpec %s/%s could not get services for service account %s", trafficTargets.Namespace, trafficTargets.Name, fmt.Sprintf("%s/%s", trafficSources.Namespace, trafficSources.Name))
				return nil, srcErr
			}
			trafficTargetPolicy := endpoint.TrafficTargetPolicies{}
			trafficTargetPolicy.PolicyName = trafficTargets.Name
			trafficTargetPolicy.Destination = endpoint.TrafficResource{
				ServiceAccount: endpoint.ServiceAccount(trafficTargets.Destination.Name),
				Namespace:      trafficTargets.Destination.Namespace,
				Services:       destServices,
				Clusters:       sc.listClustersForServices(destServices)}
			trafficTargetPolicy.Source = endpoint.TrafficResource{
				ServiceAccount: endpoint.ServiceAccount(trafficSources.Name),
				Namespace:      trafficSources.Namespace,
				Services:       srcServices,
				Clusters:       sc.listAggregatedClusters(trafficTargetPolicy.Destination.Clusters)}

			for _, trafficTargetSpecs := range trafficTargets.Specs {
				if trafficTargetSpecs.Kind != HTTPTraffic {
					glog.Errorf("[RDS][catalog] TrafficTarget %s/%s has Spec Kind %s which isn't supported for now; Skipping...", trafficTargets.Namespace, trafficTargets.Name, trafficTargetSpecs.Kind)
					continue
				}
				trafficTargetPolicy.PolicyRoutePaths = []endpoint.RoutePaths{}

				for _, specMatches := range trafficTargetSpecs.Matches {
					routePath := routes[fmt.Sprintf("%s/%s/%s/%s", trafficTargetSpecs.Name, trafficTargetSpecs.Kind, trafficTargets.Namespace, specMatches)]
					trafficTargetPolicy.PolicyRoutePaths = append(trafficTargetPolicy.PolicyRoutePaths, routePath)
				}
			}
			trafficPolicies = append(trafficPolicies, trafficTargetPolicy)
		}
	}

	glog.V(level.Debug).Infof("[catalog] Constructed traffic routes: %+v", trafficPolicies)
	return trafficPolicies, nil
}

func servicesToString(services []endpoint.ServiceName) []string {
	var svcs []string
	for _, svc := range services {
		svcs = append(svcs, fmt.Sprintf("%v", svc))
	}
	return svcs
}

func uniques(slice []endpoint.ServiceName) []endpoint.ServiceName {
	keys := make(map[endpoint.ServiceName]interface{})
	uniqueSlice := []endpoint.ServiceName{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = nil
			uniqueSlice = append(uniqueSlice, entry)
		}
	}
	return uniqueSlice
}
