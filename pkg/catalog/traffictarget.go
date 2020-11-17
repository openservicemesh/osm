package catalog

import (
	mapset "github.com/deckarep/golang-set"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"

	"github.com/openservicemesh/osm/pkg/service"
)

const (
	serviceAccountKind = "ServiceAccount"
)

// ListAllowedInboundServiceAccounts lists the downstream service accounts that can connect to the given upstream service account
func (mc *MeshCatalog) ListAllowedInboundServiceAccounts(upstream service.K8sServiceAccount) ([]service.K8sServiceAccount, error) {
	return mc.getAllowedDirectionalServiceAccounts(upstream, inbound)
}

// ListAllowedOutboundServiceAccounts lists the upstream service accounts the given downstream service account can connect to
func (mc *MeshCatalog) ListAllowedOutboundServiceAccounts(downstream service.K8sServiceAccount) ([]service.K8sServiceAccount, error) {
	return mc.getAllowedDirectionalServiceAccounts(downstream, outbound)
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

func trafficTargetIdentityToSvcAccount(identity smiAccess.IdentityBindingSubject) service.K8sServiceAccount {
	return service.K8sServiceAccount{
		Name:      identity.Name,
		Namespace: identity.Namespace,
	}
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
