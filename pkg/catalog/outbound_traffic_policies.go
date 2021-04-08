package catalog

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	xds_tcp_proxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/pkg/errors"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

// ListOutboundTrafficPolicies returns all outbound traffic policies
// 1. from service discovery for permissive mode
// 2. for the given service account from SMI Traffic Target and Traffic Split
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListOutboundTrafficPolicies(downstreamIdentity identity.ServiceIdentity) []*trafficpolicy.OutboundTrafficPolicy {
	downstreamServiceAccount := downstreamIdentity.ToK8sServiceAccount()
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		var outboundPolicies []*trafficpolicy.OutboundTrafficPolicy
		mergedPolicies := trafficpolicy.MergeOutboundPolicies(DisallowPartialHostnamesMatch, outboundPolicies, mc.buildOutboundPermissiveModePolicies()...)
		outboundPolicies = mergedPolicies
		return outboundPolicies
	}

	outbound := mc.listOutboundPoliciesForTrafficTargets(downstreamIdentity)
	outboundPoliciesFromSplits := mc.listOutboundTrafficPoliciesForTrafficSplits(downstreamServiceAccount.Namespace)
	outbound = trafficpolicy.MergeOutboundPolicies(DisallowPartialHostnamesMatch, outbound, outboundPoliciesFromSplits...)

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
				mergedPolicies := trafficpolicy.MergeOutboundPolicies(DisallowPartialHostnamesMatch, outboundPolicies, mc.buildOutboundPolicies(downstreamIdentity, t)...)
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
			Name:      kubernetes.GetServiceFromHostname(split.Spec.Service),
			Namespace: split.Namespace,
		}

		hostnames, err := mc.getServiceHostnames(svc, svc.Namespace == sourceNamespace)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting service hostnames for apex service %v", svc)
			continue
		}
		policy := trafficpolicy.NewOutboundTrafficPolicy(buildPolicyName(svc, sourceNamespace == svc.Namespace), hostnames)

		var weightedClusters []service.WeightedCluster
		for _, backend := range split.Spec.Backends {
			ms := service.MeshService{Name: backend.Service, Namespace: split.ObjectMeta.Namespace}
			wc := service.WeightedCluster{
				ClusterName: service.ClusterName(ms.String()),
				Weight:      backend.Weight,
			}
			weightedClusters = append(weightedClusters, wc)
		}

		rwc := trafficpolicy.NewRouteWeightedCluster(trafficpolicy.WildCardRouteMatch, weightedClusters)
		policy.Routes = []*trafficpolicy.RouteWeightedClusters{rwc}

		if apexServices.Contains(svc) {
			log.Error().Msgf("Skipping Traffic Split policy %s in namespaces %s as there is already a traffic split policy for apex service %v", split.Name, split.Namespace, svc)
		} else {
			outboundPoliciesFromSplits = append(outboundPoliciesFromSplits, policy)
			apexServices.Add(svc)
		}
	}
	return outboundPoliciesFromSplits
}

// ListAllowedOutboundServicesForIdentity list the services the given service account is allowed to initiate outbound connections to
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListAllowedOutboundServicesForIdentity(serviceIdentity identity.ServiceIdentity) []service.MeshService {
	ident := serviceIdentity.ToK8sServiceAccount()
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return mc.listMeshServices()
	}

	serviceSet := mapset.NewSet()
	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		for _, source := range t.Spec.Sources {
			if source.Name == ident.Name && source.Namespace == ident.Namespace { // found outbound
				destServices, err := mc.getServicesForServiceAccount(identity.K8sServiceAccount{
					Name:      t.Spec.Destination.Name,
					Namespace: t.Spec.Destination.Namespace,
				})
				if err != nil {
					log.Error().Err(err).Msgf("No Services found matching Service Account %s in Namespace %s", t.Spec.Destination.Name, t.Namespace)
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

// GetWeightedClustersForUpstream lists the apex services from traffic split policies
func (mc *MeshCatalog) GetWeightedClustersForUpstream(upstream service.MeshService) []*xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight {
	var weightedClusters []*xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight
	apexServices := mapset.NewSet()

	for _, split := range mc.meshSpec.ListTrafficSplits() {
		// Split policy must be in the same namespace as the upstream service
		if split.Namespace != upstream.Namespace {
			continue
		}
		rootServiceName := kubernetes.GetServiceFromHostname(split.Spec.Service)
		if rootServiceName != upstream.Name {
			// This split policy does not correspond to the upstream service
			continue
		}

		if apexServices.Contains(split.Spec.Service) {
			log.Error().Msgf("Skipping traffic split policy %s/%s as there is already a corresponding policy for apex service %s", split.Namespace, split.Name, split.Spec.Service)
			continue
		}

		for _, backend := range split.Spec.Backends {
			if backend.Weight == 0 {
				// Skip backends with a weight of 0
				log.Warn().Msgf("Skipping backend %s that has a weight of 0 in traffic split policy %s/%s", backend.Service, split.Namespace, split.Name)
				continue
			}
			backendCluster := &xds_tcp_proxy.TcpProxy_WeightedCluster_ClusterWeight{
				Name:   fmt.Sprintf("%s/%s", split.Namespace, backend.Service), // cluster <namespace>/<service>
				Weight: uint32(backend.Weight),
			}
			weightedClusters = append(weightedClusters, backendCluster)
		}
		apexServices.Add(split.Spec.Service)
	}

	return weightedClusters
}

// ListMeshServiceForServiceAccount lists the services for a given service account
func (mc *MeshCatalog) ListMeshServiceForServiceAccount(serviceIdentity identity.ServiceIdentity) map[service.MeshService]struct{} {
	upstreamServices := mc.ListAllowedOutboundServicesForIdentity(serviceIdentity)
	if len(upstreamServices) == 0 {
		log.Debug().Msgf("Proxy with identity %s does not have any allowed upstream services", serviceIdentity)
		return nil
	}

	var dstServicesSet = make(map[service.MeshService]struct{}) // Set, avoid duplicates
	// Transform into set, when listing apex services we might face repetitions
	for _, upstreamSvc := range upstreamServices {
		dstServicesSet[upstreamSvc] = struct{}{}
	}

	// Getting apex services referring to the outbound services
	// We get possible apexes which could traffic split to any of the possible
	// outbound services
	splitPolicy := mc.meshSpec.ListTrafficSplits()

	for upstreamSvc := range dstServicesSet {
		for _, split := range splitPolicy {
			// Split policy must be in the same namespace as the upstream service that is a backend
			if split.Namespace != upstreamSvc.Namespace {
				continue
			}
			for _, backend := range split.Spec.Backends {
				if backend.Service == upstreamSvc.Name {
					rootServiceName := kubernetes.GetServiceFromHostname(split.Spec.Service)
					rootMeshService := service.MeshService{
						Namespace: split.Namespace,
						Name:      rootServiceName,
					}

					// Add this root service into the set
					dstServicesSet[rootMeshService] = struct{}{}
				}
			}
		}
	}

	return dstServicesSet
}

func (mc *MeshCatalog) buildOutboundPermissiveModePolicies() []*trafficpolicy.OutboundTrafficPolicy {
	var outPolicies []*trafficpolicy.OutboundTrafficPolicy

	k8sServices := mc.kubeController.ListServices()
	var destServices []service.MeshService
	for _, k8sService := range k8sServices {
		destServices = append(destServices, utils.K8sSvcToMeshSvc(k8sService))
	}

	for _, destService := range destServices {
		hostnames, err := mc.getServiceHostnames(destService, false)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting service hostnames for service %s", destService)
			continue
		}

		weightedCluster := getDefaultWeightedClusterForService(destService)
		policy := trafficpolicy.NewOutboundTrafficPolicy(buildPolicyName(destService, false), hostnames)
		if err := policy.AddRoute(trafficpolicy.WildCardRouteMatch, weightedCluster); err != nil {
			log.Error().Err(err).Msgf("Error adding route to outbound policy in permissive mode for destination %s(%s)", destService.Name, destService.Namespace)
			continue
		}
		outPolicies = append(outPolicies, policy)
	}
	return outPolicies
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) buildOutboundPolicies(sourceServiceIdentity identity.ServiceIdentity, t *access.TrafficTarget) []*trafficpolicy.OutboundTrafficPolicy {
	source := sourceServiceIdentity.ToK8sServiceAccount()
	var outPolicies []*trafficpolicy.OutboundTrafficPolicy

	// fetch services running workloads with destination service account
	destServices, err := mc.getDestinationServicesFromTrafficTarget(t)
	if err != nil {
		log.Error().Err(err).Msgf("Error resolving destination from traffic target %s (%s)", t.Name, t.Namespace)
		return outPolicies
	}

	// build an outbound traffic policy for each destination service
	for _, destService := range destServices {
		hostnames, err := mc.getServiceHostnames(destService, source.Namespace == destService.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting service hostnames for service %s", destService)
			continue
		}
		weightedCluster := getDefaultWeightedClusterForService(destService)

		policy := trafficpolicy.NewOutboundTrafficPolicy(buildPolicyName(destService, source.Namespace == destService.Namespace), hostnames)
		if err := policy.AddRoute(trafficpolicy.WildCardRouteMatch, weightedCluster); err != nil {
			log.Error().Err(err).Msgf("Error adding Route to outbound policy for source %s(%s) and destination %s (%s)", source.Name, source.Namespace, destService.Name, destService.Namespace)
			continue
		}

		outPolicies = append(outPolicies, policy)
	}
	return outPolicies
}

func (mc *MeshCatalog) getDestinationServicesFromTrafficTarget(t *access.TrafficTarget) ([]service.MeshService, error) {
	sa := identity.K8sServiceAccount{
		Name:      t.Spec.Destination.Name,
		Namespace: t.Spec.Destination.Namespace,
	}
	destServices, err := mc.getServicesForServiceAccount(sa)
	if err != nil {
		return nil, errors.Errorf("Error finding Services for Service Account %#v: %v", sa, err)
	}
	return destServices, nil
}
