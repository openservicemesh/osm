package catalog

import (
	"github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// getRetryPolicy returns the RetryPolicySpec for the given downstream identity and upstream service
// TODO: Add support for wildcard destinations
func (mc *MeshCatalog) getRetryPolicy(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService) *v1alpha1.RetryPolicySpec {
	if !mc.configurator.GetFeatureFlags().EnableRetryPolicy {
		log.Trace().Msgf("Retry policy flag not enabled")
		return nil
	}
	src := downstreamIdentity.ToK8sServiceAccount()

	// List the retry policies for the source
	retryPolicies := mc.policyController.ListRetryPolicies(src)
	if retryPolicies == nil {
		log.Trace().Msgf("Did not find retry policy for downstream service %s", src)
		return nil
	}

	for _, retryCRD := range retryPolicies {
		for _, dest := range retryCRD.Spec.Destinations {
			if dest.Kind != "Service" {
				log.Error().Msgf("Retry policy destinations must be a service: %s is a %s", dest, dest.Kind)
				continue
			}
			destMeshSvc := service.MeshService{Name: dest.Name, Namespace: dest.Namespace}
			if upstreamSvc == destMeshSvc {
				// Will return retry policy that applies to the specific upstream service
				return &retryCRD.Spec.RetryPolicy
			}
		}
	}

	log.Trace().Msgf("Could not find retry policy for source %s and destination %s", src, upstreamSvc)
	return nil
}
