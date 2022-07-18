package catalog

import (
	mapset "github.com/deckarep/golang-set"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// GetOutboundMeshTrafficPolicy returns the outbound mesh traffic policy for the given downstream identity
//
// The function works as follows:
// 1. If permissive mode is enabled, builds outbound mesh traffic policies to reach every upstream service
//    discovered using service discovery, using wildcard routes.
// 2. In SMI mode, builds outbound mesh traffic policies to reach every upstream service corresponding
//    to every upstream service account that this downstream is authorized to access using SMI TrafficTarget
//    policies.
// 3. Process TraficSplit policies and update the weights for the upstream services based on the policies.
//
// The route configurations are consolidated per port, such that upstream services using the same port are a part
// of the same route configuration. This is required to avoid route conflicts that can occur when the same hostname
// needs to be routed differently based on the port used.
func (mc *MeshCatalog) GetOutboundMeshTrafficPolicy(downstreamIdentity identity.ServiceIdentity) *trafficpolicy.OutboundMeshTrafficPolicy {
	var trafficMatches []*trafficpolicy.TrafficMatch
	var clusterConfigs []*trafficpolicy.MeshClusterConfig
	routeConfigPerPort := make(map[int][]*trafficpolicy.OutboundTrafficPolicy)
	downstreamSvcAccount := downstreamIdentity.ToK8sServiceAccount()

	// For each service, build the traffic policies required to access it.
	// It is important to aggregate HTTP route configs by the service's port.
	for _, meshSvc := range mc.ListOutboundServicesForIdentity(downstreamIdentity) {
		meshSvc := meshSvc // To prevent loop variable memory aliasing in for loop

		// Retrieve the destination IP address from the endpoints for this service
		// IP range must not have duplicates, use a mapset to only add unique IP ranges
		var destinationIPRanges []string
		destinationIPSet := mapset.NewSet()
		for _, endp := range mc.getDNSResolvableServiceEndpoints(meshSvc) {
			ipCIDR := endp.IP.String() + "/32"
			if added := destinationIPSet.Add(ipCIDR); added {
				destinationIPRanges = append(destinationIPRanges, ipCIDR)
			}
		}

		// ---
		// Create the cluster config for this upstream service
		clusterConfigForServicePort := &trafficpolicy.MeshClusterConfig{
			Name:                          meshSvc.EnvoyClusterName(),
			Service:                       meshSvc,
			EnableEnvoyActiveHealthChecks: mc.configurator.GetFeatureFlags().EnableEnvoyActiveHealthChecks,
			UpstreamTrafficSetting: mc.policyController.GetUpstreamTrafficSetting(
				policy.UpstreamTrafficSettingGetOpt{MeshService: &meshSvc}),
		}
		clusterConfigs = append(clusterConfigs, clusterConfigForServicePort)

		var upstreamClusters []service.WeightedCluster
		// Check if there is a traffic split corresponding to this service.
		// The upstream clusters are to be derived from the traffic split backends
		// in that case.
		trafficSplits := mc.meshSpec.ListTrafficSplits(smi.WithTrafficSplitApexService(meshSvc))
		if len(trafficSplits) > 1 {
			// TODO: enhancement(#2759)
			log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMultipleSMISplitPerServiceUnsupported)).
				Msgf("Found more than 1 SMI TrafficSplit configuration for the same apex service %s, this is unsupported. Picking the first one!", meshSvc)
		}
		if len(trafficSplits) != 0 {
			// Program routes to the backends specified in the traffic split
			split := trafficSplits[0] // TODO(#2759): support multiple traffic splits per apex service

			for _, backend := range split.Spec.Backends {
				backendMeshSvc := service.MeshService{
					Namespace: meshSvc.Namespace, // Backends belong to the same namespace as the apex service
					Name:      backend.Service,
				}
				targetPort, err := mc.kubeController.GetTargetPortForServicePort(
					types.NamespacedName{Namespace: backendMeshSvc.Namespace, Name: backendMeshSvc.Name}, meshSvc.Port)
				if err != nil {
					log.Error().Err(err).Msgf("Error fetching target port for leaf service %s, ignoring it", backendMeshSvc)
					continue
				}
				backendMeshSvc.TargetPort = targetPort

				wc := service.WeightedCluster{
					ClusterName: service.ClusterName(backendMeshSvc.EnvoyClusterName()),
					Weight:      backend.Weight,
				}
				upstreamClusters = append(upstreamClusters, wc)
			}
		} else {
			wc := service.WeightedCluster{
				ClusterName: service.ClusterName(meshSvc.EnvoyClusterName()),
				Weight:      constants.ClusterWeightAcceptAll,
			}
			// No TrafficSplit for this upstream service, so use a default weighted cluster
			upstreamClusters = append(upstreamClusters, wc)
		}

		retryPolicy := mc.getRetryPolicy(downstreamIdentity, meshSvc)

		// ---
		// Create a TrafficMatch for this upstream service and port combination.
		// The TrafficMatch will be used by LDS to program a filter chain match
		// for this upstream service, port, and destination IP ranges. This
		// will be programmed on the downstream client.
		trafficMatchForServicePort := &trafficpolicy.TrafficMatch{
			Name:                meshSvc.OutboundTrafficMatchName(),
			DestinationPort:     int(meshSvc.Port),
			DestinationProtocol: meshSvc.Protocol,
			DestinationIPRanges: destinationIPRanges,
			WeightedClusters:    upstreamClusters,
		}
		trafficMatches = append(trafficMatches, trafficMatchForServicePort)
		log.Trace().Msgf("Built traffic match %s for downstream %s", trafficMatchForServicePort.Name, downstreamIdentity)

		// Build the HTTP route configs for this service and port combination.
		// If the port's protocol corresponds to TCP, we can skip this step
		if meshSvc.Protocol == constants.ProtocolTCP || meshSvc.Protocol == constants.ProtocolTCPServerFirst {
			continue
		}
		// Create a route to access the upstream service via it's hostnames and upstream weighted clusters
		httpHostNamesForServicePort := k8s.GetHostnamesForService(meshSvc, downstreamSvcAccount.Namespace == meshSvc.Namespace)
		outboundTrafficPolicy := trafficpolicy.NewOutboundTrafficPolicy(meshSvc.FQDN(), httpHostNamesForServicePort)
		if err := outboundTrafficPolicy.AddRoute(trafficpolicy.WildCardRouteMatch, retryPolicy, upstreamClusters...); err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrAddingRouteToOutboundTrafficPolicy)).
				Msgf("Error adding route to outbound mesh HTTP traffic policy for destination %s", meshSvc)
			continue
		}
		routeConfigPerPort[int(meshSvc.Port)] = append(routeConfigPerPort[int(meshSvc.Port)], outboundTrafficPolicy)
	}

	return &trafficpolicy.OutboundMeshTrafficPolicy{
		TrafficMatches:          trafficMatches,
		ClustersConfigs:         clusterConfigs,
		HTTPRouteConfigsPerPort: routeConfigPerPort,
	}
}

// ListOutboundServicesForIdentity list the services the given service account is allowed to initiate outbound connections to
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListOutboundServicesForIdentity(serviceIdentity identity.ServiceIdentity) []service.MeshService {
	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return mc.listMeshServices()
	}

	svcAccount := serviceIdentity.ToK8sServiceAccount()
	serviceSet := mapset.NewSet()
	var allowedServices []service.MeshService
	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		for _, source := range t.Spec.Sources {
			if source.Name != svcAccount.Name || source.Namespace != svcAccount.Namespace {
				// Source doesn't match the downstream's service identity
				continue
			}

			sa := identity.K8sServiceAccount{
				Name:      t.Spec.Destination.Name,
				Namespace: t.Spec.Destination.Namespace,
			}

			for _, destService := range mc.getServicesForServiceIdentity(sa.ToServiceIdentity()) {
				if added := serviceSet.Add(destService); added {
					allowedServices = append(allowedServices, destService)
				}
			}
			break
		}
	}

	return allowedServices
}
