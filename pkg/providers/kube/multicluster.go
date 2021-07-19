package kube

import (
	"net"
	"strconv"
	"strings"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
)

// getMulticlusterEndpoints returns the endpoints for multicluster services if such exist.
func (c *Client) getMulticlusterEndpoints(svc service.MeshService) []endpoint.Endpoint {
	if !c.meshConfigurator.GetFeatureFlags().EnableMulticlusterMode {
		return nil
	}

	log.Trace().Msgf("[%s] Getting Multicluster Endpoints for service %s", c.providerIdent, svc)

	serviceIdentities, err := c.kubeController.ListServiceIdentitiesForService(svc)
	if err != nil {
		log.Error().Err(err).Msgf("[%s] Error getting Multicluster service identities for service %s", c.providerIdent, svc.Name)
		return nil
	}

	var endpoints []endpoint.Endpoint
	for _, ident := range serviceIdentities {
		remoteEndpoints := c.getMultiClusterServiceEndpointsForServiceAccount(ident.Name, ident.Namespace)
		endpoints = append(endpoints, remoteEndpoints...)
	}

	log.Trace().Msgf("[%s] Multicluster Endpoints for service %s: %+v", c.providerIdent, svc, endpoints)

	return endpoints
}

// getMultiClusterServiceEndpointsForServiceAccount returns the multicluster services for a service account if such exist.
func (c *Client) getMultiClusterServiceEndpointsForServiceAccount(serviceAccount, namespace string) []endpoint.Endpoint {
	if !c.meshConfigurator.GetFeatureFlags().EnableMulticlusterMode {
		return nil
	}

	services := c.configClient.GetMultiClusterServiceByServiceAccount(serviceAccount, namespace)

	if len(services) <= 0 {
		return nil
	}

	var endpoints []endpoint.Endpoint
	for _, svc := range services {
		for _, cluster := range svc.Spec.Clusters {
			tokens := strings.Split(cluster.Address, ":")
			if len(tokens) != 2 {
				log.Error().Msgf("Error parsing remote service %s address %s. It should have IP address and port number", svc.Name, cluster.Address)
				continue
			}

			ip, portStr := tokens[0], tokens[1]
			port, err := strconv.Atoi(portStr)
			if err != nil {
				log.Error().Msgf("Remote service %s port number format invalid. Remote cluster address: %s", svc.Name, cluster.Address)
				continue
			}

			ept := endpoint.Endpoint{
				IP:   net.ParseIP(ip),
				Port: endpoint.Port(port),
			}
			endpoints = append(endpoints, ept)
		}
	}

	log.Trace().Msgf("[%s] Multicluster Endpoints for service account %s: %+v", c.providerIdent, serviceAccount, endpoints)

	return endpoints
}
