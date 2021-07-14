package catalog

import (
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// ListEndpointsForService returns the list of provider endpoints corresponding to a service
func (mc *MeshCatalog) listEndpointsForService(svc service.MeshService) ([]endpoint.Endpoint, error) {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		if ep := provider.ListEndpointsForService(svc); len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service=%s", provider.GetID(), svc)
		} else {
			endpoints = append(endpoints, ep...)
		}
	}
	return endpoints, nil
}

// GetResolvableServiceEndpoints returns the resolvable set of endpoint over which a service is accessible using its FQDN
func (mc *MeshCatalog) GetResolvableServiceEndpoints(svc service.MeshService) ([]endpoint.Endpoint, error) {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		ep, err := provider.GetResolvableEndpointsForService(svc)
		if err != nil {
			log.Error().Err(err).Msgf("[%s] Error getting endpoints for Service %s", provider.GetID(), svc)
			continue
		}
		if len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service=%s", provider.GetID(), svc)
			continue
		}
		endpoints = append(endpoints, ep...)
	}
	return endpoints, nil
}

// ListEndpointsForServiceIdentity returns a list of endpoints that belongs to an upstream service accounts
// from the given downstream identity's perspective
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListEndpointsForServiceIdentity(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService) ([]endpoint.Endpoint, error) {
	outboundEndpoints, err := mc.listEndpointsForService(upstreamSvc)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up endpoints for upstream service %s", upstreamSvc)
		return nil, err
	}
	outboundEndpointsSet := make(map[string][]endpoint.Endpoint)
	for _, ep := range outboundEndpoints {
		outboundEndpointsSet[ep.IP.String()] = append(outboundEndpointsSet[ep.IP.String()], ep)
	}

	destSvcIdentities, err := mc.ListOutboundServiceIdentities(downstreamIdentity)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up outbound service accounts for downstream identity %s", downstreamIdentity)
		return nil, err
	}

	// allowedEndpoints comprises of only those endpoints from outboundEndpoints that matches the endpoints from listEndpointsForServiceIdentity
	// i.e. only those interseting endpoints are taken into cosideration
	var allowedEndpoints []endpoint.Endpoint

	allowedEndpoints = append(allowedEndpoints, outboundEndpoints...)

	for _, destSvcIdentity := range destSvcIdentities {
		allowedEndpoints = append(allowedEndpoints, mc.listEndpointsForServiceIdentity(destSvcIdentity)...)
		//for _, ep := range mc.listEndpointsForServiceIdentity(destSvcIdentity) {
		// Is the endpoint IP allowed?
		// TODO(draychev): Why is this needed?
		// if _, ok := outboundEndpointsSet[ep.IP.String()]; ok {
		// allowedEndpoints = append(allowedEndpoints, outboundEndpointsSet[ep.IP.String()]...)
		// }
		//}
	}

	return allowedEndpoints, nil
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) listEndpointsForServiceIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		if ep := provider.ListEndpointsForIdentity(serviceIdentity); len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service account=%s", provider.GetID(), serviceIdentity)
		} else {
			endpoints = append(endpoints, ep...)
		}
	}
	return endpoints
}
