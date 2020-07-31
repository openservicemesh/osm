package catalog

import (
	"fmt"
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	//HTTPTraffic specifies HTTP Traffic Policy
	HTTPTraffic = "HTTPRouteGroup"

	//HostHeaderKey specifies the host header key
	HostHeaderKey = "host"
)

// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
func (mc *MeshCatalog) ListTrafficPolicies(service service.NamespacedService) ([]trafficpolicy.TrafficTarget, error) {
	log.Info().Msgf("Listing traffic policies for service: %s", service)

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		// Build traffic policies from service discovery for allow-all policy
		trafficPolicies, err := mc.buildAllowAllTrafficPolicies(service)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to build allow-all traffic policy for service %s", service)
			return nil, err
		}
		return trafficPolicies, nil
	}

	// Build traffic policies from SMI
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

// This function returns the list of connected services.
// This is a bimodal function:
//   - it could list services that are allowed to connect to the given service (inbound)
//   - it could list services that the given service can connect to (outbound)
func (mc *MeshCatalog) getAllowedDirectionalServices(svc service.NamespacedService, directn direction) ([]service.NamespacedService, error) {
	allTrafficPolicies, err := mc.ListTrafficPolicies(svc)
	if err != nil {
		log.Error().Err(err).Msg("Failed listing traffic routes")
		return nil, err
	}

	allowedServicesSet := mapset.NewSet()

	for _, policy := range allTrafficPolicies {
		if directn == inbound {
			// we are looking for services that can connect to the given service
			if policy.Destination.Service.Equals(svc) {
				allowedServicesSet.Add(policy.Source.Service)
			}
		}

		if directn == outbound {
			// we are looking for services the given svc can connect to
			if policy.Source.Service.Equals(svc) {
				allowedServicesSet.Add(policy.Destination.Service)
			}
		}
	}

	// Convert the set of interfaces to a list of namespaced services
	var allowedServices []service.NamespacedService
	for svc := range allowedServicesSet.Iter() {
		allowedServices = append(allowedServices, svc.(service.NamespacedService))
	}

	msg := map[direction]string{
		inbound:  "Allowed inbound services for destination service %q: %+v",
		outbound: "Allowed outbound services from source %q: %+v",
	}[directn]

	log.Trace().Msgf(msg, svc, allowedServices)

	return allowedServices, nil
}

// ListAllowedInboundServices lists the inbound services allowed to connect to the given service.
func (mc *MeshCatalog) ListAllowedInboundServices(destinationService service.NamespacedService) ([]service.NamespacedService, error) {
	return mc.getAllowedDirectionalServices(destinationService, inbound)

}

// ListAllowedOutboundServices lists the services the given service is allowed outbound connections to.
func (mc *MeshCatalog) ListAllowedOutboundServices(sourceService service.NamespacedService) ([]service.NamespacedService, error) {
	return mc.getAllowedDirectionalServices(sourceService, outbound)
}

//GetWeightedClusterForService returns the weighted cluster for a given service
func (mc *MeshCatalog) GetWeightedClusterForService(svc service.NamespacedService) (service.WeightedCluster, error) {
	log.Trace().Msgf("Finding weighted cluster for service %s", svc)

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return getDefaultWeightedClusterForService(svc), nil
	}

	// Retrieve the weighted clusters from traffic split
	servicesList := mc.meshSpec.ListTrafficSplitServices()
	for _, activeService := range servicesList {
		if activeService.NamespacedService == svc {
			return service.WeightedCluster{
				ClusterName: service.ClusterName(activeService.NamespacedService.String()),
				Weight:      activeService.Weight,
			}, nil
		}
	}

	// Use a default weighted cluster as an SMI TrafficSplit policy is not defined for the service
	return getDefaultWeightedClusterForService(svc), nil
}

//GetDomainForService returns the domain name of a service
func (mc *MeshCatalog) GetDomainForService(nsService service.NamespacedService, routeHeaders map[string]string) (string, error) {
	log.Trace().Msgf("Finding domain for service %s", nsService)

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return getHostHeaderFromRouteHeaders(routeHeaders)
	}

	// Retrieve the domain name from traffic split
	servicesList := mc.meshSpec.ListTrafficSplitServices()
	for _, activeService := range servicesList {
		if activeService.NamespacedService == nsService {
			return activeService.Domain, nil
		}
	}

	// Use the augmented domains from k8s service since an
	// SMI TrafficSplit policy is not defined for the service

	hostHeader, err := getHostHeaderFromRouteHeaders(routeHeaders)
	if err != nil {
		log.Warn().Msgf("Found host header %s, but using service hostnames instead", hostHeader)
	}
	services, err := mc.meshSpec.ListServices()
	if err != nil {
		return "", err
	}
	var currentService *corev1.Service
	for _, service := range services {
		if service.Name == nsService.Service && service.Namespace == nsService.Namespace {
			currentService = service
			break
		}
	}

	if currentService == nil {
		log.Error().Err(errServiceNotFound).Msgf("Could not find %s", nsService)
		return "", errServiceNotFound
	}

	hostList := kubernetes.GetDomainsForService(currentService)
	host := strings.Join(hostList, ",")

	return host, nil
}

func (mc *MeshCatalog) getHTTPPathsPerRoute() (map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.Route, error) {
	routePolicies := make(map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.Route)
	for _, trafficSpecs := range mc.meshSpec.ListHTTPTrafficSpecs() {
		log.Debug().Msgf("Discovered TrafficSpec resource: %s/%s", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Matches == nil {
			log.Error().Msgf("TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}

		// since this method gets only specs related to HTTPRouteGroups added HTTPTraffic to the specKey by default
		specKey := mc.getTrafficSpecName(HTTPTraffic, trafficSpecs.Namespace, trafficSpecs.Name)
		routePolicies[specKey] = make(map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.Route)
		for _, trafficSpecsMatches := range trafficSpecs.Matches {
			serviceRoute := trafficpolicy.Route{}
			serviceRoute.PathRegex = trafficSpecsMatches.PathRegex
			serviceRoute.Methods = trafficSpecsMatches.Methods
			serviceRoute.Headers = trafficSpecsMatches.Headers
			if len(serviceRoute.Headers) != 0 {
				// When pathRegex and methods are not defined, the header filters are applied to any path and all HTTP methods
				if serviceRoute.PathRegex == "" {
					serviceRoute.PathRegex = constants.RegexMatchAll
				}
				if serviceRoute.Methods == nil {
					serviceRoute.Methods = []string{constants.WildcardHTTPMethod}
				}
			}
			routePolicies[specKey][trafficpolicy.TrafficSpecMatchName(trafficSpecsMatches.Name)] = serviceRoute
		}
	}
	log.Debug().Msgf("Constructed HTTP path routes: %+v", routePolicies)
	return routePolicies, nil
}

func (mc *MeshCatalog) getTrafficSpecName(trafficSpecKind string, trafficSpecNamespace string, trafficSpecName string) trafficpolicy.TrafficSpecName {
	specKey := fmt.Sprintf("%s/%s/%s", trafficSpecKind, trafficSpecNamespace, trafficSpecName)
	return trafficpolicy.TrafficSpecName(specKey)
}

func getTrafficPolicyPerRoute(mc *MeshCatalog, routePolicies map[trafficpolicy.TrafficSpecName]map[trafficpolicy.TrafficSpecMatchName]trafficpolicy.Route, nsService service.NamespacedService) ([]trafficpolicy.TrafficTarget, error) {
	var trafficPolicies []trafficpolicy.TrafficTarget
	for _, trafficTargets := range mc.meshSpec.ListTrafficTargets() {
		log.Debug().Msgf("Discovered TrafficTarget resource: %s/%s", trafficTargets.Namespace, trafficTargets.Name)
		if trafficTargets.Specs == nil || len(trafficTargets.Specs) == 0 {
			log.Error().Msgf("TrafficTarget %s/%s has no spec routes; Skipping...", trafficTargets.Namespace, trafficTargets.Name)
			continue
		}

		dstNamespacedServiceAcc := service.K8sServiceAccount{
			Namespace: trafficTargets.Destination.Namespace,
			Name:      trafficTargets.Destination.Name,
		}
		destService, destErr := mc.GetServiceForServiceAccount(dstNamespacedServiceAcc)
		if destErr != nil {
			log.Error().Msgf("TrafficTarget %s/%s could not get destination services for service account %s", trafficTargets.Namespace, trafficTargets.Name, dstNamespacedServiceAcc.String())
			return nil, destErr
		}

		for _, trafficSources := range trafficTargets.Sources {
			namespacedServiceAccount := service.K8sServiceAccount{
				Namespace: trafficSources.Namespace,
				Name:      trafficSources.Name,
			}

			srcServices, srcErr := mc.GetServiceForServiceAccount(namespacedServiceAccount)
			if srcErr != nil {
				log.Error().Msgf("TrafficTarget %s/%s could not get source services for service account %s", trafficTargets.Namespace, trafficTargets.Name, fmt.Sprintf("%s/%s", trafficSources.Namespace, trafficSources.Name))
				return nil, srcErr
			}
			policy := trafficpolicy.TrafficTarget{}
			policy.Name = trafficTargets.Name
			policy.Destination = trafficpolicy.TrafficResource{
				Namespace: trafficTargets.Destination.Namespace,
				Service:   destService}
			policy.Source = trafficpolicy.TrafficResource{
				Namespace: trafficSources.Namespace,
				Service:   srcServices}

			for _, trafficTargetSpecs := range trafficTargets.Specs {
				if trafficTargetSpecs.Kind != HTTPTraffic {
					log.Error().Msgf("TrafficTarget %s/%s has Spec Kind %s which isn't supported for now; Skipping...", trafficTargets.Namespace, trafficTargets.Name, trafficTargetSpecs.Kind)
					continue
				}

				specKey := mc.getTrafficSpecName(trafficTargetSpecs.Kind, trafficTargets.Namespace, trafficTargetSpecs.Name)
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
						routePolicy, matchFound := routePoliciesMatched[trafficpolicy.TrafficSpecMatchName(specMatchesName)]
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

func (mc *MeshCatalog) buildAllowAllTrafficPolicies(service service.NamespacedService) ([]trafficpolicy.TrafficTarget, error) {
	services, err := mc.meshSpec.ListServices()
	if err != nil {
		log.Error().Err(err).Msgf("Error building traffic policies for service %s", service)
		return nil, err
	}

	var trafficTargets []trafficpolicy.TrafficTarget
	for _, source := range services {
		for _, destination := range services {
			if reflect.DeepEqual(source, destination) {
				continue
			}
			allowTrafficTarget := mc.buildAllowPolicyForSourceToDest(source, destination)
			trafficTargets = append(trafficTargets, allowTrafficTarget)
		}
	}
	log.Trace().Msgf("all traffic policies: %v", trafficTargets)
	return trafficTargets, nil
}

func (mc *MeshCatalog) buildAllowPolicyForSourceToDest(source *corev1.Service, destination *corev1.Service) trafficpolicy.TrafficTarget {
	sourceTrafficResource := trafficpolicy.TrafficResource{
		Namespace: source.Namespace,
		Service: service.NamespacedService{
			Namespace: source.Namespace,
			Service:   source.Name,
		},
	}
	destinationTrafficResource := trafficpolicy.TrafficResource{
		Namespace: destination.Namespace,
		Service: service.NamespacedService{
			Namespace: destination.Namespace,
			Service:   destination.Name,
		},
	}

	serviceDomains := kubernetes.GetDomainsForService(destination)
	hostHeader := map[string]string{HostHeaderKey: strings.Join(serviceDomains[:], ",")}
	allowAllRoute := trafficpolicy.Route{
		PathRegex: constants.RegexMatchAll,
		Methods:   []string{constants.WildcardHTTPMethod},
		Headers:   hostHeader,
	}
	return trafficpolicy.TrafficTarget{
		Name:        fmt.Sprintf("%s->%s", source, destination),
		Destination: destinationTrafficResource,
		Source:      sourceTrafficResource,
		Route:       allowAllRoute,
	}
}

func getHostHeaderFromRouteHeaders(routeHeaders map[string]string) (string, error) {
	hostName, hostExists := routeHeaders[HostHeaderKey]
	if hostExists {
		return hostName, nil
	}
	return "", errDomainNotFoundForService
}

func getDefaultWeightedClusterForService(nsService service.NamespacedService) service.WeightedCluster {
	return service.WeightedCluster{
		ClusterName: service.ClusterName(nsService.String()),
		Weight:      constants.ClusterWeightAcceptAll,
	}
}
