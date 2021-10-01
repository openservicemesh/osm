package catalog

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	networkingV1 "k8s.io/api/networking/v1"
	networkingV1beta1 "k8s.io/api/networking/v1beta1"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// prefixMatchPathElementsRegex is the regex pattern used to match zero or one grouping of path elements.
	// A path element is of the form /p, /p1/p2, /p1/p2/p3 etc.
	// This regex pattern is used to match paths in a way that is compatible with Kubernetes ingress requirements
	// for Prefix path type, where the prefix must be an element wise prefix and not a string prefix.
	// Ref: https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types
	// It is used to regex match paths such that request /foo matches /foo and /foo/bar, but not /foobar.
	prefixMatchPathElementsRegex = `(/.*)?$`

	// commonRegexChars is a string comprising of characters commonly used in a regex
	// It is used to guess whether a path specified appears as a regex.
	// It is used as a fallback to match ingress paths whose PathType is set to be ImplementationSpecific.
	commonRegexChars = `^$*+[]%|`

	// singeIPPrefixLen is the IP prefix length for a single IP address
	singeIPPrefixLen = "/32"
)

// Ensure the regex pattern for prefix matching for path elements compiles
var _ = regexp.MustCompile(prefixMatchPathElementsRegex)

// GetIngressTrafficPolicy returns the ingress traffic policy for the given mesh service
// Depending on if the IngressBackend API is enabled, the policies will be generated either from the IngressBackend
// or Kubernetes Ingress API.
func (mc *MeshCatalog) GetIngressTrafficPolicy(svc service.MeshService) (*trafficpolicy.IngressTrafficPolicy, error) {
	if mc.configurator.GetFeatureFlags().EnableIngressBackendPolicy {
		return mc.getIngressTrafficPolicy(svc)
	}

	return mc.getIngressTrafficPolicyFromK8s(svc)
}

// getIngressTrafficPolicy returns the ingress traffic policy for the given mesh service from corresponding IngressBackend resource
func (mc *MeshCatalog) getIngressTrafficPolicy(svc service.MeshService) (*trafficpolicy.IngressTrafficPolicy, error) {
	ingressBackendPolicy := mc.policyController.GetIngressBackendPolicy(svc)
	if ingressBackendPolicy == nil {
		log.Trace().Msgf("Did not find IngressBackend policy for service %s", svc)
		return nil, nil
	}

	// The status field will be updated after the policy is processed.
	// Note: The original pointer returned by cache.Store must not be modified for thread safety.
	ingressBackendWithStatus := *ingressBackendPolicy

	var trafficRoutingRules []*trafficpolicy.Rule
	sourceServiceIdentities := mapset.NewSet()
	var trafficMatches []*trafficpolicy.IngressTrafficMatch
	for _, backend := range ingressBackendPolicy.Spec.Backends {
		if backend.Name != svc.Name || backend.Port.Number != int(svc.TargetPort) {
			continue
		}

		trafficMatch := &trafficpolicy.IngressTrafficMatch{
			Name:                     fmt.Sprintf("ingress_%s_%d_%s", svc, backend.Port.Number, backend.Port.Protocol),
			Port:                     uint32(backend.Port.Number),
			Protocol:                 backend.Port.Protocol,
			ServerNames:              backend.TLS.SNIHosts,
			SkipClientCertValidation: backend.TLS.SkipClientCertValidation,
		}

		var sourceIPRanges []string
		sourceIPSet := mapset.NewSet() // Used to avoid duplicate IP ranges
		for _, source := range ingressBackendPolicy.Spec.Sources {
			switch source.Kind {
			case policyV1alpha1.KindService:
				sourceMeshSvc := service.MeshService{Name: source.Name, Namespace: source.Namespace}
				endpoints := mc.listEndpointsForService(sourceMeshSvc)
				if len(endpoints) == 0 {
					ingressBackendWithStatus.Status = policyV1alpha1.IngressBackendStatus{
						CurrentStatus: "error",
						Reason:        fmt.Sprintf("endpoints not found for service %s/%s", source.Namespace, source.Name),
					}
					if _, err := mc.kubeController.UpdateStatus(&ingressBackendWithStatus); err != nil {
						log.Error().Err(err).Msg("Error updating status for IngressBackend")
					}
					return nil, errors.Errorf("Could not list endpoints of the source service %s/%s specified in the IngressBackend %s/%s",
						source.Namespace, source.Name, ingressBackendPolicy.Namespace, ingressBackendPolicy.Name)
				}

				for _, ep := range endpoints {
					sourceCIDR := ep.IP.String() + singeIPPrefixLen
					if sourceIPSet.Add(sourceCIDR) {
						sourceIPRanges = append(sourceIPRanges, sourceCIDR)
					}
				}

			case policyV1alpha1.KindIPRange:
				if _, _, err := net.ParseCIDR(source.Name); err != nil {
					// This should not happen because the validating webhook will prevent it. This check has
					// been added as a safety net to prevent invalid configs.
					log.Error().Err(err).Msgf("Invalid IP address range specified in IngressBackend %s/%s: %s",
						ingressBackendPolicy.Namespace, ingressBackendPolicy.Name, source.Name)
					continue
				}
				sourceIPRanges = append(sourceIPRanges, source.Name)

			case policyV1alpha1.KindAuthenticatedPrincipal:
				var sourceIdentity identity.ServiceIdentity
				if backend.TLS.SkipClientCertValidation {
					sourceIdentity = identity.WildcardServiceIdentity
				} else {
					sourceIdentity = identity.ServiceIdentity(source.Name)
				}
				sourceServiceIdentities.Add(sourceIdentity)
			}
		}

		// If this ingress is corresponding to an HTTP port, wildcard the downstream's identity
		// because the identity cannot be verified for HTTP traffic. HTTP based ingress can
		// restrict downstreams based on their endpoint's IP address.
		if strings.EqualFold(backend.Port.Protocol, constants.ProtocolHTTP) {
			sourceServiceIdentities.Add(identity.WildcardServiceIdentity)
		}

		trafficMatch.SourceIPRanges = sourceIPRanges
		trafficMatches = append(trafficMatches, trafficMatch)

		// Build the routing rule for this backend and source combination.
		// Currently IngressBackend only supports a wildcard HTTP route. The
		// 'Matches' field in the spec can be used to extend this to perform
		// stricter enforcement.
		backendCluster := service.WeightedCluster{
			ClusterName: service.ClusterName(svc.EnvoyLocalClusterName()),
			Weight:      constants.ClusterWeightAcceptAll,
		}
		routingRule := &trafficpolicy.Rule{
			Route: trafficpolicy.RouteWeightedClusters{
				HTTPRouteMatch:   trafficpolicy.WildCardRouteMatch,
				WeightedClusters: mapset.NewSet(backendCluster),
			},
			AllowedServiceIdentities: sourceServiceIdentities,
		}
		trafficRoutingRules = append(trafficRoutingRules, routingRule)
	}

	if len(trafficMatches) == 0 {
		// Since no trafficMatches exist for this IngressBackend config, it implies that the given
		// MeshService does not map to this IngressBackend config.
		log.Debug().Msgf("No ingress traffic matches exist for MeshService %s, no ingress config required", svc.EnvoyLocalClusterName())
		return nil, nil
	}

	ingressBackendWithStatus.Status = policyV1alpha1.IngressBackendStatus{
		CurrentStatus: "committed",
		Reason:        "successfully committed by the system",
	}
	if _, err := mc.kubeController.UpdateStatus(&ingressBackendWithStatus); err != nil {
		log.Error().Err(err).Msg("Error updating status for IngressBackend")
	}

	// Create an inbound traffic policy from the routing rules
	// TODO(#3779): Implement HTTP route matching from IngressBackend.Spec.Matches
	httpRoutePolicy := &trafficpolicy.InboundTrafficPolicy{
		Name:      fmt.Sprintf("%s_from_%s", svc, ingressBackendPolicy.Name),
		Hostnames: []string{"*"},
		Rules:     trafficRoutingRules,
	}

	return &trafficpolicy.IngressTrafficPolicy{
		TrafficMatches:    trafficMatches,
		HTTPRoutePolicies: []*trafficpolicy.InboundTrafficPolicy{httpRoutePolicy},
	}, nil
}

// getIngressTrafficPolicyFromK8s returns the ingress traffic policy for the given mesh service from the corresponding k8s Ingress resource
// TODO: DEPRECATE once IngressBackend API is the default for configuring an ingress backend.
func (mc *MeshCatalog) getIngressTrafficPolicyFromK8s(svc service.MeshService) (*trafficpolicy.IngressTrafficPolicy, error) {
	if svc.Protocol != constants.ProtocolHTTP {
		// Only HTTP ports can accept traffic using k8s Ingress
		return nil, nil
	}

	httpRoutePolicies, err := mc.getIngressPoliciesFromK8s(svc)
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving ingress HTTP routing policies for service %s from Kubernetes", svc)
	}

	if httpRoutePolicies == nil {
		// There are no routes for ingress, which implies ingress does not need to be configured
		return nil, nil
	}

	enableHTTPSIngress := mc.configurator.UseHTTPSIngress()
	var trafficMatches []*trafficpolicy.IngressTrafficMatch
	trafficMatch := &trafficpolicy.IngressTrafficMatch{
		Port: uint32(svc.TargetPort),
	}

	if enableHTTPSIngress {
		// Configure 2 taffic matches for HTTPS ingress (TLS):
		// 1. Without SNI: to match clients that don't set the SNI
		// 2. With SNI: to match clients that set the SNI

		trafficMatch.Name = fmt.Sprintf("ingress_%s_%d_%s", svc, svc.TargetPort, constants.ProtocolHTTPS)
		trafficMatch.Protocol = constants.ProtocolHTTPS
		trafficMatch.SkipClientCertValidation = true
		trafficMatches = append(trafficMatches, trafficMatch)

		trafficMatchWithSNI := *trafficMatch
		trafficMatchWithSNI.Name = fmt.Sprintf("ingress_%s_%d_%s_with_sni", svc, svc.TargetPort, constants.ProtocolHTTPS)
		trafficMatchWithSNI.ServerNames = []string{svc.ServerName()}
		trafficMatches = append(trafficMatches, &trafficMatchWithSNI)
	} else {
		trafficMatch.Name = fmt.Sprintf("ingress_%s_%d_%s", svc, svc.TargetPort, constants.ProtocolHTTP)
		trafficMatch.Protocol = constants.ProtocolHTTP
		trafficMatches = append(trafficMatches, trafficMatch)
	}

	return &trafficpolicy.IngressTrafficPolicy{
		TrafficMatches:    trafficMatches,
		HTTPRoutePolicies: httpRoutePolicies,
	}, nil
}

// getIngressPoliciesFromK8s returns a list of inbound traffic policies for a service as defined in observed ingress k8s resources.
func (mc *MeshCatalog) getIngressPoliciesFromK8s(svc service.MeshService) ([]*trafficpolicy.InboundTrafficPolicy, error) {
	var inboundTrafficPolicies []*trafficpolicy.InboundTrafficPolicy

	// Build policies for ingress v1
	if v1Policies, err := mc.getIngressPoliciesNetworkingV1(svc); err != nil {
		log.Error().Err(err).Msgf("Error building inbound ingress v1 inbound policies for service %s", svc)
	} else {
		inboundTrafficPolicies = append(inboundTrafficPolicies, v1Policies...)
	}

	// Build policies for ingress v1beta1
	if v1beta1Policies, err := mc.getIngressPoliciesNetworkingV1beta1(svc); err != nil {
		log.Error().Err(err).Msgf("Error building inbound ingress v1beta inbound policies for service %s", svc)
	} else {
		inboundTrafficPolicies = append(inboundTrafficPolicies, v1beta1Policies...)
	}

	return inboundTrafficPolicies, nil
}

func getIngressTrafficPolicyName(name, namespace, host string) string {
	policyName := fmt.Sprintf("%s.%s|%s", name, namespace, host)
	return policyName
}

// getIngressPoliciesNetworkingV1beta1 returns the list of inbound traffic policies associated with networking.k8s.io/v1beta1 ingress resources for the given service
func (mc *MeshCatalog) getIngressPoliciesNetworkingV1beta1(svc service.MeshService) ([]*trafficpolicy.InboundTrafficPolicy, error) {
	var inboundIngressPolicies []*trafficpolicy.InboundTrafficPolicy

	ingresses, err := mc.ingressMonitor.GetIngressNetworkingV1beta1(svc)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources for service %s", svc)
		return inboundIngressPolicies, err
	}
	if len(ingresses) == 0 {
		log.Trace().Msgf("No ingress resources found for service %s", svc)
		return inboundIngressPolicies, err
	}

	ingressWeightedCluster := service.WeightedCluster{
		ClusterName: service.ClusterName(svc.EnvoyLocalClusterName()),
		Weight:      constants.ClusterWeightAcceptAll,
	}

	for _, ingress := range ingresses {
		if ingress.Spec.Backend != nil && ingress.Spec.Backend.ServiceName == svc.Name {
			wildcardIngressPolicy := trafficpolicy.NewInboundTrafficPolicy(getIngressTrafficPolicyName(ingress.ObjectMeta.Name, ingress.ObjectMeta.Namespace, constants.WildcardHTTPMethod), []string{constants.WildcardHTTPMethod})
			wildcardIngressPolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(trafficpolicy.WildCardRouteMatch, []service.WeightedCluster{ingressWeightedCluster}), identity.WildcardServiceIdentity)
			inboundIngressPolicies = trafficpolicy.MergeInboundPolicies(DisallowPartialHostnamesMatch, inboundIngressPolicies, wildcardIngressPolicy)
		}

		for _, rule := range ingress.Spec.Rules {
			domain := rule.Host
			if domain == "" {
				domain = constants.WildcardHTTPMethod
			}
			ingressPolicy := trafficpolicy.NewInboundTrafficPolicy(getIngressTrafficPolicyName(ingress.ObjectMeta.Name, ingress.ObjectMeta.Namespace, domain), []string{domain})

			for _, ingressPath := range rule.HTTP.Paths {
				if ingressPath.Backend.ServiceName != svc.Name {
					continue
				}

				httpRouteMatch := trafficpolicy.HTTPRouteMatch{
					Methods: []string{constants.WildcardHTTPMethod},
				}

				// Default ingress path type to PathTypeImplementationSpecific if unspecified
				pathType := networkingV1beta1.PathTypeImplementationSpecific
				if ingressPath.PathType != nil {
					pathType = *ingressPath.PathType
				}

				switch pathType {
				case networkingV1beta1.PathTypeExact:
					// Exact match
					// Request /foo matches path /foo, not /foobar or /foo/bar
					httpRouteMatch.Path = ingressPath.Path
					httpRouteMatch.PathMatchType = trafficpolicy.PathMatchExact

				case networkingV1beta1.PathTypePrefix:
					// Element wise prefix match
					// Request /foo matches path /foo and /foo/bar, not /foobar
					if ingressPath.Path == "/" {
						// A wildcard path '/' for Prefix pathType must be matched
						// as a string based prefix match, ie. path '/' should
						// match any path in the request.
						httpRouteMatch.Path = ingressPath.Path
						httpRouteMatch.PathMatchType = trafficpolicy.PathMatchPrefix
					} else {
						// Non-wildcard path of the form '/path' must be matched as a
						// regex match to meet k8s Ingress API requirement of element-wise
						// prefix matching.
						// There is also the requirement for prefix /foo/ to match /foo
						// based on k8s API interpretation of element-wise matching, so
						// account for this case by trimming trailing '/'.
						path := strings.TrimRight(ingressPath.Path, "/")
						httpRouteMatch.Path = path + prefixMatchPathElementsRegex
						httpRouteMatch.PathMatchType = trafficpolicy.PathMatchRegex
					}

				case networkingV1beta1.PathTypeImplementationSpecific:
					httpRouteMatch.Path = ingressPath.Path
					// If the path looks like a regex, use regex matching.
					// Else use string based prefix matching.
					if strings.ContainsAny(ingressPath.Path, commonRegexChars) {
						// Path contains regex characters, use regex matching for the path
						// Request /foo/bar matches path /foo.*
						httpRouteMatch.PathMatchType = trafficpolicy.PathMatchRegex
					} else {
						// String based prefix path matching
						// Request /foo matches /foo/bar and /foobar
						httpRouteMatch.PathMatchType = trafficpolicy.PathMatchPrefix
					}

				default:
					log.Error().Msgf("Invalid pathType=%s unspecified for path %s in ingress resource %s/%s, ignoring this path", *ingressPath.PathType, ingressPath.Path, ingress.Namespace, ingress.Name)
					continue
				}

				ingressPolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(httpRouteMatch, []service.WeightedCluster{ingressWeightedCluster}), identity.WildcardServiceIdentity)
			}

			// Only create an ingress policy if the ingress policy resulted in valid rules
			if len(ingressPolicy.Rules) > 0 {
				inboundIngressPolicies = trafficpolicy.MergeInboundPolicies(DisallowPartialHostnamesMatch, inboundIngressPolicies, ingressPolicy)
			}
		}
	}
	return inboundIngressPolicies, nil
}

// getIngressPoliciesNetworkingV1 returns the list of inbound traffic policies associated with networking.k8s.io/v1 ingress resources for the given service
func (mc *MeshCatalog) getIngressPoliciesNetworkingV1(svc service.MeshService) ([]*trafficpolicy.InboundTrafficPolicy, error) {
	var inboundIngressPolicies []*trafficpolicy.InboundTrafficPolicy

	ingresses, err := mc.ingressMonitor.GetIngressNetworkingV1(svc)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get ingress resources for service %s", svc)
		return inboundIngressPolicies, err
	}
	if len(ingresses) == 0 {
		log.Trace().Msgf("No ingress resources found for service %s", svc)
		return inboundIngressPolicies, err
	}

	ingressWeightedCluster := service.WeightedCluster{
		ClusterName: service.ClusterName(svc.EnvoyLocalClusterName()),
		Weight:      constants.ClusterWeightAcceptAll,
	}

	for _, ingress := range ingresses {
		if ingress.Spec.DefaultBackend != nil && ingress.Spec.DefaultBackend.Service.Name == svc.Name {
			wildcardIngressPolicy := trafficpolicy.NewInboundTrafficPolicy(getIngressTrafficPolicyName(ingress.ObjectMeta.Name, ingress.ObjectMeta.Namespace, constants.WildcardHTTPMethod), []string{constants.WildcardHTTPMethod})
			wildcardIngressPolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(trafficpolicy.WildCardRouteMatch, []service.WeightedCluster{ingressWeightedCluster}), identity.WildcardServiceIdentity)
			inboundIngressPolicies = trafficpolicy.MergeInboundPolicies(DisallowPartialHostnamesMatch, inboundIngressPolicies, wildcardIngressPolicy)
		}

		for _, rule := range ingress.Spec.Rules {
			domain := rule.Host
			if domain == "" {
				domain = constants.WildcardHTTPMethod
			}
			ingressPolicy := trafficpolicy.NewInboundTrafficPolicy(getIngressTrafficPolicyName(ingress.ObjectMeta.Name, ingress.ObjectMeta.Namespace, domain), []string{domain})

			for _, ingressPath := range rule.HTTP.Paths {
				if ingressPath.Backend.Service.Name != svc.Name {
					continue
				}

				httpRouteMatch := trafficpolicy.HTTPRouteMatch{
					Methods: []string{constants.WildcardHTTPMethod},
				}

				// Default ingress path type to PathTypeImplementationSpecific if unspecified
				pathType := networkingV1.PathTypeImplementationSpecific
				if ingressPath.PathType != nil {
					pathType = *ingressPath.PathType
				}

				switch pathType {
				case networkingV1.PathTypeExact:
					// Exact match
					// Request /foo matches path /foo, not /foobar or /foo/bar
					httpRouteMatch.Path = ingressPath.Path
					httpRouteMatch.PathMatchType = trafficpolicy.PathMatchExact

				case networkingV1.PathTypePrefix:
					// Element wise prefix match
					// Request /foo matches path /foo and /foo/bar, not /foobar
					if ingressPath.Path == "/" {
						// A wildcard path '/' for Prefix pathType must be matched
						// as a string based prefix match, ie. path '/' should
						// match any path in the request.
						httpRouteMatch.Path = ingressPath.Path
						httpRouteMatch.PathMatchType = trafficpolicy.PathMatchPrefix
					} else {
						// Non-wildcard path of the form '/path' must be matched as a
						// regex match to meet k8s Ingress API requirement of element-wise
						// prefix matching.
						// There is also the requirement for prefix /foo/ to match /foo
						// based on k8s API interpretation of element-wise matching, so
						// account for this case by trimming trailing '/'.
						path := strings.TrimRight(ingressPath.Path, "/")
						httpRouteMatch.Path = path + prefixMatchPathElementsRegex
						httpRouteMatch.PathMatchType = trafficpolicy.PathMatchRegex
					}

				case networkingV1.PathTypeImplementationSpecific:
					httpRouteMatch.Path = ingressPath.Path
					// If the path looks like a regex, use regex matching.
					// Else use string based prefix matching.
					if strings.ContainsAny(ingressPath.Path, commonRegexChars) {
						// Path contains regex characters, use regex matching for the path
						// Request /foo/bar matches path /foo.*
						httpRouteMatch.PathMatchType = trafficpolicy.PathMatchRegex
					} else {
						// String based prefix path matching
						// Request /foo matches /foo/bar and /foobar
						httpRouteMatch.PathMatchType = trafficpolicy.PathMatchPrefix
					}

				default:
					log.Error().Msgf("Invalid pathType=%s unspecified for path %s in ingress resource %s/%s, ignoring this path", *ingressPath.PathType, ingressPath.Path, ingress.Namespace, ingress.Name)
					continue
				}

				ingressPolicy.AddRule(*trafficpolicy.NewRouteWeightedCluster(httpRouteMatch, []service.WeightedCluster{ingressWeightedCluster}), identity.WildcardServiceIdentity)
			}

			// Only create an ingress policy if the ingress policy resulted in valid rules
			if len(ingressPolicy.Rules) > 0 {
				inboundIngressPolicies = trafficpolicy.MergeInboundPolicies(DisallowPartialHostnamesMatch, inboundIngressPolicies, ingressPolicy)
			}
		}
	}
	return inboundIngressPolicies, nil
}
