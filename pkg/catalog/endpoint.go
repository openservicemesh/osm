package catalog

import (
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
)

// ListEndpointsForService returns the list of provider endpoints corresponding to a service
func (mc *MeshCatalog) ListEndpointsForService(svc service.MeshService) ([]endpoint.Endpoint, error) {
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
func (mc *MeshCatalog) ListAllowedEndpointsForService(downstreamIdentity service.K8sServiceAccount, upstreamSvc service.MeshService) ([]endpoint.Endpoint, error) {
	outboundEndpoints, err := mc.ListEndpointsForService(upstreamSvc)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up endpoints for upstream service %s", upstreamSvc)
		return nil, err
	}

	destSvcAccounts, err := mc.ListAllowedOutboundServiceAccounts(downstreamIdentity)
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up outbound service accounts for downstream identity %s", downstreamIdentity)
		return nil, err
	}

	// allowedEndpoints comprises of only those endpoints from outboundEndpoints that matches the endpoints from listEndpointsforIdentity
	// i.e. only those interseting endpoints are taken into cosideration
	var allowedEndpoints []endpoint.Endpoint
	for _, destSvcAccount := range destSvcAccounts {
		podEndpoints := mc.listEndpointsforIdentity(destSvcAccount)
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

// listEndpointsforIdentity retrieves the list of endpoints for the given service account
func (mc *MeshCatalog) listEndpointsforIdentity(sa service.K8sServiceAccount) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint
	for _, provider := range mc.endpointsProviders {
		ep := provider.ListEndpointsForIdentity(sa)
		if len(ep) == 0 {
			log.Trace().Msgf("[%s] No endpoints found for service account=%s", provider.GetID(), sa)
			continue
		}
		endpoints = append(endpoints, ep...)
	}
	return endpoints
}
