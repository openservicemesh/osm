package catalog

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// ListOutboundTrafficPolicies returns all outbound traffic policies
// 1. from service discovery for permissive mode
// 2. for the given service account from SMI Traffic Target and Traffic Split
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListOutboundTrafficPolicies(downstreamIdentity identity.ServiceIdentity) []*trafficpolicy.OutboundTrafficPolicy {
	downstreamServiceAccount := downstreamIdentity.ToK8sServiceAccount()
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		var outboundPolicies []*trafficpolicy.OutboundTrafficPolicy
		mergedPolicies := trafficpolicy.MergeOutboundPolicies(DisallowPartialHostnamesMatch, outboundPolicies, mc.buildOutboundPermissiveModePolicies(downstreamServiceAccount.Namespace)...)
		outboundPolicies = mergedPolicies
		return outboundPolicies
	}

	outbound := mc.listOutboundPoliciesForTrafficTargets(downstreamIdentity)
	outboundPoliciesFromSplits := mc.listOutboundTrafficPoliciesForTrafficSplits(downstreamServiceAccount.Namespace)
	outbound = trafficpolicy.MergeOutboundPolicies(AllowPartialHostnamesMatch, outbound, outboundPoliciesFromSplits...)

	return outbound
}

// listOutboundPoliciesForTrafficTargets loops through all SMI Traffic Target resources and returns outbound traffic policies
// when the given service account matches a source in the Traffic Target resource
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) listOutboundPoliciesForTrafficTargets(downstreamIdentity identity.ServiceIdentity) []*trafficpolicy.OutboundTrafficPolicy {
	downstreamServiceAccount := downstreamIdentity.ToK8sServiceAccount()
	var outboundPolicies []*trafficpolicy.OutboundTrafficPolicy

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !isValidTrafficTarget(t) {
			continue
		}

		for _, source := range t.Spec.Sources {
			// TODO(draychev): must check for the correct type of ServiceIdentity as well
			if source.Name == downstreamServiceAccount.Name && source.Namespace == downstreamServiceAccount.Namespace { // found outbound
				mergedPolicies := trafficpolicy.MergeOutboundPolicies(AllowPartialHostnamesMatch, outboundPolicies, mc.buildOutboundPolicies(downstreamIdentity, t)...)
				outboundPolicies = mergedPolicies
				break
			}
		}
	}
	return outboundPolicies
}

func (mc *MeshCatalog) listOutboundTrafficPoliciesForTrafficSplits(sourceNamespace string) []*trafficpolicy.OutboundTrafficPolicy {
	var outboundPoliciesFromSplits []*trafficpolicy.OutboundTrafficPolicy

	apexServices := mapset.NewSet()
	for _, split := range mc.meshSpec.ListTrafficSplits() {
		svc := service.MeshService{
			Name:          k8s.GetServiceFromHostname(split.Spec.Service),
			Namespace:     split.Namespace,
			ClusterDomain: constants.LocalDomain,
		}

		locality := service.LocalCluster
		if svc.Namespace == sourceNamespace {
			locality = service.LocalNS
		}
		hostnames, err := mc.GetServiceHostnames(svc, locality)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.ErrServiceHostnames.String()).
				Msgf("Error getting service hostnames for apex service %v", svc)
			continue
		}
		policy := trafficpolicy.NewOutboundTrafficPolicy(svc.FQDN(), hostnames)

		var weightedClusters []service.WeightedCluster
		for _, backend := range split.Spec.Backends {
			ms := service.MeshService{
				Name:          backend.Service,
				Namespace:     split.ObjectMeta.Namespace,
				ClusterDomain: constants.LocalDomain,
			}
			wc := service.WeightedCluster{
				ClusterName: service.ClusterName(ms.String()),
				Weight:      backend.Weight,
			}
			weightedClusters = append(weightedClusters, wc)
		}

		rwc := trafficpolicy.NewRouteWeightedCluster(trafficpolicy.WildCardRouteMatch, weightedClusters)
		policy.Routes = []*trafficpolicy.RouteWeightedClusters{rwc}

		if apexServices.Contains(svc) {
			// TODO: enhancement(#2759)
			log.Error().Str(errcode.Kind, errcode.ErrMultipleSMISplitPerServiceUnsupported.String()).
				Msgf("Skipping Traffic Split policy %s in namespaces %s as there is already a traffic split policy for apex service %v", split.Name, split.Namespace, svc)
		} else {
			outboundPoliciesFromSplits = append(outboundPoliciesFromSplits, policy)
			apexServices.Add(svc)
		}
	}
	return outboundPoliciesFromSplits
}

// ListOutboundServicesForIdentity list the services the given service account is allowed to initiate outbound connections to
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListOutboundServicesForIdentity(serviceIdentity identity.ServiceIdentity) []service.MeshService {
	ident := serviceIdentity.ToK8sServiceAccount()
	if mc.isOSMGateway(serviceIdentity) {
		var services []service.MeshService
		for _, svc := range mc.listMeshServices() {
			// The gateway can only forward to local services.
			if svc.Local() {
				services = append(services, svc)
			}
		}
		return services
	}
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return mc.listMeshServices()
	}

	serviceSet := mapset.NewSet()
	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		for _, source := range t.Spec.Sources {
			if source.Name == ident.Name && source.Namespace == ident.Namespace { // found outbound
				sa := identity.K8sServiceAccount{
					Name:      t.Spec.Destination.Name,
					Namespace: t.Spec.Destination.Namespace,
				}
				destServices, err := mc.getServicesForServiceIdentity(sa.ToServiceIdentity())
				if err != nil {
					log.Error().Err(err).Str(errcode.Kind, errcode.ErrNoMatchingServiceForServiceAccount.String()).
						Msgf("No Services found matching Service Account %s in Namespace %s", t.Spec.Destination.Name, t.Namespace)
					break
				}
				for _, destService := range destServices {
					serviceSet.Add(destService)
				}
				break
			}
		}
	}

	var allowedServices []service.MeshService
	for elem := range serviceSet.Iter() {
		allowedServices = append(allowedServices, elem.(service.MeshService))
	}
	return allowedServices
}

func (mc *MeshCatalog) buildOutboundPermissiveModePolicies(sourceNamespace string) []*trafficpolicy.OutboundTrafficPolicy {
	var outPolicies []*trafficpolicy.OutboundTrafficPolicy

	destServices := mc.listMeshServices()

	for _, destService := range destServices {
		locality := service.LocalCluster
		if destService.Namespace == sourceNamespace {
			locality = service.LocalNS
		}
		hostnames, err := mc.GetServiceHostnames(destService, locality)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.ErrServiceHostnames.String()).
				Msgf("Error getting service hostnames for service %s", destService)
			continue
		}

		weightedCluster := getDefaultWeightedClusterForService(destService)
		policy := trafficpolicy.NewOutboundTrafficPolicy(destService.FQDN(), hostnames)
		if err := policy.AddRoute(trafficpolicy.WildCardRouteMatch, weightedCluster); err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.ErrAddingRouteToOutboundTrafficPolicy.String()).
				Msgf("Error adding route to outbound policy in permissive mode for destination %s", destService)
			continue
		}
		outPolicies = append(outPolicies, policy)
	}
	return outPolicies
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) buildOutboundPolicies(sourceServiceIdentity identity.ServiceIdentity, t *access.TrafficTarget) []*trafficpolicy.OutboundTrafficPolicy {
	source := sourceServiceIdentity.ToK8sServiceAccount()
	var outboundPolicies []*trafficpolicy.OutboundTrafficPolicy

	// fetch services running workloads with destination service account
	destServices, err := mc.getDestinationServicesFromTrafficTarget(t)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrFetchingServiceForTrafficTargetDestination.String()).
			Msgf("Error resolving destination services from TraficTarget %s/%s", t.Namespace, t.Name)
		return nil
	}

	// fetch all routes referenced in the TrafficTarget
	routeMatches, err := mc.routesFromRules(t.Spec.Rules, t.Namespace)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.ErrFetchingSMIHTTPRouteGroupForTrafficTarget.String()).
			Msgf("Error finding route matches from TrafficTarget %s/%s", t.Namespace, t.Name)
		return nil
	}

	// build an outbound traffic policy for each destination service
	for _, destService := range destServices {
		// Do not build an outbound policy if the destination service is an apex service in a traffic target
		// this will be handled while building policies from traffic split (with the backend services as weighted clusters)
		if !mc.isTrafficSplitApexService(destService) {
			locality := service.LocalCluster
			if destService.Namespace == source.Namespace {
				locality = service.LocalNS
			}
			hostnames, err := mc.GetServiceHostnames(destService, locality)
			if err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.ErrServiceHostnames.String()).
					Msgf("Error getting service hostnames for service %s", destService)
				continue
			}
			weightedCluster := getDefaultWeightedClusterForService(destService)

			policy := trafficpolicy.NewOutboundTrafficPolicy(destService.FQDN(), hostnames)
			needWildCardRoute := false
			for _, routeMatch := range routeMatches {
				// If the traffic target has a route with host headers
				// we need to create a new outbound traffic policy with the host header as the required hostnames
				// else the hosnames will be hostnames corresponding to the service
				if _, ok := routeMatch.Headers[hostHeaderKey]; ok {
					policyWithHostHeader := trafficpolicy.NewOutboundTrafficPolicy(routeMatch.Headers[hostHeaderKey], []string{routeMatch.Headers[hostHeaderKey]})
					if err := policyWithHostHeader.AddRoute(trafficpolicy.WildCardRouteMatch, weightedCluster); err != nil {
						log.Error().Err(err).Str(errcode.Kind, errcode.ErrAddingRouteToOutboundTrafficPolicy.String()).
							Msgf("Error adding Route to outbound policy for source %s/%s and destination %s/%s with host header %s", source.Namespace, source.Name, destService.Namespace, destService.Name, routeMatch.Headers[hostHeaderKey])
						continue
					}
					outboundPolicies = trafficpolicy.MergeOutboundPolicies(AllowPartialHostnamesMatch, outboundPolicies, policyWithHostHeader)
				} else {
					needWildCardRoute = true
				}
			}
			if needWildCardRoute {
				if err := policy.AddRoute(trafficpolicy.WildCardRouteMatch, weightedCluster); err != nil {
					log.Error().Err(err).Str(errcode.Kind, errcode.ErrAddingRouteToOutboundTrafficPolicy.String()).
						Msgf("Error adding Route to outbound policy for source %s/%s and destination %s/%s", source.Namespace, source.Name, destService.Namespace, destService.Name)
					continue
				}
			}

			outboundPolicies = trafficpolicy.MergeOutboundPolicies(AllowPartialHostnamesMatch, outboundPolicies, policy)
		}
	}
	return outboundPolicies
}

func (mc *MeshCatalog) getDestinationServicesFromTrafficTarget(t *access.TrafficTarget) ([]service.MeshService, error) {
	sa := identity.K8sServiceAccount{
		Name:      t.Spec.Destination.Name,
		Namespace: t.Spec.Destination.Namespace,
	}
	destServices, err := mc.getServicesForServiceIdentity(sa.ToServiceIdentity())
	if err != nil {
		return nil, errors.Errorf("Error finding Services for Service Account %#v: %v", sa, err)
	}
	return destServices, nil
}

// GetWeightedClustersForUpstream returns Envoy cluster weights for the given
// upstream service, the apex service of a TrafficSplit.
func (mc *MeshCatalog) GetWeightedClustersForUpstream(upstream service.MeshService) []service.WeightedCluster {
	var weightedClusters []service.WeightedCluster
	apexServices := mapset.NewSet()

	for _, split := range mc.meshSpec.ListTrafficSplits() {
		// Split policy must be in the same namespace as the upstream service
		if split.Namespace != upstream.Namespace {
			continue
		}
		rootServiceName := k8s.GetServiceFromHostname(split.Spec.Service)
		if rootServiceName != upstream.Name {
			// This split policy does not correspond to the upstream service
			continue
		}

		if apexServices.Contains(split.Spec.Service) {
			log.Error().Str(errcode.Kind, errcode.ErrMultipleSMISplitPerServiceUnsupported.String()).
				Msgf("Skipping traffic split policy %s/%s as there is already a corresponding policy for apex service %s", split.Namespace, split.Name, split.Spec.Service)
			continue
		}

		for _, backend := range split.Spec.Backends {
			if backend.Weight == 0 {
				// Skip backends with a weight of 0
				log.Warn().Msgf("Skipping backend %s that has a weight of 0 in traffic split policy %s/%s", backend.Service, split.Namespace, split.Name)
				continue
			}
			backendCluster := service.WeightedCluster{
				// TODO(steeling) splits only work in the local cluster as of now.
				ClusterName: service.ClusterName(split.Namespace + "/" + backend.Service + "/" + constants.LocalDomain.String()),
				Weight:      backend.Weight,
			}
			weightedClusters = append(weightedClusters, backendCluster)
		}
		apexServices.Add(split.Spec.Service)
	}

	return weightedClusters
}

// ListMeshServicesForIdentity returns a list of services the service with the
// given identity can communicate with, including apex TrafficSplit services.
func (mc *MeshCatalog) ListMeshServicesForIdentity(identity identity.ServiceIdentity) []service.MeshService {
	upstreamServices := mc.ListOutboundServicesForIdentity(identity)
	if len(upstreamServices) == 0 {
		log.Debug().Msgf("Proxy with identity %s does not have any allowed upstream services", identity)
		return nil
	}

	dstServicesSet := make(map[service.MeshService]struct{}) // Set, avoid duplicates
	// Transform into set, when listing apex services we might face repetitions
	for _, upstreamSvc := range upstreamServices {
		dstServicesSet[upstreamSvc] = struct{}{}
	}

	// Getting apex services referring to the outbound services
	// We get possible apexes which could traffic split to any of the possible
	// outbound services
	splitPolicy := mc.meshSpec.ListTrafficSplits()

	for upstreamSvc := range dstServicesSet {
		// Traffic Splits aren't yet supported for non-local services.
		if !upstreamSvc.Local() {
			continue
		}
		for _, split := range splitPolicy {
			// Split policy must be in the same namespace as the upstream service that is a backend
			if split.Namespace != upstreamSvc.Namespace {
				continue
			}
			for _, backend := range split.Spec.Backends {
				if backend.Service == upstreamSvc.Name {
					rootServiceName := k8s.GetServiceFromHostname(split.Spec.Service)
					rootMeshService := service.MeshService{
						Namespace:     split.Namespace,
						Name:          rootServiceName,
						ClusterDomain: constants.LocalDomain,
					}

					// Add this root service into the set
					dstServicesSet[rootMeshService] = struct{}{}
				}
			}
		}
	}

	dstServices := make([]service.MeshService, 0, len(dstServicesSet))
	for svc := range dstServicesSet {
		dstServices = append(dstServices, svc)
	}

	return dstServices
}
