package catalog

import (
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// ListAllowedUpstreamEndpointsForService returns the list of endpoints over which the downstream client identity
// is allowed access the upstream service
func (mc *MeshCatalog) ListAllowedUpstreamEndpointsForService(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService) []endpoint.Endpoint {
	outboundEndpoints := mc.ListEndpointsForService(upstreamSvc)
	if len(outboundEndpoints) == 0 {
		return nil
	}

	if mc.GetMeshConfig().Spec.Traffic.EnablePermissiveTrafficPolicyMode {
		return outboundEndpoints
	}

	// In SMI mode, the endpoints for an upstream service must be filtered based on the service account
	// associated with the endpoint. Only endpoints associated with authorized service accounts as referenced
	// in SMI TrafficTarget resources should be returned.
	//
	// The following code filters the upstream service's endpoints for this purpose.
	outboundEndpointsSet := make(map[string][]endpoint.Endpoint)
	for _, ep := range outboundEndpoints {
		ipStr := ep.IP.String()
		outboundEndpointsSet[ipStr] = append(outboundEndpointsSet[ipStr], ep)
	}

	// allowedEndpoints comprises of only those endpoints from outboundEndpoints that matches the endpoints from ListEndpointsForIdentity
	// i.e. only those intersecting endpoints are taken into cosideration
	var allowedEndpoints []endpoint.Endpoint
	for _, destSvcIdentity := range mc.ListOutboundServiceIdentities(downstreamIdentity) {
		for _, ep := range mc.ListEndpointsForIdentity(destSvcIdentity) {
			epIPStr := ep.IP.String()
			// check if endpoint IP is allowed
			if _, ok := outboundEndpointsSet[epIPStr]; ok {
				// add all allowed endpoints on the pod to result list
				allowedEndpoints = append(allowedEndpoints, outboundEndpointsSet[epIPStr]...)
			}
		}
	}

	return allowedEndpoints
}
