package kube

import (
	"net"
	"strconv"
	"strings"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
)

const portIPSeparator = `:`

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
		var endpointsForService []endpoint.Endpoint
		log.Trace().Msgf("Working on service: %s --> spec=%+v", svc, svc.Spec.Clusters)
		for _, cluster := range svc.Spec.Clusters {
			log.Trace().Msgf("Looking for IP and Port for cluster=%s for service %s", cluster.Name, svc)
			if ip, port, err := getIPPort(cluster); err != nil {
				log.Err(err).Msgf("Error getting IP and Port for cluster=%s for service %s", cluster.Name, svc)
			} else {
				endpointsForService = append(endpointsForService, endpoint.Endpoint{
					IP:   ip,
					Port: endpoint.Port(port),
				})
			}
		}
		log.Trace().Msgf("Multicluster endpoints for service %+v: %+v", services, endpointsForService)
		endpoints = append(endpoints, endpointsForService...)
	}

	log.Trace().Msgf("[%s] Multicluster Endpoints for service account %s: %+v", c.providerIdent, serviceAccount, endpoints)
	return endpoints
}

func getIPPort(cluster v1alpha1.ClusterSpec) (ip net.IP, port int, err error) {
	tokens := strings.Split(cluster.Address, portIPSeparator)
	if len(tokens) != 2 {
		log.Error().Msgf("Invalid address format %s. It should have IP address and port number separated by %s", cluster.Address, portIPSeparator)
		return nil, 0, err
	}

	ipStr, portStr := tokens[0], tokens[1]
	port, err = strconv.Atoi(portStr)
	if err != nil {
		log.Error().Msgf("Invalid port number format %s for cluster address: %s", portStr, cluster.Address)
		return nil, 0, err
	}

	ip = net.ParseIP(ipStr)

	return ip, port, err
}
