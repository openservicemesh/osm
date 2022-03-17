package catalog

import (
	"fmt"
	"net"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	"k8s.io/apimachinery/pkg/types"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// upstreamTrafficSettingKind is the upstreamTrafficSettingKind API kind
	upstreamTrafficSettingKind = "UpstreamTrafficSetting"
)

// GetEgressTrafficPolicy returns the Egress traffic policy associated with the given service identity
func (mc *MeshCatalog) GetEgressTrafficPolicy(serviceIdentity identity.ServiceIdentity) (*trafficpolicy.EgressTrafficPolicy, error) {
	if !mc.configurator.GetFeatureFlags().EnableEgressPolicy {
		return nil, nil
	}

	var trafficMatches []*trafficpolicy.TrafficMatch
	var clusterConfigs []*trafficpolicy.EgressClusterConfig
	portToRouteConfigMap := make(map[int][]*trafficpolicy.EgressHTTPRouteConfig)
	egressResources := mc.policyController.ListEgressPoliciesForSourceIdentity(serviceIdentity.ToK8sServiceAccount())

	for _, egress := range egressResources {
		upstreamTrafficSetting, err := mc.getUpstreamTrafficSettingForEgress(egress)
		if err != nil {
			log.Error().Err(err).Msg("Ignoring invalid Egress policy")
			continue
		}

		for _, portSpec := range egress.Spec.Ports {
			switch strings.ToLower(portSpec.Protocol) {
			case constants.ProtocolHTTP:
				// ---
				// Build the HTTP route configs for the given Egress policy
				httpRouteConfigs, httpClusterConfigs := mc.buildHTTPRouteConfigs(egress, portSpec.Number, upstreamTrafficSetting)
				portToRouteConfigMap[portSpec.Number] = append(portToRouteConfigMap[portSpec.Number], httpRouteConfigs...)
				clusterConfigs = append(clusterConfigs, httpClusterConfigs...)

				// Configure port based TrafficMatch for HTTP port
				trafficMatches = append(trafficMatches, &trafficpolicy.TrafficMatch{
					DestinationPort:     portSpec.Number,
					DestinationProtocol: portSpec.Protocol,
				})

			case constants.ProtocolTCP, constants.ProtocolTCPServerFirst:
				// ---
				// Build the TCP cluster config for this port
				clusterConfigs = append(clusterConfigs, &trafficpolicy.EgressClusterConfig{
					Name:                   fmt.Sprintf("%d", portSpec.Number),
					Port:                   portSpec.Number,
					UpstreamTrafficSetting: upstreamTrafficSetting,
				})

				// Configure port + IP range TrafficMatches
				trafficMatches = append(trafficMatches, &trafficpolicy.TrafficMatch{
					DestinationPort:     portSpec.Number,
					DestinationProtocol: portSpec.Protocol,
					DestinationIPRanges: egress.Spec.IPAddresses,
					Cluster:             fmt.Sprintf("%d", portSpec.Number),
				})

			case constants.ProtocolHTTPS:
				// ---
				// Build the HTTPS cluster config for this port
				// HTTPS is TLS encrypted, so will be proxied as a TCP stream
				clusterConfigs = append(clusterConfigs, &trafficpolicy.EgressClusterConfig{
					Name:                   fmt.Sprintf("%d", portSpec.Number),
					Port:                   portSpec.Number,
					UpstreamTrafficSetting: upstreamTrafficSetting,
				})

				// Configure port + IP range TrafficMatches
				trafficMatches = append(trafficMatches, &trafficpolicy.TrafficMatch{
					DestinationPort:     portSpec.Number,
					DestinationProtocol: portSpec.Protocol,
					DestinationIPRanges: egress.Spec.IPAddresses,
					ServerNames:         egress.Spec.Hosts,
					Cluster:             fmt.Sprintf("%d", portSpec.Number),
				})
			}
		}
	}

	var err error
	// Deduplicate the list of TrafficMatch objects
	trafficMatches, err = trafficpolicy.DeduplicateTrafficMatches(trafficMatches)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDedupEgressTrafficMatches)).
			Msgf("Error deduplicating egress traffic matches for service identity %s", serviceIdentity)
		return nil, err
	}

	// Deduplicate the list of EgressClusterConfig objects
	clusterConfigs, err = trafficpolicy.DeduplicateClusterConfigs(clusterConfigs)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDedupEgressClusterConfigs)).
			Msgf("Error deduplicating egress clusters configs for service identity %s", serviceIdentity)
		return nil, err
	}

	return &trafficpolicy.EgressTrafficPolicy{
		HTTPRouteConfigsPerPort: portToRouteConfigMap,
		TrafficMatches:          trafficMatches,
		ClustersConfigs:         clusterConfigs,
	}, nil
}

func (mc *MeshCatalog) getUpstreamTrafficSettingForEgress(egressPolicy *policyv1alpha1.Egress) (*policyv1alpha1.UpstreamTrafficSetting, error) {
	if egressPolicy == nil {
		return nil, nil
	}

	for _, match := range egressPolicy.Spec.Matches {
		if match.APIGroup != nil && *match.APIGroup == policyv1alpha1.SchemeGroupVersion.String() && match.Kind == upstreamTrafficSettingKind {
			namespacedName := types.NamespacedName{
				Namespace: egressPolicy.Namespace,
				Name:      match.Name,
			}
			upstreamtrafficSetting := mc.policyController.GetUpstreamTrafficSetting(
				policy.UpstreamTrafficSettingGetOpt{NamespacedName: &namespacedName})

			if upstreamtrafficSetting == nil {
				return nil, errors.Errorf("UpstreamTrafficSetting %s specified in Egress policy %s/%s could not be found, ignoring it",
					namespacedName.String(), egressPolicy.Namespace, egressPolicy.Name)
			}

			return upstreamtrafficSetting, nil
		}
	}

	return nil, nil
}

func (mc *MeshCatalog) buildHTTPRouteConfigs(egressPolicy *policyv1alpha1.Egress, port int,
	upstreamTrafficSetting *policyv1alpha1.UpstreamTrafficSetting) ([]*trafficpolicy.EgressHTTPRouteConfig, []*trafficpolicy.EgressClusterConfig) {
	if egressPolicy == nil {
		return nil, nil
	}

	var routeConfigs []*trafficpolicy.EgressHTTPRouteConfig
	var clusterConfigs []*trafficpolicy.EgressClusterConfig

	// Before building the route configs, pre-compute the allowed IP ranges since they
	// will be the same for every HTTP route config derived from the given Egress policy.
	var allowedDestinationIPRanges []string
	destIPSet := mapset.NewSet()
	for _, ipRange := range egressPolicy.Spec.IPAddresses {
		if _, _, err := net.ParseCIDR(ipRange); err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidEgressIPRange)).
				Msgf("Invalid IP range [%s] specified in egress policy %s/%s; will be skipped", ipRange, egressPolicy.Namespace, egressPolicy.Name)
			continue
		}
		newlyAdded := destIPSet.Add(ipRange)
		if newlyAdded {
			allowedDestinationIPRanges = append(allowedDestinationIPRanges, ipRange)
		}
	}

	// Check if there are object references to HTTP routes specified
	// in the Egress policy's 'matches' attribute. If there are HTTP route
	// matches, apply these routes.
	var httpRouteMatches []trafficpolicy.HTTPRouteMatch
	httpMatchSpecified := false
	for _, match := range egressPolicy.Spec.Matches {
		if match.APIGroup != nil && *match.APIGroup == smiSpecs.SchemeGroupVersion.String() && match.Kind == smi.HTTPRouteGroupKind {
			// HTTPRouteGroup resource referenced, build a routing rule from this resource
			httpMatchSpecified = true

			// A TypedLocalObjectReference (Spec.Matches) is a reference to another object in the same namespace
			httpRouteName := fmt.Sprintf("%s/%s", egressPolicy.Namespace, match.Name)
			if httpRouteGroup := mc.meshSpec.GetHTTPRouteGroup(httpRouteName); httpRouteGroup == nil {
				log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEgressSMIHTTPRouteGroupNotFound)).
					Msgf("Error fetching HTTPRouteGroup resource %s referenced in Egress policy %s/%s", httpRouteName, egressPolicy.Namespace, egressPolicy.Name)
			} else {
				matches := getHTTPRouteMatchesFromHTTPRouteGroup(httpRouteGroup)
				httpRouteMatches = append(httpRouteMatches, matches...)
			}
		} else {
			log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidEgressMatches)).
				Msgf("Unsupported match object specified: %v, ignoring it", match)
		}
	}

	if !httpMatchSpecified {
		// No HTTP match specified, use a wildcard
		httpRouteMatches = append(httpRouteMatches, trafficpolicy.WildCardRouteMatch)
	}

	// Parse the hosts specified and build routing rules for the specified hosts
	for _, host := range egressPolicy.Spec.Hosts {
		// A route matching an HTTP host will include host header matching for the following:
		// 1. host (ex. foo.com)
		// 2. host:port (ex. foo.com:80)
		hostnameWithPort := fmt.Sprintf("%s:%d", host, port)
		hostnames := []string{host, hostnameWithPort}

		// Create cluster config for this host and port combination
		clusterName := hostnameWithPort
		clusterConfig := &trafficpolicy.EgressClusterConfig{
			Name:                   clusterName,
			Host:                   host,
			Port:                   port,
			UpstreamTrafficSetting: upstreamTrafficSetting,
		}
		clusterConfigs = append(clusterConfigs, clusterConfig)

		// Build egress routing rules from the given HTTP route matches and allowed destination attributes
		var httpRoutingRules []*trafficpolicy.EgressHTTPRoutingRule
		for _, match := range httpRouteMatches {
			routeWeightedCluster := trafficpolicy.RouteWeightedClusters{
				HTTPRouteMatch: match,
				WeightedClusters: mapset.NewSetFromSlice([]interface{}{
					service.WeightedCluster{ClusterName: service.ClusterName(clusterName), Weight: constants.ClusterWeightAcceptAll},
				}),
			}
			routingRule := &trafficpolicy.EgressHTTPRoutingRule{
				Route:                      routeWeightedCluster,
				AllowedDestinationIPRanges: allowedDestinationIPRanges,
			}
			httpRoutingRules = append(httpRoutingRules, routingRule)
		}

		// Hostnames and routing rules are computed for the given host, build an HTTP route config for it
		hostSpecificRouteConfig := &trafficpolicy.EgressHTTPRouteConfig{
			Name:         host,
			Hostnames:    hostnames,
			RoutingRules: httpRoutingRules,
		}

		routeConfigs = append(routeConfigs, hostSpecificRouteConfig)
	}

	return routeConfigs, clusterConfigs
}

func getHTTPRouteMatchesFromHTTPRouteGroup(httpRouteGroup *smiSpecs.HTTPRouteGroup) []trafficpolicy.HTTPRouteMatch {
	if httpRouteGroup == nil {
		return nil
	}

	var matches []trafficpolicy.HTTPRouteMatch
	for _, match := range httpRouteGroup.Spec.Matches {
		httpRouteMatch := trafficpolicy.HTTPRouteMatch{
			Path:          match.PathRegex,
			PathMatchType: trafficpolicy.PathMatchRegex,
			Methods:       match.Methods,
			Headers:       match.Headers,
		}

		// When pathRegex and/or methods are not defined, they should be wildcarded
		if httpRouteMatch.Path == "" {
			httpRouteMatch.Path = constants.RegexMatchAll
		}
		if len(httpRouteMatch.Methods) == 0 {
			httpRouteMatch.Methods = []string{constants.WildcardHTTPMethod}
		}

		matches = append(matches, httpRouteMatch)
	}

	return matches
}
