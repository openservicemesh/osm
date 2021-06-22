package eds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(meshCatalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, _ configurator.Configurator, _ certificate.Manager, _ *registry.ProxyRegistry) ([]types.Resource, error) {
	proxyIdentity, err := envoy.GetServiceAccountFromProxyCertificate(proxy.GetCertificateCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up proxy identity for proxy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}

<<<<<<< HEAD
	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	//log.Debug().Msgf("EDS svc %s allTrafficPolicies %+v", proxyServiceName, allTrafficPolicies)

=======
	allowedEndpoints, err := getEndpointsForProxy(meshCatalog, proxyIdentity.ToServiceIdentity())
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up endpoints for proxy with SerialNumber=%s on Pod with UID=%s", proxy.GetCertificateSerialNumber(), proxy.GetPodUID())
		return nil, err
	}

<<<<<<< HEAD
	outboundServicesEndpoints := make(map[service.MeshServicePort][]endpoint.Endpoint)
	for _, trafficPolicy := range allTrafficPolicies {
		isSourceService := trafficPolicy.Source.Equals(proxyServiceName)
		if isSourceService {
			destService := trafficPolicy.Destination.GetMeshService()
			serviceEndpoints, err := catalog.ListEndpointsForService(destService)
			log.Trace().Msgf("EDS: proxy:%s, serviceEndpoints:%+v", proxyServiceName, serviceEndpoints)
			if err != nil {
				log.Error().Err(err).Msgf("Failed listing endpoints for proxy %s", proxyServiceName)
				return nil, err
			}
			destServicePort := trafficPolicy.Destination
			if destServicePort.Port == 0  {
				outboundServicesEndpoints[destServicePort] = serviceEndpoints
				continue
			}
			// if port specified, filter based on port
			filteredEndpoints := make([]endpoint.Endpoint, 0)
			for _, endpoint := range serviceEndpoints {
				if int(endpoint.Port) != destServicePort.Port {
					continue
				}
				filteredEndpoints = append(filteredEndpoints, endpoint)
			}
			outboundServicesEndpoints[destServicePort] = filteredEndpoints
		}
=======
	var rdsResources []types.Resource
	for svc, endpoints := range allowedEndpoints {
		loadAssignment := newClusterLoadAssignment(svc, endpoints)
		rdsResources = append(rdsResources, loadAssignment)
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
	}

	return rdsResources, nil
}

// getEndpointsForProxy returns only those service endpoints that belong to the allowed outbound service accounts for the proxy
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func getEndpointsForProxy(meshCatalog catalog.MeshCataloger, proxyIdentity identity.ServiceIdentity) (map[service.MeshService][]endpoint.Endpoint, error) {
	allowedServicesEndpoints := make(map[service.MeshService][]endpoint.Endpoint)

<<<<<<< HEAD
	var protos []*any.Any
	for svc, endpoints := range outboundServicesEndpoints {
		if catalog.GetWitesandCataloger().IsWSEdgePodService(svc) {
			loadAssignments := cla.NewWSEdgePodClusterLoadAssignment(catalog, svc)
			for _, loadAssignment := range *loadAssignments {
				proto, err := ptypes.MarshalAny(loadAssignment)
				if err != nil {
					log.Error().Err(err).Msgf("Error marshalling EDS payload for proxy %s: %+v", proxyServiceName, loadAssignment)
					continue
				}
				protos = append(protos, proto)
			}
			continue
		} else if catalog.GetWitesandCataloger().IsWSUnicastService(svc.Name) {
			loadAssignments := cla.NewWSUnicastClusterLoadAssignment(catalog, svc)
			for _, loadAssignment := range *loadAssignments {
				proto, err := ptypes.MarshalAny(loadAssignment)
				if err != nil {
					log.Error().Err(err).Msgf("Error marshalling EDS payload for proxy %s: %+v", proxyServiceName, loadAssignment)
					continue
				}
				protos = append(protos, proto)
			}
			// fall thru for default CLAs
		}
		loadAssignment := cla.NewClusterLoadAssignment(svc, endpoints)
		proto, err := ptypes.MarshalAny(loadAssignment)
=======
	for _, dstSvc := range meshCatalog.ListAllowedOutboundServicesForIdentity(proxyIdentity) {
		endpoints, err := meshCatalog.ListAllowedEndpointsForService(proxyIdentity, dstSvc)
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
		if err != nil {
			log.Error().Err(err).Msgf("Failed listing allowed endpoints for service %s for proxy identity %s", dstSvc, proxyIdentity)
			continue
		}
<<<<<<< HEAD
		protos = append(protos, proto)
	}

	log.Debug().Msgf("EDS url:%s protos: %+v", string(envoy.TypeEDS), protos)
	resp := &xds_discovery.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   string(envoy.TypeEDS),
=======
		allowedServicesEndpoints[dstSvc] = endpoints
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
	}
	log.Trace().Msgf("Allowed outbound service endpoints for proxy with identity %s: %v", proxyIdentity, allowedServicesEndpoints)
	return allowedServicesEndpoints, nil
}
