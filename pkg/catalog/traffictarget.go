package catalog

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

// ListInboundServiceIdentities lists the downstream service identities that are allowed to connect to the given service identity
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListInboundServiceIdentities(upstream identity.ServiceIdentity) []identity.ServiceIdentity {
	return mc.getAllowedDirectionalServiceAccounts(upstream, inbound)
}

// ListOutboundServiceIdentities lists the upstream service identities the given service identity are allowed to connect to
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListOutboundServiceIdentities(downstream identity.ServiceIdentity) []identity.ServiceIdentity {
	return mc.getAllowedDirectionalServiceAccounts(downstream, outbound)
}

// ListInboundTrafficTargetsWithRoutes returns a list traffic target objects composed of its routes for the given destination service account
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListInboundTrafficTargetsWithRoutes(upstream identity.ServiceIdentity) ([]trafficpolicy.TrafficTargetWithRoutes, error) {
	var trafficTargets []trafficpolicy.TrafficTargetWithRoutes

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return nil, nil
	}

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		destinationSvcIdentity := trafficTargetIdentityToSvcAccount(t.Spec.Destination).ToServiceIdentity()
		if destinationSvcIdentity != upstream {
			continue
		}

		destinationIdentity := trafficTargetIdentityToServiceIdentity(t.Spec.Destination)

		// Create a traffic target for this destination identity
		trafficTarget := trafficpolicy.TrafficTargetWithRoutes{
			Name:        fmt.Sprintf("%s/%s", t.Namespace, t.Name),
			Destination: destinationIdentity,
		}

		// Source identifies for this traffic target
		var sourceIdentities []identity.ServiceIdentity
		for _, source := range t.Spec.Sources {
			srcIdentity := trafficTargetIdentityToServiceIdentity(source)
			sourceIdentities = append(sourceIdentities, srcIdentity)
		}
		trafficTarget.Sources = sourceIdentities

		// TCP routes for this traffic target
		if tcpRouteMatches, err := mc.getTCPRouteMatchesFromTrafficTarget(*t); err != nil {
			log.Error().Err(err).Msgf("Error fetching TCP Routes for TrafficTarget %s/%s", t.Namespace, t.Name)
		} else {
			// Add this traffic target to the final list
			trafficTarget.TCPRouteMatches = tcpRouteMatches
			trafficTargets = append(trafficTargets, trafficTarget)
		}
	}

	return trafficTargets, nil
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) getAllowedDirectionalServiceAccounts(svcIdentity identity.ServiceIdentity, direction trafficDirection) []identity.ServiceIdentity {
	svcAccount := svcIdentity.ToK8sServiceAccount()
	allowed := mapset.NewSet()

	allTrafficTargets := mc.meshSpec.ListTrafficTargets()
	for _, trafficTarget := range allTrafficTargets {
		spec := trafficTarget.Spec

		if spec.Destination.Kind != smi.ServiceAccountKind {
			// Destination kind is not valid
			log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidDestinationKind)).
				Msgf("Applied TrafficTarget policy %s has invalid Destination kind: %s", trafficTarget.Name, spec.Destination.Kind)
			continue
		}

		// For inbound direction, match TrafficTargets with destination corresponding to the given service account
		if direction == inbound {
			if spec.Destination.Name != svcAccount.Name || spec.Destination.Namespace != svcAccount.Namespace {
				// This TrafficTarget has a destination that does not match the given service account, ignore it
				continue
			}
			for _, source := range spec.Sources {
				if source.Kind != smi.ServiceAccountKind {
					// Destination kind is not valid
					log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidSourceKind)).
						Msgf("Applied TrafficTarget policy %s has invalid Source kind: %s", trafficTarget.Name, spec.Destination.Kind)
					continue
				}

				allowed.Add(trafficTargetIdentityToSvcAccount(source))
			}
		}

		// For outbound direction, match TrafficTargets with source corresponding to the given service account
		if direction == outbound {
			for _, source := range spec.Sources {
				if source.Kind != smi.ServiceAccountKind {
					// Destination kind is not valid
					log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidSourceKind)).
						Msgf("Applied TrafficTarget policy %s has invalid Source kind: %s", trafficTarget.Name, spec.Destination.Kind)
					continue
				}

				if source.Name != svcAccount.Name || source.Namespace != svcAccount.Namespace {
					// This TrafficTarget source does not match the given service account, ignore it
					continue
				}

				allowed.Add(trafficTargetIdentityToSvcAccount(spec.Destination))
			}
		}
	}

	var allowedSvcIdentities []identity.ServiceIdentity
	for svcAccount := range allowed.Iter() {
		allowedSvcIdentities = append(allowedSvcIdentities, svcAccount.(identity.K8sServiceAccount).ToServiceIdentity())
	}

	return allowedSvcIdentities
}

func trafficTargetIdentityToSvcAccount(identitySubject smiAccess.IdentityBindingSubject) identity.K8sServiceAccount {
	return identity.K8sServiceAccount{
		Name:      identitySubject.Name,
		Namespace: identitySubject.Namespace,
	}
}

// trafficTargetIdentityToServiceIdentity returns an identity of the form <namespace>/<service-account>
func trafficTargetIdentityToServiceIdentity(identitySubject smiAccess.IdentityBindingSubject) identity.ServiceIdentity {
	svcAccount := trafficTargetIdentityToSvcAccount(identitySubject)
	return identity.GetKubernetesServiceIdentity(svcAccount, identity.ClusterLocalTrustDomain)
}

// trafficTargetIdentitiesToSvcAccounts returns a list of Service Accounts from the given list of identities from a Traffic Target
func trafficTargetIdentitiesToSvcAccounts(identities []smiAccess.IdentityBindingSubject) []identity.K8sServiceAccount {
	serviceAccountsMap := map[identity.K8sServiceAccount]bool{}

	for _, id := range identities {
		sa := trafficTargetIdentityToSvcAccount(id)
		serviceAccountsMap[sa] = true
	}

	var serviceAccounts []identity.K8sServiceAccount
	for k := range serviceAccountsMap {
		serviceAccounts = append(serviceAccounts, k)
	}

	return serviceAccounts
}

func (mc *MeshCatalog) getTCPRouteMatchesFromTrafficTarget(trafficTarget smiAccess.TrafficTarget) ([]trafficpolicy.TCPRouteMatch, error) {
	var matches []trafficpolicy.TCPRouteMatch

	for _, rule := range trafficTarget.Spec.Rules {
		if rule.Kind != smi.TCPRouteKind {
			continue
		}

		// A route referenced in a traffic target must belong to the same namespace as the traffic target
		tcpRouteName := fmt.Sprintf("%s/%s", trafficTarget.Namespace, rule.Name)

		tcpRoute := mc.meshSpec.GetTCPRoute(tcpRouteName)
		if tcpRoute == nil {
			return nil, errNoTrafficSpecFoundForTrafficPolicy
		}

		tcpRouteMatch := trafficpolicy.TCPRouteMatch{
			Ports: tcpRoute.Spec.Matches.Ports,
		}
		matches = append(matches, tcpRouteMatch)
	}

	return matches, nil
}
