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
		ep := provider.ListEndpointsForService(svc)
		if len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service=%s", provider.GetID(), svc)
			continue
		}
		endpoints = append(endpoints, ep...)
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

// ListAllowedEndpointsForService returns only those endpoints for a service that belong to the allowed outbound service accounts
// for the given downstream identity
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) ListAllowedEndpointsForService(downstreamIdentity identity.ServiceIdentity, upstreamSvc service.MeshService) ([]endpoint.Endpoint, error) {
	outboundEndpoints, err := mc.listEndpointsForService(upstreamSvc)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up endpoints for upstream service %s", upstreamSvc)
		return nil, err
	}

	destSvcAccounts, err := mc.ListAllowedOutboundServiceIdentities(downstreamIdentity)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up outbound service accounts for downstream identity %s", downstreamIdentity)
		return nil, err
	}

	// allowedEndpoints comprises of only those endpoints from outboundEndpoints that matches the endpoints from listEndpointsForServiceIdentity
	// i.e. only those interseting endpoints are taken into cosideration
	var allowedEndpoints []endpoint.Endpoint
	for _, destSvcAccount := range destSvcAccounts {
		podEndpoints := mc.listEndpointsForServiceIdentity(destSvcAccount)
		for _, ep := range outboundEndpoints {
			for _, podIP := range podEndpoints {
				if ep.IP.Equal(podIP.IP) {
					allowedEndpoints = append(allowedEndpoints, ep)
				}
			}
		}
	}
	return allowedEndpoints, nil
}

// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (mc *MeshCatalog) listEndpointsForServiceIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		ep := provider.ListEndpointsForIdentity(serviceIdentity)
		if len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service account=%s", provider.GetID(), serviceIdentity)
			continue
		}
		endpoints = append(endpoints, ep...)
	}
	return endpoints
}
