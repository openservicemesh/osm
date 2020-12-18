package catalog

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/trafficpolicy"
)

const (
	serviceAccountKind = "ServiceAccount"
	tcpRouteKind       = "TCPRoute"
)

// ListAllowedInboundServiceAccounts lists the downstream service accounts that can connect to the given upstream service account
func (mc *MeshCatalog) ListAllowedInboundServiceAccounts(upstream service.K8sServiceAccount) ([]service.K8sServiceAccount, error) {
	return mc.getAllowedDirectionalServiceAccounts(upstream, inbound)
}

// ListAllowedOutboundServiceAccounts lists the upstream service accounts the given downstream service account can connect to
func (mc *MeshCatalog) ListAllowedOutboundServiceAccounts(downstream service.K8sServiceAccount) ([]service.K8sServiceAccount, error) {
	return mc.getAllowedDirectionalServiceAccounts(downstream, outbound)
}

// ListInboundTrafficTargetsWithRoutes returns a list traffic target objects componsed of its routes for the given destination identity
func (mc *MeshCatalog) ListInboundTrafficTargetsWithRoutes(upstreamIdentity identity.ServiceIdentity) ([]trafficpolicy.TrafficTargetWithRoutes, error) {
	var trafficTargets []trafficpolicy.TrafficTargetWithRoutes

	if mc.configurator.IsPermissiveTrafficPolicyMode() {
		return nil, nil
	}

	for _, t := range mc.meshSpec.ListTrafficTargets() { // loop through all traffic targets
		if !isValidTrafficTarget(t) {
			continue
		}

		destinationIdentity := trafficTargetIdentityToServiceIdentity(t.Spec.Destination)
		if destinationIdentity != upstreamIdentity {
			continue
		}

		// Create a traffic target for this destination indentity
		trafficTarget := trafficpolicy.TrafficTargetWithRoutes{
			Name:        t.Name,
			Destination: destinationIdentity,
		}

		// Source identies for this traffic target
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

func (mc *MeshCatalog) getAllowedDirectionalServiceAccounts(svcAccount service.K8sServiceAccount, direction trafficDirection) ([]service.K8sServiceAccount, error) {
	var allowedSvcAccounts []service.K8sServiceAccount
	allowed := mapset.NewSet()

	allTrafficTargets := mc.meshSpec.ListTrafficTargets()
	for _, trafficTarget := range allTrafficTargets {
		spec := trafficTarget.Spec

		if spec.Destination.Kind != serviceAccountKind {
			// Destination kind is not valid
			log.Error().Msgf("Applied TrafficTarget policy %s has invalid Destination kind: %s", trafficTarget.Name, spec.Destination.Kind)
			continue
		}

		// For inbound direction, match TrafficTargets with destination corresponding to the given service account
		if direction == inbound {
			if spec.Destination.Name != svcAccount.Name || spec.Destination.Namespace != svcAccount.Namespace {
				// This TrafficTarget has a destination that does not match the given service account, ignore it
				continue
			}
			for _, source := range spec.Sources {
				if source.Kind != serviceAccountKind {
					// Destination kind is not valid
					log.Error().Msgf("Applied TrafficTarget policy %s has invalid Source kind: %s", trafficTarget.Name, spec.Destination.Kind)
					continue
				}

				allowed.Add(trafficTargetIdentityToSvcAccount(source))
			}
		}

		// For outbound direction, match TrafficTargets with source corresponding to the given service account
		if direction == outbound {
			for _, source := range spec.Sources {
				if source.Kind != serviceAccountKind {
					// Destination kind is not valid
					log.Error().Msgf("Applied TrafficTarget policy %s has invalid Source kind: %s", trafficTarget.Name, spec.Destination.Kind)
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

	for svcAccount := range allowed.Iter() {
		allowedSvcAccounts = append(allowedSvcAccounts, svcAccount.(service.K8sServiceAccount))
	}

	return allowedSvcAccounts, nil
}

func trafficTargetIdentityToSvcAccount(identitySubject smiAccess.IdentityBindingSubject) service.K8sServiceAccount {
	return service.K8sServiceAccount{
		Name:      identitySubject.Name,
		Namespace: identitySubject.Namespace,
	}
}

func trafficTargetIdentityToServiceIdentity(identitySubject smiAccess.IdentityBindingSubject) identity.ServiceIdentity {
	return identity.ServiceIdentity(fmt.Sprintf("%s/%s", identitySubject.Namespace, identitySubject.Name))
}

// trafficTargetIdentitiesToSvcAccounts returns a list of Service Accounts from the given list of identities from a Traffic Target
func trafficTargetIdentitiesToSvcAccounts(identities []smiAccess.IdentityBindingSubject) []service.K8sServiceAccount {
	serviceAccountsMap := map[service.K8sServiceAccount]bool{}

	for _, id := range identities {
		sa := trafficTargetIdentityToSvcAccount(id)
		serviceAccountsMap[sa] = true
	}

	serviceAccounts := []service.K8sServiceAccount{}
	for k := range serviceAccountsMap {
		serviceAccounts = append(serviceAccounts, k)
	}

	return serviceAccounts
}

func (mc *MeshCatalog) getTCPRouteMatchesFromTrafficTarget(trafficTarget smiAccess.TrafficTarget) ([]trafficpolicy.TCPRouteMatch, error) {
	var matches []trafficpolicy.TCPRouteMatch

	for _, rule := range trafficTarget.Spec.Rules {
		if rule.Kind != tcpRouteKind {
			continue
		}

		// A route referenced in a traffic target must belong to the same namespace as the traffic target
		tcpRouteName := fmt.Sprintf("%s/%s", trafficTarget.Namespace, rule.Name)

		tcpRoute := mc.meshSpec.GetTCPRoute(tcpRouteName)
		if tcpRoute == nil {
			return nil, errNoTrafficSpecFoundForTrafficPolicy
		}

		// TODO(#1521): Create an actual TCP route match once v1alpha4 TCPRoute spec is available
		tcpRouteMatch := trafficpolicy.TCPRouteMatch{ /* allow all ports */ }
		matches = append(matches, tcpRouteMatch)
	}

	return matches, nil
}
