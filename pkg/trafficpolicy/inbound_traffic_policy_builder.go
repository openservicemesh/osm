package trafficpolicy

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"

	"github.com/rs/zerolog/log"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
)

type inboundTrafficPolicyBuilder struct {
	upstreamServices                  []service.MeshService
	upstreamIdentity                  identity.ServiceIdentity
	upstreamServicesIncludeApex       []service.MeshService
	upstreamTrafficSettingsPerService map[service.MeshService]*policyv1alpha1.UpstreamTrafficSetting
	hostnamesPerService               map[service.MeshService][]string
	trafficTargetsByOptions           []*smiAccess.TrafficTarget
	httpTrafficSpecsList              []*smiSpec.HTTPRouteGroup
	enablePermissiveTrafficPolicyMode bool
	trustDomain                       certificate.TrustDomain
}

func InboundTrafficPolicyBuilder() *inboundTrafficPolicyBuilder { //nolint: revive // unexported-return
	return &inboundTrafficPolicyBuilder{}
}

func (b *inboundTrafficPolicyBuilder) UpstreamServices(upstreamServices []service.MeshService) *inboundTrafficPolicyBuilder {
	b.upstreamServices = upstreamServices
	return b
}

func (b *inboundTrafficPolicyBuilder) UpstreamIdentity(upstreamIdentity identity.ServiceIdentity) *inboundTrafficPolicyBuilder {
	b.upstreamIdentity = upstreamIdentity
	return b
}

func (b *inboundTrafficPolicyBuilder) UpstreamServicesIncludeApex(upstreamServicesIncludeApex []service.MeshService) *inboundTrafficPolicyBuilder {
	b.upstreamServicesIncludeApex = upstreamServicesIncludeApex
	return b
}

func (b *inboundTrafficPolicyBuilder) UpstreamTrafficSettingsPerService(upstreamTrafficSettingsPerService map[service.MeshService]*policyv1alpha1.UpstreamTrafficSetting) *inboundTrafficPolicyBuilder {
	b.upstreamTrafficSettingsPerService = upstreamTrafficSettingsPerService
	return b
}

func (b *inboundTrafficPolicyBuilder) HostnamesPerService(hostnamesPerService map[service.MeshService][]string) *inboundTrafficPolicyBuilder {
	b.hostnamesPerService = hostnamesPerService
	return b
}

func (b *inboundTrafficPolicyBuilder) TrafficTargetsByOptions(trafficTargetsByOptions []*smiAccess.TrafficTarget) *inboundTrafficPolicyBuilder {
	b.trafficTargetsByOptions = trafficTargetsByOptions
	return b
}

func (b *inboundTrafficPolicyBuilder) HTTPTrafficSpecsList(httpTrafficSpecsList []*smiSpec.HTTPRouteGroup) *inboundTrafficPolicyBuilder {
	b.httpTrafficSpecsList = httpTrafficSpecsList
	return b
}

func (b *inboundTrafficPolicyBuilder) EnablePermissiveTrafficPolicyMode(enablePermissiveTrafficPolicyMode bool) *inboundTrafficPolicyBuilder {
	b.enablePermissiveTrafficPolicyMode = enablePermissiveTrafficPolicyMode
	return b
}

func (b *inboundTrafficPolicyBuilder) TrustDomain(trustDomain certificate.TrustDomain) *inboundTrafficPolicyBuilder {
	b.trustDomain = trustDomain
	return b
}

// GetInboundMeshClusterConfigs returns the cluster configs for the inbound mesh traffic policy for the given upstream services
func (b *inboundTrafficPolicyBuilder) GetInboundMeshClusterConfigs() []*MeshClusterConfig {
	// Used to avoid duplicate clusters that can arise when multiple
	// upstream services reference the same global rate limit service
	rlsClusterSet := mapset.NewSet()

	var clusterConfigs []*MeshClusterConfig

	// Build configurations per upstream service
	for _, upstreamSvc := range b.upstreamServicesIncludeApex {
		upstreamSvc := upstreamSvc // To prevent loop variable memory aliasing in for loop

		// ---
		// Create local cluster configs for this upstram service
		clusterConfigForSvc := &MeshClusterConfig{
			Name:    upstreamSvc.EnvoyLocalClusterName(),
			Service: upstreamSvc,
			Address: constants.LocalhostIPAddress,
			Port:    uint32(upstreamSvc.TargetPort),
		}
		clusterConfigs = append(clusterConfigs, clusterConfigForSvc)

		upstreamTrafficSetting := b.upstreamTrafficSettingsPerService[upstreamSvc]
		clusterConfigs = append(clusterConfigs, getRateLimitServiceClusters(upstreamTrafficSetting, rlsClusterSet)...)
	}

	return clusterConfigs
}

// GetInboundMeshTrafficMatches returns the traffic matches for the inbound mesh traffic policy for the given upstream services
func (b *inboundTrafficPolicyBuilder) GetInboundMeshTrafficMatches() []*TrafficMatch {
	var trafficMatches []*TrafficMatch

	// Build configurations per upstream service
	for _, upstreamSvc := range b.upstreamServices {
		upstreamSvc := upstreamSvc // To prevent loop variable memory aliasing in for loop

		upstreamTrafficSetting := b.upstreamTrafficSettingsPerService[upstreamSvc]

		// ---
		// Create a TrafficMatch for this upstream servic.
		// The TrafficMatch will be used by LDS to program a filter chain match
		// for this upstream service on the upstream server to accept inbound
		// traffic.
		//
		// Note: a TrafficMatch must exist only for a service part of the given
		// 'upstreamServices' list, and not a virtual (apex) service that
		// may be returned as a part of the 'allUpstreamServices' list.
		// A virtual (apex) service is required for the purpose of building
		// HTTP routing rules, but should not result in a TrafficMatch rule
		// as TrafficMatch rules are meant to map to actual services backed
		// by a proxy, defined by the 'upstreamServices' list.
		trafficMatchForUpstreamSvc := &TrafficMatch{
			Name:                upstreamSvc.InboundTrafficMatchName(),
			DestinationPort:     int(upstreamSvc.TargetPort),
			DestinationProtocol: upstreamSvc.Protocol,
			ServerNames:         []string{upstreamSvc.ServerName()},
			Cluster:             upstreamSvc.EnvoyLocalClusterName(),
		}
		if upstreamTrafficSetting != nil {
			trafficMatchForUpstreamSvc.RateLimit = upstreamTrafficSetting.Spec.RateLimit
		}
		trafficMatches = append(trafficMatches, trafficMatchForUpstreamSvc)
	}

	return trafficMatches
}

// GetInboundMeshHTTPRouteConfigsPerPort returns a map of the given inbound traffic policy per port for the given upstream identity and services
func (b *inboundTrafficPolicyBuilder) GetInboundMeshHTTPRouteConfigsPerPort() map[int][]*InboundTrafficPolicy {
	var trafficTargets []*smiAccess.TrafficTarget
	routeConfigPerPort := make(map[int][]*InboundTrafficPolicy)

	if !b.enablePermissiveTrafficPolicyMode {
		// Pre-computing the list of TrafficTarget optimizes to avoid repeated
		// cache lookups for each upstream service.
		trafficTargets = b.trafficTargetsByOptions
	}

	// Build configurations per upstream service
	for _, upstreamSvc := range b.upstreamServicesIncludeApex {
		upstreamSvc := upstreamSvc // To prevent loop variable memory aliasing in for loop

		upstreamTrafficSetting := b.upstreamTrafficSettingsPerService[upstreamSvc]

		// Build the HTTP route configs for this service and port combination.
		// If the port's protocol corresponds to TCP, we can skip this step
		if upstreamSvc.Protocol == constants.ProtocolTCP || upstreamSvc.Protocol == constants.ProtocolTCPServerFirst {
			continue
		}
		// ---
		// Build the HTTP route configs per port
		// Each upstream service accepts traffic from downstreams on a list of allowed routes.
		// The routes are derived from SMI TrafficTarget and TrafficSplit policies in SMI mode,
		// and are wildcarded in permissive mode. The downstreams that can access this upstream
		// on the configured routes is also determined based on the traffic policy mode.
		inboundTrafficPolicies := b.getInboundTrafficPoliciesForUpstream(upstreamSvc, trafficTargets, upstreamTrafficSetting)
		routeConfigPerPort[int(upstreamSvc.TargetPort)] = append(routeConfigPerPort[int(upstreamSvc.TargetPort)], inboundTrafficPolicies)
	}

	return routeConfigPerPort
}

func (b *inboundTrafficPolicyBuilder) getInboundTrafficPoliciesForUpstream(upstreamSvc service.MeshService,
	trafficTargets []*smiAccess.TrafficTarget, upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting) *InboundTrafficPolicy {
	var inboundPolicyForUpstreamSvc *InboundTrafficPolicy

	if b.enablePermissiveTrafficPolicyMode {
		// Add a wildcard HTTP route that allows any downstream client to access the upstream service
		hostnames := b.hostnamesPerService[upstreamSvc]
		inboundPolicyForUpstreamSvc = NewInboundTrafficPolicy(upstreamSvc.FQDN(), hostnames, upstreamTrafficSetting)
		localCluster := service.WeightedCluster{
			ClusterName: service.ClusterName(upstreamSvc.EnvoyLocalClusterName()),
			Weight:      constants.ClusterWeightAcceptAll,
		}
		// Only a single rule for permissive mode.
		inboundPolicyForUpstreamSvc.Rules = []*Rule{
			{
				Route:             *NewRouteWeightedCluster(WildCardRouteMatch, []service.WeightedCluster{localCluster}, upstreamTrafficSetting),
				AllowedPrincipals: mapset.NewSetWith(identity.WildcardPrincipal),
			},
		}
	} else {
		// Build the HTTP routes from SMI TrafficTarget and HTTPRouteGroup configurations
		inboundPolicyForUpstreamSvc = b.buildInboundHTTPPolicyFromTrafficTarget(upstreamSvc, trafficTargets, upstreamTrafficSetting)
	}

	return inboundPolicyForUpstreamSvc
}

func (b *inboundTrafficPolicyBuilder) buildInboundHTTPPolicyFromTrafficTarget(upstreamSvc service.MeshService, trafficTargets []*smiAccess.TrafficTarget,
	upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting) *InboundTrafficPolicy {
	hostnames := b.hostnamesPerService[upstreamSvc]
	inboundPolicy := NewInboundTrafficPolicy(upstreamSvc.FQDN(), hostnames, upstreamTrafficSetting)

	localCluster := service.WeightedCluster{
		ClusterName: service.ClusterName(upstreamSvc.EnvoyLocalClusterName()),
		Weight:      constants.ClusterWeightAcceptAll,
	}

	var routingRules []*Rule
	// From each TrafficTarget and HTTPRouteGroup configuration associated with this service, build routes for it.
	for _, trafficTarget := range trafficTargets {
		rules := b.getRoutingRulesFromTrafficTarget(*trafficTarget, localCluster, upstreamTrafficSetting)
		// Multiple TrafficTarget objects can reference the same route, in which case such routes
		// need to be merged to create a single route that includes all the downstream client identities
		// this route is authorized for.
		routingRules = MergeRules(routingRules, rules)
	}
	inboundPolicy.Rules = routingRules

	return inboundPolicy
}

func (b *inboundTrafficPolicyBuilder) getRoutingRulesFromTrafficTarget(trafficTarget smiAccess.TrafficTarget, routingCluster service.WeightedCluster,
	upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting) []*Rule {
	// Compute the HTTP route matches associated with the given TrafficTarget object
	httpRouteMatches, err := b.RoutesFromRules(trafficTarget.Spec.Rules, trafficTarget.Namespace)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingSMIHTTPRouteGroupForTrafficTarget)).
			Msgf("Error finding route matches from TrafficTarget %s in namespace %s", trafficTarget.Name, trafficTarget.Namespace)
		return nil
	}

	// Compute the allowed downstream service identities for the given TrafficTarget object
	allowedDownstreamPrincipals := mapset.NewSet()
	for _, source := range trafficTarget.Spec.Sources {
		allowedDownstreamPrincipals.Add(trafficTargetIdentityToSvcAccount(source).AsPrincipal(b.trustDomain.Signing))

		if b.trustDomain.AreDifferent() {
			allowedDownstreamPrincipals.Add(trafficTargetIdentityToSvcAccount(source).AsPrincipal(b.trustDomain.Validating))
		}
	}

	var routingRules []*Rule
	for _, httpRouteMatch := range httpRouteMatches {
		rule := &Rule{
			Route:             *NewRouteWeightedCluster(httpRouteMatch, []service.WeightedCluster{routingCluster}, upstreamTrafficSetting),
			AllowedPrincipals: allowedDownstreamPrincipals,
		}
		routingRules = append(routingRules, rule)
	}

	return routingRules
}

func trafficTargetIdentityToSvcAccount(identitySubject smiAccess.IdentityBindingSubject) identity.K8sServiceAccount {
	return identity.K8sServiceAccount{
		Name:      identitySubject.Name,
		Namespace: identitySubject.Namespace,
	}
}

// RoutesFromRules takes a set of traffic target rules and the namespace of the traffic target and returns a list of
// http route matches (trafficpolicy.HTTPRouteMatch)
func (b *inboundTrafficPolicyBuilder) RoutesFromRules(rules []smiAccess.TrafficTargetRule, trafficTargetNamespace string) ([]HTTPRouteMatch, error) {
	var routes []HTTPRouteMatch

	specMatchRoute, err := b.GetHTTPPathsPerRoute() // returns map[traffic_spec_name]map[match_name]trafficpolicy.HTTPRoute
	if err != nil {
		return nil, err
	}

	if len(specMatchRoute) == 0 {
		log.Trace().Msg("No elements in map[traffic_spec_name]map[match name]trafficpolicyHTTPRoute")
		return routes, nil
	}

	for _, rule := range rules {
		trafficSpecName := GetTrafficSpecName(smi.HTTPRouteGroupKind, trafficTargetNamespace, rule.Name)
		for _, match := range rule.Matches {
			matchedRoute, found := specMatchRoute[trafficSpecName][TrafficSpecMatchName(match)]
			if found {
				routes = append(routes, matchedRoute)
			} else {
				log.Debug().Msgf("No matching trafficpolicy.HTTPRoute found for match name %s in Traffic Spec %s (in namespace %s)", match, trafficSpecName, trafficTargetNamespace)
			}
		}
	}

	return routes, nil
}

func (b *inboundTrafficPolicyBuilder) GetHTTPPathsPerRoute() (map[TrafficSpecName]map[TrafficSpecMatchName]HTTPRouteMatch, error) {
	routePolicies := make(map[TrafficSpecName]map[TrafficSpecMatchName]HTTPRouteMatch)
	for _, trafficSpecs := range b.httpTrafficSpecsList {
		log.Debug().Msgf("Discovered TrafficSpec resource: %s/%s", trafficSpecs.Namespace, trafficSpecs.Name)
		if trafficSpecs.Spec.Matches == nil {
			log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrSMIHTTPRouteGroupNoMatch)).
				Msgf("TrafficSpec %s/%s has no matches in route; Skipping...", trafficSpecs.Namespace, trafficSpecs.Name)
			continue
		}

		// since this method gets only specs related to HTTPRouteGroups added HTTPTraffic to the specKey by default
		specKey := GetTrafficSpecName(smi.HTTPRouteGroupKind, trafficSpecs.Namespace, trafficSpecs.Name)
		routePolicies[specKey] = make(map[TrafficSpecMatchName]HTTPRouteMatch)
		for _, trafficSpecsMatches := range trafficSpecs.Spec.Matches {
			serviceRoute := HTTPRouteMatch{
				Path:          trafficSpecsMatches.PathRegex,
				PathMatchType: PathMatchRegex,
				Methods:       trafficSpecsMatches.Methods,
				Headers:       trafficSpecsMatches.Headers,
			}

			// When pathRegex or/and methods are not defined, they will be wildcarded
			if serviceRoute.Path == "" {
				serviceRoute.Path = constants.RegexMatchAll
			}
			if len(serviceRoute.Methods) == 0 {
				serviceRoute.Methods = []string{constants.WildcardHTTPMethod}
			}
			routePolicies[specKey][TrafficSpecMatchName(trafficSpecsMatches.Name)] = serviceRoute
		}
	}
	log.Debug().Msgf("Constructed HTTP path routes: %+v", routePolicies)
	return routePolicies, nil
}

// GetTrafficSpectName returns the formatted TrafficSpecName from the Traffic Spec kind, namespace, name.
func GetTrafficSpecName(trafficSpecKind string, trafficSpecNamespace string, trafficSpecName string) TrafficSpecName {
	specKey := fmt.Sprintf("%s/%s/%s", trafficSpecKind, trafficSpecNamespace, trafficSpecName)
	return TrafficSpecName(specKey)
}

// getRateLimitServiceClusters returns a list of MeshClusterConfig objects corresponding to the global
// rate limit service instance. It ensures only a single cluster config if the same rate limit service
// is used for both TCP and HTTP rate limiting.
func getRateLimitServiceClusters(upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting, clusterSet mapset.Set) []*MeshClusterConfig {
	if upstreamTrafficSetting == nil || upstreamTrafficSetting.Spec.RateLimit == nil || upstreamTrafficSetting.Spec.RateLimit.Global == nil {
		return nil
	}

	rateLimit := upstreamTrafficSetting.Spec.RateLimit
	var clusters []*MeshClusterConfig

	if rateLimit.Global.TCP != nil {
		clusterName := service.RateLimitServiceClusterName(rateLimit.Global.TCP.RateLimitService)
		if !clusterSet.Contains(clusterName) {
			clusters = append(clusters, &MeshClusterConfig{
				Name:     clusterName,
				Address:  rateLimit.Global.TCP.RateLimitService.Host,
				Port:     uint32(rateLimit.Global.TCP.RateLimitService.Port),
				Protocol: constants.ProtocolH2C,
			})
			clusterSet.Add(clusterName)
		}
	}

	if rateLimit.Global.HTTP != nil {
		clusterName := service.RateLimitServiceClusterName(rateLimit.Global.HTTP.RateLimitService)
		// Only configure an HTTP rate limiting cluster if the same cluster is not already
		// referenced by a TCP rate limiting config
		if !clusterSet.Contains(clusterName) {
			clusters = append(clusters, &MeshClusterConfig{
				Name:     clusterName,
				Address:  rateLimit.Global.HTTP.RateLimitService.Host,
				Port:     uint32(rateLimit.Global.HTTP.RateLimitService.Port),
				Protocol: constants.ProtocolH2C,
			})
			clusterSet.Add(clusterName)
		}
	}

	return clusters
}
