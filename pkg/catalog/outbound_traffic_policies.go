package catalog

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// GetOutboundMeshTrafficMatches returns the traffic matches for the outbound mesh traffic policy for the given downstream identity
func (mc *MeshCatalog) GetOutboundMeshTrafficMatches(downstreamIdentity identity.ServiceIdentity) []*trafficpolicy.TrafficMatch {
	var trafficMatches []*trafficpolicy.TrafficMatch

	for _, meshSvc := range mc.ListOutboundServicesForIdentity(downstreamIdentity) {
		meshSvc := meshSvc // To prevent loop variable memory aliasing in for loop
		upstreamClusters := mc.getUpstreamClusters(meshSvc)
		var destinationIPRanges []string
		destinationIPSet := mapset.NewSet()
		for _, endp := range mc.GetResolvableEndpointsForService(meshSvc) {
			ipCIDR := endp.IP.String() + "/32"
			if added := destinationIPSet.Add(ipCIDR); added {
				destinationIPRanges = append(destinationIPRanges, ipCIDR)
			}
		}
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
	}

	return trafficMatches
}

// GetOutboundMeshClusterConfigs returns the cluster configs for the outbound mesh traffic policy for the given downstream identity
func (mc *MeshCatalog) GetOutboundMeshClusterConfigs(downstreamIdentity identity.ServiceIdentity) []*trafficpolicy.MeshClusterConfig {
	var clusterConfigs []*trafficpolicy.MeshClusterConfig

	for _, meshSvc := range mc.ListOutboundServicesForIdentity(downstreamIdentity) {
		meshSvc := meshSvc // To prevent loop variable memory aliasing in for loop

		// ---
		// Create the cluster config for this upstream service
		clusterConfigForServicePort := &trafficpolicy.MeshClusterConfig{
			Name:                          meshSvc.EnvoyClusterName(),
			Service:                       meshSvc,
			EnableEnvoyActiveHealthChecks: mc.GetMeshConfig().Spec.FeatureFlags.EnableEnvoyActiveHealthChecks,
			UpstreamTrafficSetting:        mc.GetUpstreamTrafficSettingByService(&meshSvc),
		}
		clusterConfigs = append(clusterConfigs, clusterConfigForServicePort)
	}

	return clusterConfigs
}

// GetOutboundMeshHTTPRouteConfigsPerPort returns the map of outbound traffic policies per port for the given downstream identity
func (mc *MeshCatalog) GetOutboundMeshHTTPRouteConfigsPerPort(downstreamIdentity identity.ServiceIdentity) map[int][]*trafficpolicy.OutboundTrafficPolicy {
	routeConfigPerPort := make(map[int][]*trafficpolicy.OutboundTrafficPolicy)
	downstreamSvcAccount := downstreamIdentity.ToK8sServiceAccount()

	// For each service, build the traffic policies required to access it.
	// It is important to aggregate HTTP route configs by the service's port.
	for _, meshSvc := range mc.ListOutboundServicesForIdentity(downstreamIdentity) {
		meshSvc := meshSvc // To prevent loop variable memory aliasing in for loop
		upstreamClusters := mc.getUpstreamClusters(meshSvc)
		retryPolicy := mc.getRetryPolicy(downstreamIdentity, meshSvc)

		// Build the HTTP route configs for this service and port combination.
		// If the port's protocol corresponds to TCP, we can skip this step
		if meshSvc.Protocol == constants.ProtocolTCP || meshSvc.Protocol == constants.ProtocolTCPServerFirst {
			continue
		}
		// Create a route to access the upstream service via it's hostnames and upstream weighted clusters
		httpHostNamesForServicePort := mc.GetHostnamesForService(meshSvc, downstreamSvcAccount.Namespace == meshSvc.Namespace)
		outboundTrafficPolicy := trafficpolicy.NewOutboundTrafficPolicy(meshSvc.FQDN(), httpHostNamesForServicePort)
		if err := outboundTrafficPolicy.AddRoute(trafficpolicy.WildCardRouteMatch, retryPolicy, upstreamClusters...); err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrAddingRouteToOutboundTrafficPolicy)).
				Msgf("Error adding route to outbound mesh HTTP traffic policy for destination %s", meshSvc)
			continue
		}
		routeConfigPerPort[int(meshSvc.Port)] = append(routeConfigPerPort[int(meshSvc.Port)], outboundTrafficPolicy)
	}
	return routeConfigPerPort
}

func (mc *MeshCatalog) getUpstreamClusters(meshSvc service.MeshService) []service.WeightedCluster {
	var upstreamClusters []service.WeightedCluster
	// Check if there is a traffic split corresponding to this service.
	// The upstream clusters are to be derived from the traffic split backends
	// in that case.
	trafficSplits := mc.ListTrafficSplitsByOptions(smi.WithTrafficSplitApexService(meshSvc))
	if len(trafficSplits) > 1 {
		// TODO: enhancement(#2759)
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMultipleSMISplitPerServiceUnsupported)).
			Msgf("Found more than 1 SMI TrafficSplit configuration for the same apex service %s, this is unsupported. Picking the first one!", meshSvc)
	}
	if len(trafficSplits) != 0 {
		// Program routes to the backends specified in the traffic split
		split := trafficSplits[0] // TODO(#2759): support multiple traffic splits per apex service

		for _, backend := range split.Spec.Backends {
			backendMeshSvc, err := mc.GetMeshService(backend.Service, meshSvc.Namespace, meshSvc.Port)
			if err != nil {
				log.Error().Err(err).Msgf("Error fetching target port for leaf service %s, ignoring it", backendMeshSvc)
				continue
			}

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

	return upstreamClusters
}

// ListOutboundServicesForIdentity list the services the given service account is allowed to initiate outbound connections to
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListOutboundServicesForIdentity(serviceIdentity identity.ServiceIdentity) []service.MeshService {
	if mc.GetMeshConfig().Spec.Traffic.EnablePermissiveTrafficPolicyMode {
		return mc.ListServices()
	}
	svcAccount := serviceIdentity.ToK8sServiceAccount()
	serviceSet := mapset.NewSet()
	var allowedServices []service.MeshService

	fmt.Println("\n------listing tt by options for outbound------")
	for _, t := range mc.ListTrafficTargetsByOptions() { // loop through all traffic targets
		for _, source := range t.Spec.Sources {
			if source.Name != svcAccount.Name || source.Namespace != svcAccount.Namespace {
				// Source doesn't match the downstream's service identity
				continue
			}
			sa := identity.K8sServiceAccount{
				Name:      t.Spec.Destination.Name,
				Namespace: t.Spec.Destination.Namespace,
			}
			for _, destService := range mc.GetServicesForServiceIdentity(sa.ToServiceIdentity()) {
				if added := serviceSet.Add(destService); added {
					allowedServices = append(allowedServices, destService)
				}
			}
			break
		}
	}

	return allowedServices
}
