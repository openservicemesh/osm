package catalog

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
	"github.com/openservicemesh/osm/pkg/utils"
)

// ListOutboundTrafficPolicies returns all outbound traffic policies
// 1. from service discovery for permissive mode
// 2. for the given service account from SMI Traffic Target and Traffic Split
func (mc *MeshCatalog) ListOutboundTrafficPolicies(downstreamIdentity service.K8sServiceAccount) []*trafficpolicy.OutboundTrafficPolicy {
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		outboundPolicies := []*trafficpolicy.OutboundTrafficPolicy{}
		mergedPolicies := trafficpolicy.MergeOutboundPolicies(DisallowPartialHostnamesMatch, outboundPolicies, mc.buildOutboundPermissiveModePolicies()...)
		outboundPolicies = mergedPolicies
		return outboundPolicies
	}

	outbound := mc.listOutboundPoliciesForTrafficTargets(downstreamIdentity)
	outboundPoliciesFromSplits := mc.listOutboundTrafficPoliciesForTrafficSplits(downstreamIdentity.Namespace)
	outbound = trafficpolicy.MergeOutboundPolicies(DisallowPartialHostnamesMatch, outbound, outboundPoliciesFromSplits...)

	return outbound
}

// listOutboundPoliciesForTrafficTargets loops through all SMI Traffic Target resources and returns outbound traffic policies
// when the given service account matches a source in the Traffic Target resource
func (mc *MeshCatalog) listOutboundPoliciesForTrafficTargets(downstreamIdentity service.K8sServiceAccount) []*trafficpolicy.OutboundTrafficPolicy {
	outboundPolicies := []*trafficpolicy.OutboundTrafficPolicy{}

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !isValidTrafficTarget(t) {
			continue
		}

		for _, source := range t.Spec.Sources {
			if source.Name == downstreamIdentity.Name && source.Namespace == downstreamIdentity.Namespace { // found outbound
				mergedPolicies := trafficpolicy.MergeOutboundPolicies(DisallowPartialHostnamesMatch, outboundPolicies, mc.buildOutboundPolicies(downstreamIdentity, t)...)
				outboundPolicies = mergedPolicies
				break
			}
		}
	}
	return outboundPolicies
}

func (mc *MeshCatalog) listOutboundTrafficPoliciesForTrafficSplits(sourceNamespace string) []*trafficpolicy.OutboundTrafficPolicy {
	outboundPoliciesFromSplits := []*trafficpolicy.OutboundTrafficPolicy{}

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

		weightedClusters := []service.WeightedCluster{}
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
func (mc *MeshCatalog) ListAllowedOutboundServicesForIdentity(identity service.K8sServiceAccount) []service.MeshService {
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return mc.listMeshServices()
	}

	serviceSet := mapset.NewSet()
	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		for _, source := range t.Spec.Sources {
			if source.Name == identity.Name && source.Namespace == identity.Namespace { // found outbound
				destServices, err := mc.getServicesForServiceAccount(service.K8sServiceAccount{
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

func (mc *MeshCatalog) buildOutboundPermissiveModePolicies() []*trafficpolicy.OutboundTrafficPolicy {
	outPolicies := []*trafficpolicy.OutboundTrafficPolicy{}

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

func (mc *MeshCatalog) buildOutboundPolicies(source service.K8sServiceAccount, t *access.TrafficTarget) []*trafficpolicy.OutboundTrafficPolicy {
	outPolicies := []*trafficpolicy.OutboundTrafficPolicy{}

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
	sa := service.K8sServiceAccount{
		Name:      t.Spec.Destination.Name,
		Namespace: t.Spec.Destination.Namespace,
	}
	destServices, err := mc.getServicesForServiceAccount(sa)
	if err != nil {
		return nil, errors.Errorf("Error finding Services for Service Account %#v: %v", sa, err)
	}
	return destServices, nil
}
