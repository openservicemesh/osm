package catalog

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

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
	for _, meshSvc := range mc.listAllowedUpstreamServicesIncludeApex(downstreamIdentity) {
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
					Namespace:  meshSvc.Namespace, // Backends belong to the same namespace as the apex service
					Name:       backend.Service,
					TargetPort: meshSvc.TargetPort,
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

		retryPolicy := mc.getRetryPolicy(downstreamIdentity, meshSvc)

		// ---
		// Create a TrafficMatch for this upstream service and port combination.
		// The TrafficMatch will be used by LDS to program a filter chain match
		// for this upstream service, port, and destination IP ranges. This
		// will be programmed on the downstream client.
		trafficMatchForServicePort := &trafficpolicy.TrafficMatch{
			Name:                fmt.Sprintf("%s_%d_%s", meshSvc, meshSvc.Port, meshSvc.Protocol),
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

// ListOutboundServicesForMulticlusterGateway lists the upstream services for the multicluster gateway
// TODO: improve code by combining with ListOutboundServicesForIdentity
func (mc *MeshCatalog) ListOutboundServicesForMulticlusterGateway() []service.MeshService {
	return mc.listMeshServices()
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

func (mc *MeshCatalog) getDestinationServicesFromTrafficTarget(t *access.TrafficTarget) []service.MeshService {
	sa := identity.K8sServiceAccount{
		Name:      t.Spec.Destination.Name,
		Namespace: t.Spec.Destination.Namespace,
	}
	return mc.getServicesForServiceIdentity(sa.ToServiceIdentity())
}

// listAllowedUpstreamServicesIncludeApex returns a list of services the given downstream service identity
// is authorized to communicate with, including traffic split apex services that are not backed by
// pods.
func (mc *MeshCatalog) listAllowedUpstreamServicesIncludeApex(downstreamIdentity identity.ServiceIdentity) []service.MeshService {
	upstreamServices := mc.ListOutboundServicesForIdentity(downstreamIdentity)
	if len(upstreamServices) == 0 {
		log.Debug().Msgf("Downstream identity %s does not have any allowed upstream services", downstreamIdentity)
		return nil
	}

	dstServicesSet := make(map[service.MeshService]struct{}) // mapset to avoid duplicates
	for _, upstreamSvc := range upstreamServices {
		// All upstreams with an endpoint are expected to have TargetPort set.
		// Only a TrafficSplit apex service (virtual service) that does not have endpoints
		// will have an unset TargetPort. We will not include such a service in the initial
		// set because it will be correctly added to the set later on when each upstream
		// service is matched to a TrafficSplit object. This is important to avoid duplicate
		// TrafficSplit apex/virtual service from being computed with and without TargetPort set.
		if upstreamSvc.TargetPort != 0 {
			dstServicesSet[upstreamSvc] = struct{}{}
		}
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
					rootServiceName := k8s.GetServiceFromHostname(split.Spec.Service)
					rootMeshService := service.MeshService{
						Namespace:  split.Namespace,
						Name:       rootServiceName,
						Port:       upstreamSvc.Port,
						TargetPort: upstreamSvc.TargetPort,
						Protocol:   upstreamSvc.Protocol,
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
