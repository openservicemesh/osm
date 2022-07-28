package catalog

import (
	"fmt"
	"net"
	"strings"

	mapset "github.com/deckarep/golang-set"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	// singeIPPrefixLen is the IP prefix length for a single IP address
	singeIPPrefixLen = "/32"
)

// GetIngressTrafficPolicy returns the ingress traffic policy for the given mesh service
// Depending on if the IngressBackend API is enabled, the policies will be generated either from the IngressBackend
// or Kubernetes Ingress API.
func (mc *MeshCatalog) GetIngressTrafficPolicy(svc service.MeshService) (*trafficpolicy.IngressTrafficPolicy, error) {
	ingressBackendPolicy := mc.policyController.GetIngressBackendPolicy(svc)
	if ingressBackendPolicy == nil {
		log.Trace().Msgf("Did not find IngressBackend policy for service %s", svc)
		return nil, nil
	}

	// The status field will be updated after the policy is processed.
	// Note: The original pointer returned by cache.Store must not be modified for thread safety.
	ingressBackendWithStatus := *ingressBackendPolicy

	var trafficRoutingRules []*trafficpolicy.Rule
	// The ingress backend deals with principals (not identities). Principals have the trust domain included.
	sourcePrincipals := mapset.NewSet()
	var trafficMatches []*trafficpolicy.IngressTrafficMatch
	for _, backend := range ingressBackendPolicy.Spec.Backends {
		if backend.Name != svc.Name || backend.Port.Number != int(svc.TargetPort) {
			continue
		}

		trafficMatch := &trafficpolicy.IngressTrafficMatch{
			Name:                     service.IngressTrafficMatchName(svc.Name, svc.Namespace, uint16(backend.Port.Number), backend.Port.Protocol),
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
				sourceMeshSvc := service.MeshService{
					Name:      source.Name,
					Namespace: source.Namespace,
				}
				endpoints := mc.listEndpointsForService(sourceMeshSvc)
				if len(endpoints) == 0 {
					ingressBackendWithStatus.Status = policyV1alpha1.IngressBackendStatus{
						CurrentStatus: "error",
						Reason:        fmt.Sprintf("endpoints not found for service %s/%s", source.Namespace, source.Name),
					}
					if _, err := mc.kubeController.UpdateStatus(&ingressBackendWithStatus); err != nil {
						log.Error().Err(err).Msg("Error updating status for IngressBackend")
					}
					return nil, fmt.Errorf("Could not list endpoints of the source service %s/%s specified in the IngressBackend %s/%s",
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
				if backend.TLS.SkipClientCertValidation {
					sourcePrincipals.Add(identity.WildcardServiceIdentity.String())
				} else {
					sourcePrincipals.Add(source.Name)
				}
			}
		}

		// If this ingress is corresponding to an HTTP port, wildcard the downstream's identity
		// because the identity cannot be verified for HTTP traffic. HTTP based ingress can
		// restrict downstreams based on their endpoint's IP address.
		if strings.EqualFold(backend.Port.Protocol, constants.ProtocolHTTP) {
			sourcePrincipals.Add(identity.WildcardPrincipal)
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
			AllowedPrincipals: sourcePrincipals,
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
